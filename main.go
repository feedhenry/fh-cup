package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/BurntSushi/toml"
	"github.com/cenkalti/backoff"
	"github.com/codeskyblue/go-sh"
	"github.com/fatih/color"
	"github.com/hashicorp/go-version"
	"github.com/mitchellh/go-homedir"
	"github.com/samalba/dockerclient"
	"github.com/urfave/cli"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"syscall"
	"time"
)

type Config struct {
	IP                      string
	FhCupDir                string
	CoreProjectName         string
	CoreOpenShiftTemplates  string
	DockerUsername          string
	DockerPassword          string
	DockerEmail             string
	DockerConfigJSONPath    string
	ClusterDomain           string
	MBaaSOpenShiftTemplates string
	MBaaSProjectName        string
	FhcTarget               string
	FhcUsername             string
	FhcPassword             string
}

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

func Cup() {
	color.Set(color.Bold)
	var cup = `
   ( (
    ) )
  ........
  |  fh  |]
  \      /
   '----'
`
	fmt.Println(cup)
	color.Unset()
}

func isMacOS() bool {
	if runtime.GOOS == "darwin" {
		return true
	}
	return false
}

func isLinux() bool {
	if runtime.GOOS == "linux" {
		return true
	}
	return false
}

func hasSELinux() bool {
	if !isLinux() {
		return false
	}

	// TODO: check `getenforce` for 'Enforcing' or 'Permissive'
	if _, err := os.Stat("/usr/bin/chcon"); err == nil {
		return true
	}

	return false
}

func CleanDataDirectories(fhCupDir string) {
	log.Println(fmt.Sprintf("Cleaning: %s/cluster", fhCupDir))

	if fhCupDir == "" {
		log.Println("Error removing cluster files - no path to data dir specified, aborting.")
		os.Exit(1)
	}

	var cmd = sh.Command("sh", "-c", fmt.Sprintf("sudo rm -rf %s/cluster", fhCupDir))
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin

	if err := cmd.Run(); err != nil {
		log.Println("Error removing cluster files.")
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func MakeDataDirectories(fhCupDir string) {
	os.Mkdir(fmt.Sprintf("%s/cluster", fhCupDir), 0777)
	os.Mkdir(fmt.Sprintf("%s/cluster/data", fhCupDir), 0777)
	os.Mkdir(fmt.Sprintf("%s/cluster/config", fhCupDir), 0777)
	os.Mkdir(fmt.Sprintf("%s/cluster/volumes", fhCupDir), 0777)
}

func CreatePVDirectories(fhCupDir string) {
	for i := 0; i < 10; i++ {
		os.Mkdir(fmt.Sprintf("%s/cluster/volumes/devpv%v", fhCupDir, i), 0777)

		if hasSELinux() {
			// Change security context of this folder to prevent permissions errors
			var cmd = sh.Command("sh", "-c", fmt.Sprintf("chcon -R -t svirt_sandbox_file_t %s/cluster/volumes/devpv%v", fhCupDir, i))
			cmd.Stdout = os.Stdout
			cmd.Stderr = os.Stderr
			cmd.Stdin = os.Stdin

			if err := cmd.Run(); err != nil {
				fmt.Println("Error changing security context on PVs")
				fmt.Fprintln(os.Stderr, err)
				os.Exit(1)
			}
		}
	}
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

func UpdateSecurityContextContraints(conf Config) {
	log.Println("Updating Security Context Constraints...")
	oc := new(OpenShiftClient)
	log.Println("Switching to admin...")
	oc.SwitchToAdmin()
	oc.Create(fmt.Sprintf("%s/gitlab-shell/scc-anyuid-with-chroot.json", conf.CoreOpenShiftTemplates))
	log.Println("SCC created, adding policy for project...")

	var args []string = []string{
		"adm",
		"policy",
		"add-scc-to-user",
		"anyuid-with-chroot",
		fmt.Sprintf("system:serviceaccount:%s:default", conf.CoreProjectName)}
	oc.RunOCCommand(args)
	log.Println("Done, switching back to Developer account.")
	oc.SwitchToDeveloper()
}

func CreatePrivateDockerConfig(conf Config) {
	log.Println("Creating private-docker-cfg secret from ~/.docker/config.json ...")

	oc := new(OpenShiftClient)
	oc.RunOCCommand([]string{
		"secrets",
		"new",
		"private-docker-cfg",
		fmt.Sprintf(".dockerconfigjson=%s", conf.DockerConfigJSONPath)})

	oc.RunOCCommand([]string{
		"secrets",
		"link",
		"default",
		"private-docker-cfg",
		"--for=pull"})

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

func CreatePVS(fhCupDir string) {
	// Construct new PVS config from template
	input, err := ioutil.ReadFile(fmt.Sprintf("%s/pvs_template.json", fhCupDir))
	if err != nil {
		log.Fatalln(err)
	}

	// Replace paths with real-paths
	pvConfig := strings.Replace(string(input), "REPLACE_ME", fmt.Sprintf("%s/cluster/volumes", fhCupDir), -1)

	err = ioutil.WriteFile(fmt.Sprintf("%s/pvs.json", fhCupDir), []byte(pvConfig), 0755)
	if err != nil {
		panic(err)
	}

	oc := new(OpenShiftClient)
	log.Println("Switching to admin...")
	oc.SwitchToAdmin()

	// Create the PVs (as an admin)
	oc.Create(fmt.Sprintf("%s/pvs.json", fhCupDir))
	log.Println("PVs created, switching back to Developer...")
	oc.SwitchToDeveloper()

	log.Println("Done.")
}

func Clean(conf Config) {
	CleanDataDirectories(conf.FhCupDir)
	MakeDataDirectories(conf.FhCupDir)
	CreatePVDirectories(conf.FhCupDir)
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
	UpdateSecurityContextContraints(conf)
	CreatePrivateDockerConfig(conf)
	RunInfraSetup(conf)
	RunBackendSetup(conf)
	RunFrontendSetup(conf)
	RunMonitoringSetup(conf)
}

func InstallMBaaS(conf Config) {
	oc := new(OpenShiftClient)
	oc.SwitchToDeveloper()
	oc.CreateProject(conf.MBaaSProjectName)
	CreatePrivateDockerConfig(conf)

	// Setup our MBaaS via shelling out to MBaaS setup scripts
	oc.RunOCCommand([]string{
		"new-app",
		"-f",
		fmt.Sprintf("%s/fh-mbaas-template-1node-persistent.json", conf.MBaaSOpenShiftTemplates)})
	PollForFinishedDeployment(120)
	log.Println("MBaaS setup Done.")
}

// Check oc is installed and the correct version
func CheckOpenShiftClient() {
	oc := new(OpenShiftClient)
	v, err := version.NewVersion(oc.GetOpenShiftClientVersion())
	if err != nil {
		log.Fatal("Could not determine OpenShift Client Version")
	}

	constraints, err := version.NewConstraint(">= 1.3, < 1.4")
	if constraints.Check(v) {
		log.Println(fmt.Sprintf("OK - oc version: %s (required: >= 1.3, < 1.4)", v))
	}
}

func PreFlightChecks() {
	CheckOpenShiftClient()
	// fhc installed
	// fhc version
	// docker installed
	// docker version
	// docker - warn on Mac about Memory/CPU shares
	// docker auth OK
	// TOML Config OK
	// Templates and external files exist/accessible
}

func LinkMBaaSAndCore(conf Config) {
	RunFHCCommand([]string{
		"target",
		conf.FhcTarget,
		conf.FhcUsername,
		conf.FhcPassword})

	oc := new(OpenShiftClient)
	oc.SwitchToDeveloper()
	var MBaaSKey = oc.GetMBaaSkey()
	var openshiftToken = oc.GetUserToken()

	RunFHCCommand([]string{
		"admin",
		"mbaas",
		"create",
		"--id=dev",
		"--url=https://cup.feedhenry.io:8443", //TODO de-hard-code
		fmt.Sprintf("--servicekey=%s", MBaaSKey),
		"--label=dev",
		"--username=test",
		"--password=test",
		"--type=openshift3",
		"--routerDNSUrl=*.cup.feedhenry.io",
		"--fhMbaasHost=https://mbaas-mbaas1.cup.feedhenry.io"})

	RunFHCCommand([]string{
		"admin",
		"environments",
		"create",
		"--id=dev",
		"--label=dev",
		"--target=dev",
		fmt.Sprintf("--token=%s", openshiftToken),
	})

	log.Println("Cluster is now up: https://rhmap.cup.feedhenry.io")
	log.Println("Login with: rhmap-admin@example.com / Password1")
}

func RunFHCCommand(arguments []string) {
	cmd := exec.Command("fhc", arguments...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin

	if err := cmd.Run(); err != nil {
		fmt.Println("Error calling `fhc` command")
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func main() {
	// Set umask for the process - fixes some permissions errors with creating
	// cluster data folders where umask is inherited from the running user
	syscall.Umask(0)
	Cup()

	var conf Config
	home, err := homedir.Dir()
	if err != nil {
		log.Fatal(err)
		os.Exit(1)
		return
	}

	log.Println(fmt.Sprintf("Loading config from: %s/.fh-cup.toml", home))

	if _, err := toml.DecodeFile(fmt.Sprintf("%s/.fh-cup.toml", home), &conf); err != nil {
		log.Fatal(err)
		os.Exit(1)
		return
	}

	app := cli.NewApp()
	app.Name = "fh-cup"
	app.Usage = "Wrapper for `oc cluster up` to install OpenShift and RHMAP"
	app.Version = "1.0.0"

	app.Commands = []cli.Command{
		{
			Name:    "up",
			Aliases: []string{"u"},
			Usage:   "Bring up an OpenShift cluster locally via `oc cluster up`, and install RHMAP",
			Action: func(c *cli.Context) error {
				// Seed images
				if !c.Bool("skip-image-seeding") {
					SeedImages(conf, ListImages(conf))
				} else {
					log.Println("Skipping seeding images.")
				}

				// Cluster status check
				if IsUp(conf.IP) {
					log.Println("Cluster already up, aborting.")
					os.Exit(1)
					return nil
				}

				// Reset cluster if --clean
				if c.Bool("clean") {
					log.Println("Cleaning...")
					Clean(conf)
					log.Println("Done.")
				}

				if c.Bool("no-virtual-interface") {
					log.Println("Skipping Virtual Interface creation.")
					// `oc cluster up`
					OcClusterUp(conf.FhCupDir, conf.IP, conf.ClusterDomain, false)
				} else {
					CreateVirtualInterface(conf.IP)

					// `oc cluster up`
					OcClusterUp(conf.FhCupDir, conf.IP, conf.ClusterDomain, true)

					// Cluster status check
					if !IsUp(conf.IP) {
						log.Println("Cluster has failed to start, aborting.")
						os.Exit(1)
						return nil
					}
					log.Println("Cluster is now up.")
				}

				// Create PVs
				CreatePVS(conf.FhCupDir)

				log.Println("PVs Created, installing Core...")
				InstallCore(conf)
				log.Println("Installing MBaaS...")
				InstallMBaaS(conf)
				log.Println("Linking MBaaS & Core...")
				LinkMBaaSAndCore(conf)

				return nil
			},
			Flags: []cli.Flag{
				cli.BoolFlag{
					Name: "clean", Usage: "Wipe existing state & data directories before bringing cluster up",
				},
				cli.BoolFlag{
					Name: "no-virtual-interface", Usage: "Don't create a virtual interface, bind to whatever interface is up",
				},
				cli.BoolFlag{
					Name: "skip-image-seeding", Usage: "Skip the seeding of images prior to cluster creation",
				},
			},
		},
		{
			Name:    "down",
			Aliases: []string{"d"},
			Usage:   "Tear down an OpenShift cluster via `oc cluster down`",
			Action: func(c *cli.Context) error {
				// Reset cluster if --clean
				if c.Bool("clean") {
					log.Println("Cleaning...")
					Clean(conf)
					log.Println("Done.")
				}

				DestroyVirtualInterface(conf.IP)

				// `oc cluster up`
				OcClusterDown()

				// Cluster status check
				if IsUp(conf.IP) {
					log.Println("Cluster has failed to go down, aborting.")
					os.Exit(1)
					return nil
				}

				log.Println("Cluster is now down.")

				return nil
			},
			Flags: []cli.Flag{
				cli.BoolFlag{
					Name: "clean", Usage: "Wipe existing state & data directories after bringing cluster down",
				},
			},
		},
		{
			Name:    "check",
			Aliases: []string{"c"},
			Usage:   "Check current environment to see if we're good to go.",
			Action: func(c *cli.Context) error {
				log.Println("Checking environment...")
				PreFlightChecks()
				return nil
			},
		},
		{
			Name:    "link",
			Aliases: []string{"c"},
			Usage:   "Link Core & MBaaS via fhc",
			Action: func(c *cli.Context) error {
				log.Println("Linking Core & MBaaS via fhc...")
				LinkMBaaSAndCore(conf)
				return nil
			},
		},
	}

	app.Run(os.Args)
}
