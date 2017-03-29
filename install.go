package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/cenkalti/backoff"
	"github.com/samalba/dockerclient"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"strings"
	"time"
)

type TemplateObject struct {
	Parameters []Parameter `json:"parameters"`
}

// "name": "MEMCACHED_SERVICE_NAME",
// "displayName": "Memcached Service Name",
// "description": "The name of the OpenShift Service exposed for memcached.",
// "value": "memcached",
// "required": true
type Parameter struct {
	Name        string `json:"name"`
	DisplayName string `json:"displayName"`
	Description string `json:"description"`
	Value       string `json:"value"`
	Required    bool   `json:"required"`
}

// PollForFinishedDeployment - returns when either polling times out with an error, or with nil when no pods are in a deploying state
func PollForFinishedDeployment(timeOut int) error {
	operation := func() error {
		var (
			cmdOut []byte
			err    error
		)
		log.Println("Checking Pod deploy status...")

		// TODO: resolve RHMAP-11819
		cmd := exec.Command("/bin/sh", "-c", "oc get pods | { grep -v ups || true; } | { grep \"deploy\" || true; }")
		if cmdOut, err = cmd.Output(); err != nil {
			fmt.Fprintln(os.Stderr, "Error checking for finished Pod deployment: ", err)
		}
		log.Println(string(cmdOut))

		if len(cmdOut) == 0 {
			// Done
			return nil
		}

		// In progress
		return errors.New("Waiting for pods to be ready.")
	}

	bo := backoff.NewExponentialBackOff()
	bo.MaxElapsedTime = time.Duration(timeOut) * time.Second
	bo.Multiplier = 1.1
	err := backoff.Retry(operation, bo)
	if err != nil {
		log.Println("An error occured when waiting for pod deployment completion - aborting.")
		os.Exit(-1)
	}

	log.Println("No pending deployments, done.")
	return nil
}

func RunPreRequisites(conf Config) {
	log.Println("Running prerequisites.sh...")
	env := os.Environ()
	env = append(env, fmt.Sprintf("CLUSTER_DOMAIN=%s", conf.ClusterDomain))
	cmd := exec.Command("/bin/bash", fmt.Sprintf("%s/scripts/core/prerequisites.sh", conf.CoreOpenShiftTemplates))
	cmd.Env = env

	// Redirect stdout/stderr/stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin

	if err := cmd.Run(); err != nil {
		fmt.Println("Error calling `prerequisites.sh`")
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	log.Println("Done.")
}

func RunSetupScript(timeOut int, scriptName string, conf Config) {
	log.Println(fmt.Sprintf("Running %s...", scriptName))
	env := os.Environ()
	env = append(env, fmt.Sprintf("CLUSTER_DOMAIN=%s", conf.ClusterDomain))
	cmd := exec.Command("/bin/bash", fmt.Sprintf("%s/scripts/core/%s", conf.CoreOpenShiftTemplates, scriptName))
	cmd.Env = env

	// Redirect stdout/stderr/stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin

	if err := cmd.Run(); err != nil {
		fmt.Println(fmt.Sprintf("Error calling %s", scriptName))
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	PollForFinishedDeployment(timeOut)

	log.Println(fmt.Sprintf("%s - complete.", scriptName))
}

func RunInfraSetup(conf Config) {
	RunSetupScript(120, "infra.sh", conf)
}

func RunBackendSetup(conf Config) {
	RunSetupScript(120, "backend.sh", conf)
}

func RunFrontendSetup(conf Config) {
	RunSetupScript(120, "frontend.sh", conf)
}

func RunMonitoringSetup(conf Config) {
	RunSetupScript(60, "monitoring.sh", conf)
}

func ReadImages(templateJsonPath string) (images []string) {
	jsonBody, err := ioutil.ReadFile(templateJsonPath)
	if err != nil {
		log.Fatalln(err)
	}

	var infraTemplate TemplateObject
	json.Unmarshal(jsonBody, &infraTemplate)

	var tmpList []string // in the format {name, tag, name, tag} etc.

	for _, v := range infraTemplate.Parameters {
		if strings.HasSuffix(v.Name, "_IMAGE") {
			tmpList = append(tmpList, v.Value)
		}

		if strings.HasSuffix(v.Name, "_IMAGE_VERSION") {
			tmpList = append(tmpList, v.Value)
		}
	}

	// Turn {name, tag, name, tag}
	// Into {"name:tag", "name:tag"}
	var imageList []string
	for i := 0; i < len(tmpList); i += 2 {
		imageList = append(imageList, strings.Join(tmpList[i:i+2], ":"))
	}

	return imageList
}

func ListImages(conf Config) (images []string) {
	var allImages []string

	var infraImages = ReadImages(fmt.Sprintf("%s/generated/fh-core-infra.json", conf.CoreOpenShiftTemplates))
	var backendImages = ReadImages(fmt.Sprintf("%s/generated/fh-core-backend.json", conf.CoreOpenShiftTemplates))
	var frontendImages = ReadImages(fmt.Sprintf("%s/generated/fh-core-frontend.json", conf.CoreOpenShiftTemplates))
	var monitoringImages = ReadImages(fmt.Sprintf("%s/generated/fh-core-monitoring.json", conf.CoreOpenShiftTemplates))

	allImages = append(infraImages, backendImages...)
	allImages = append(allImages, backendImages...)
	allImages = append(allImages, frontendImages...)
	allImages = append(allImages, monitoringImages...)

	return allImages
}

func ImageExists(imageName string) bool {
	docker, _ := dockerclient.NewDockerClient("unix:///var/run/docker.sock", nil)
	allImages, errListImages := docker.ListImages(true)
	if errListImages != nil {
		log.Fatal("error listing all images: %s", errListImages)
	}

	var imageExists = false
	for _, image := range allImages {
		if len(image.RepoTags) > 0 {
			if strings.Contains(imageName, image.RepoTags[0]) {
				imageExists = true
			}
		}
	}

	return imageExists
}

func SeedImages(conf Config, images []string) {
	// Init the client
	docker, _ := dockerclient.NewDockerClient("unix:///var/run/docker.sock", nil)
	a := &dockerclient.AuthConfig{Username: conf.DockerUsername, Password: conf.DockerPassword, Email: conf.DockerEmail}

	// TODO: skip pull if available in local images
	log.Println("Seeding inital Docker Images...")
	for _, imageName := range images {
		if ImageExists(imageName) {
			log.Println(fmt.Sprintf("Image already pulled, skipping: %s", imageName))
		} else {
			log.Println(fmt.Sprintf("Pulling image: %s", imageName))
			err := docker.PullImage(imageName, a)

			if err != nil {
				log.Fatal("error pulling image: %s", err)
			}
		}

	}
	log.Println("Done.")
}

func InstallCore(conf Config) {
	oc := new(OpenShiftClient)
	oc.CreateProject(conf.CoreProjectName)

	// Shell-out and run our core setup scripts
	RunPreRequisites(conf)
	CreatePrivateDockerConfig(conf)
	RunInfraSetup(conf)
	RunBackendSetup(conf)
	RunFrontendSetup(conf)
	RunMonitoringSetup(conf)
}

func InstallMBaaS(conf Config, switchUser bool) {
	oc := new(OpenShiftClient)
	if switchUser {
		oc.SwitchToDeveloper()
	}
	oc.CreateProject(conf.MBaaSProjectName)
	CreatePrivateDockerConfig(conf)

	// Setup our MBaaS via shelling out to MBaaS setup scripts
	oc.RunOCCommand([]string{
		"new-app",
		"-f",
		fmt.Sprintf("%s/fh-mbaas-template-1node.json", conf.MBaaSOpenShiftTemplates)})
	PollForFinishedDeployment(120)
	log.Println("MBaaS setup Done.")
}
