package main

import (
	"fmt"
	"github.com/codeskyblue/go-sh"
	"os"
	"strings"
)

func RunFHCCommand(arguments []string) {
	var args = strings.Join(arguments[:], " ")
	// Use a Dockerised version of fhc
	var cmd = sh.Command("sh", "-c", fmt.Sprintf("docker run -v $HOME:/root -it feedhenry/fhc %s", args))
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin

	if err := cmd.Run(); err != nil {
		fmt.Println("Error calling `fhc` command")
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
