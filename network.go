package main

import (
	"fmt"
	"net"
	"os"
	"strings"

	"github.com/codeskyblue/go-sh"
	"log"
)

func InterfaceExists(ipAddress string, interfaceName string) bool {
	intf, err := net.InterfaceByName("lo0")
	if err != nil {
		log.Panic(err)
	}

	addrs, err := intf.Addrs()
	if err != nil {
		log.Panic(err)
	}

	var match bool
	for _, addr := range addrs {
		if strings.Contains(addr.String(), ipAddress) {
			match = true
		}
	}

	return match
}

func MacOSInterfaceExists(ipAddress string) bool {
	return InterfaceExists(ipAddress, "lo0")
}

func LinuxInterfaceExists(ipAddress string) bool {
	return InterfaceExists(ipAddress, "lo:0")
}

func RemoveMacOSInterface(ipAddress string) {
	fmt.Printf("Removing interface with IP: %s\n", ipAddress)

	var cmd = sh.Command("sh", "-c", fmt.Sprintf("sudo ifconfig lo0 -alias %s", ipAddress))
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin

	if err := cmd.Run(); err != nil {
		log.Println("Error removing interface. Try running `sudo ifconfig lo0 -alias 192.168.44.10` manually")
		fmt.Fprintln(os.Stderr, err)
	}
}

func RemoveLinuxInterface(ipAddress string) {
	fmt.Printf("Removing interface with IP: %s\n", ipAddress)

	var cmd = sh.Command("sh", "-c", "sudo ifconfig lo:0 down")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin

	if err := cmd.Run(); err != nil {
		log.Println("Error removing interface. Try running `sudo ifconfig lo0 -alias 192.168.44.10` manually")
		fmt.Fprintln(os.Stderr, err)
	}
}

func CreateMacOSInterface(ipAddress string) {
	fmt.Printf("Creating interface with IP: %s\n", ipAddress)

	var cmd = sh.Command("sh", "-c", fmt.Sprintf("sudo ifconfig lo0 alias %s", ipAddress))
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin

	if err := cmd.Run(); err != nil {
		log.Println("Error creating interface. Try running `sudo ifconfig lo0 alias 192.168.44.10` manually")
		fmt.Fprintln(os.Stderr, err)
	}

	if !MacOSInterfaceExists(ipAddress) {
		log.Fatal("Alias created, but does not seem to be available via loopback interface. Aborting.")
	}
}

func CreateLinuxInterface(ipAddress string) {
	fmt.Printf("Creating interface with IP: %s\n", ipAddress)

	var cmd = sh.Command("sh", "-c", fmt.Sprintf("sudo ifconfig lo:0 %s", ipAddress))
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin

	if err := cmd.Run(); err != nil {
		log.Println("Error creating interface. Try running `sudo ifconfig lo:0 $VIRTUAL_INTERFACE_IP` manually")
		fmt.Fprintln(os.Stderr, err)
	}

	if !LinuxInterfaceExists(ipAddress) {
		log.Fatal("Alias created, but does not seem to be available via loopback interface. Aborting.")
	}
}

func CreateVirtualInterface(ipAddress string) {
	log.Println("Creating or re-using Virtual Interface...")

	if isMacOS() && !MacOSInterfaceExists(ipAddress) {
		CreateMacOSInterface(ipAddress)
	} else if isLinux() && !LinuxInterfaceExists(ipAddress) {
		CreateLinuxInterface(ipAddress)
	} else {
		log.Println("Interface already exists, continuing.")
	}
}

func DestroyVirtualInterface(ipAddress string) {
	log.Println("Destroying Virtual Interface...")

	if isMacOS() && MacOSInterfaceExists(ipAddress) {
		RemoveMacOSInterface(ipAddress)
	} else if isLinux() && LinuxInterfaceExists(ipAddress) {
		RemoveLinuxInterface(ipAddress)
	} else {
		log.Println("Interface does not exist, not removing.")
	}
}
