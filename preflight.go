package main

import (
	"fmt"
	"github.com/hashicorp/go-version"
	"log"
)

func PreFlightChecks() {
	CheckOpenShiftClient()
	// docker installed
	// docker version
	// docker - warn on Mac about Memory/CPU shares
	// docker auth OK
	// TOML Config OK
	// Templates and external files exist/accessible
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
