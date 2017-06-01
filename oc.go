package main

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"strings"
	"time"
)

type OpenShiftClient struct{}

func RunWithTimeout(command *exec.Cmd) error {
	timer := time.AfterFunc(3*time.Second, func() {
		fmt.Fprintln(os.Stderr, "Command timed out")
		// TODO: callback with an error?
		command.Process.Kill()
	})

	err := command.Wait()
	if err != nil {
		fmt.Fprintln(os.Stderr, "Error: ", err)
		return err
	}
	timer.Stop()

	return err
}

func (oc OpenShiftClient) SwitchUser(user string, password string) {
	// Login as developer User with shelled oc
	fmt.Printf("Switching user to %s...\n", user)
	cmd := exec.Command("oc", "login", "-u", user, "-p", password)
	if err := cmd.Start(); err != nil {
		log.Fatal(err)
		os.Exit(1)
	}

	err := RunWithTimeout(cmd)
	if err != nil {
		log.Fatal(err)
		return
	}

	fmt.Println("Done.")
}

func (oc OpenShiftClient) SwitchToDeveloper() {
	oc.SwitchUser("developer", "developer")
}

func (oc OpenShiftClient) SwitchToAdmin() {
	oc.SwitchUser("system:admin", "")
}

func (oc OpenShiftClient) GetMBaaSkey() string {
	// `oc env dc/fh-mbaas --list -n mbaas1 | grep FHMBAAS_KEY`
	var (
		cmdOut []byte
		err    error
	)

	cmd := exec.Command("/bin/sh", "-c", "oc env dc/fh-mbaas --list -n mbaas1 | { grep FHMBAAS_KEY || true; } | cut -d '=' -f2")
	if cmdOut, err = cmd.Output(); err != nil {
		fmt.Fprintln(os.Stderr, "Error: ", err)
	}
	log.Println(string(cmdOut))
	return strings.TrimSpace(string(cmdOut))
}

func (oc OpenShiftClient) GetUserToken() string {
	// `oc whoami -t`
	var (
		cmdOut []byte
		err    error
	)

	cmd := exec.Command("/bin/sh", "-c", "oc whoami -t")
	if cmdOut, err = cmd.Output(); err != nil {
		fmt.Fprintln(os.Stderr, "Error ", err)
	}
	log.Println(string(cmdOut))
	return strings.TrimSpace(string(cmdOut))
}

func (oc OpenShiftClient) GetOpenShiftClientVersion() string {
	// `oc version | head -n 1 | cut -d "v" -f 2`
	var (
		cmdOut []byte
		err    error
	)

	cmd := exec.Command("/bin/sh", "-c", "oc version | head -n 1 | cut -d \"v\" -f 2")
	if cmdOut, err = cmd.Output(); err != nil {
		fmt.Fprintln(os.Stderr, "Error ", err)
	}
	return strings.TrimSpace(string(cmdOut))
}

func (oc OpenShiftClient) RunOCCommand(arguments []string) {
	cmd := exec.Command("oc", arguments...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin

	if err := cmd.Run(); err != nil {
		fmt.Println("Error calling `oc` command")
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func (oc OpenShiftClient) Create(path string) {
	log.Println("Creating via OC")
	cmd := exec.Command("oc", "create", "-f", path)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin
	if err := cmd.Start(); err != nil {
		log.Println(err)
	}

	err := RunWithTimeout(cmd)
	if err != nil {
		log.Println(err)
	}

	fmt.Println("Done.")
}

func (oc OpenShiftClient) CreateProject(projectName string) {
	log.Println("Creating via OC")
	cmd := exec.Command("oc", "new-project", projectName)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin
	if err := cmd.Start(); err != nil {
		log.Println(err)
	}

	err := RunWithTimeout(cmd)
	if err != nil {
		log.Println(err)
	}

	fmt.Println("Done.")
}

func IsUp(ipAddress string) bool {
	// TODO - also check for `socat`` processes?
	var (
		cmdOut []byte
		err    error
	)

	cmd := exec.Command("/bin/sh", "-c", "docker ps | grep openshift | awk '{print $1}'")
	cmdOut, err = cmd.Output()

	if err != nil {
		fmt.Fprintln(os.Stderr, "Error ", err)
	}

	var trimmedOut = strings.TrimSpace(string(cmdOut))

	if trimmedOut != "" {
		log.Println(fmt.Sprintf("OpenShift running: %s", trimmedOut))
		return true
	}

	log.Println("OpenShift is not running.")
	return false
}

func OcClusterUp(cupDir string, ipAddress string, routingSuffix string, bindToAlternativePort bool) {
	var args []string

	if bindToAlternativePort {
		args = []string{
			"cluster",
			"up",
			fmt.Sprintf("--host-data-dir=%s/cluster/data", cupDir),
			fmt.Sprintf("--host-config-dir=%s/cluster/config", cupDir),
			fmt.Sprintf("--public-hostname=%s", ipAddress),
			fmt.Sprintf("--routing-suffix=%s", routingSuffix)}
	} else {
		args = []string{
			"cluster",
			"up",
			fmt.Sprintf("--host-data-dir=%s/cluster/data", cupDir),
			fmt.Sprintf("--host-config-dir=%s/cluster/config", cupDir)}
	}

	cmd := exec.Command("oc", args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin

	if err := cmd.Run(); err != nil {
		fmt.Println("Error calling `oc cluster up`")
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func OcClusterDown() {
	cmd := exec.Command(
		"oc",
		"cluster",
		"down")

	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin

	if err := cmd.Run(); err != nil {
		fmt.Println("Error creating interface")
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
