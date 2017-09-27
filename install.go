package main

import (
	"fmt"
	"github.com/codeskyblue/go-sh"
	"log"
	"os"
)

func InstallRHMAP(conf Config) {
	log.Println("Running rhmap-ansible installer...")

	var cmd = sh.Command("sh", "-c", fmt.Sprintf("docker run "+
		" -v %s/generated:/opt/rhmap/templates/core"+
		" -v %s:/opt/rhmap/templates/mbaas"+
		" -e PLAYBOOK_FILE=/opt/app-root/src/playbooks/poc.yml"+
		" -e INVENTORY_FILE=/opt/app-root/src/inventory-templates/fh-cup-example"+
		" -e OPTS=\"-e core_templates_dir=/opt/rhmap/templates/core -e mbaas_templates_dir=/opt/rhmap/templates/mbaas"+
		" -e mbaas_project_name=mbaas -e core_project_name=core -e strict_mode=false --tags deploy\""+
		" %s",
		conf.CoreOpenShiftTemplates, conf.MBaaSOpenShiftTemplates, conf.RhmapAnsibleImage))
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin

	if err := cmd.Run(); err != nil {
		fmt.Println("Error calling `rhmap-ansible` installer")
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
