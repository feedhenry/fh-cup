package main

import (
	"fmt"
	"github.com/BurntSushi/toml"
	"github.com/codeskyblue/go-sh"
	"github.com/fatih/color"
	"github.com/mitchellh/go-homedir"
	"github.com/urfave/cli"
	"io/ioutil"
	"log"
	"os"
	"runtime"
	"strings"
	"syscall"
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

	log.Println("MBaaS and Core now linked.")
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

				// Create PVs - Can be removed post 3.5
				// https://github.com/openshift/origin/pull/12456
				// CreatePVS(conf.FhCupDir)

				log.Println("PVs Created, installing Core...")
				InstallCore(conf)
				log.Println("Installing MBaaS...")
				InstallMBaaS(conf, true)
				log.Println("Linking MBaaS & Core...")
				LinkMBaaSAndCore(conf)

				log.Println("Cluster is now up: https://rhmap.cup.feedhenry.io")
				log.Println("Login with: rhmap-admin@example.com / Password1")

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
