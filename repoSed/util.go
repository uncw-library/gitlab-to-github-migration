package main

import (
	"log"
	"os/exec"
	"strings"
)

// runCommand executes a command in a specified folder and returns the output and any error encountered.
func runCommand(folder string, command string, args ...string) (content string, err error) {
	log.Printf("Running\tfolder: %v, command: %v %v", folder, command, args)
	cmd := exec.Command(command, args...)
	cmd.Dir = folder
	out, err := cmd.Output()
	log.Printf("Command\t%s %s", command, args)
	content = strings.TrimSpace(string(out))
	if err != nil {
		return content, err
	}
	return content, nil
}
