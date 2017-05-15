package main

import (
	"encoding/json"
	"fmt"
	"github.com/samalba/dockerclient"
	"io/ioutil"
	"log"
	"strings"
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
