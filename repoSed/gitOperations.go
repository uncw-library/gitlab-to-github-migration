package main

import (
	"fmt"
	"log"
	"os/exec"
	"strings"
)

func gitClone(folder string, project Project) error {
	log.Printf("Cloning %s", project.Name)
	cmd := exec.Command("git", "clone", project.URL, folder)
	output, err := cmd.CombinedOutput()
	if strings.Contains(string(output), "already exists and is not an empty directory") {
		log.Printf("Not pulling %s because it already exists\n", project.Name)
		return nil
	}
	if err != nil {
		log.Printf("Failed to clone repository: %s", project.URL)
		log.Printf("%s", output)
		return fmt.Errorf("error cloning repository %s", project.URL)
	}
	return nil
}

func gitFetchPull(folder string) error {
	log.Printf("Fetching and pulling in %s", folder)
	_, err := runCommand(folder, "git", "fetch", "--all")
	if err != nil {
		log.Printf("Error\t%v", err)
		return fmt.Errorf("error fetching in folder %s", folder)
	}
	_, err = runCommand(folder, "git", "pull", "--all")
	if err != nil {
		log.Printf("Error\t%v", err)
		return fmt.Errorf("error pulling in folder %s", folder)
	}
	return nil
}

func checkoutBranch(folder string, branch Branch) error {
	log.Printf("branch is: %v", branch)
	_, err := runCommand(folder, "git", "checkout", branch.Name)
	if err != nil {
		log.Printf("Error\t%v", err)
		return fmt.Errorf("error checking out branch %s in folder %s", branch.Name, folder)
	}
	return nil
}

func commitAndPushBranch(folder string) error {
	_, err := runCommand(folder, "git", "add", ".")
	if err != nil {
		log.Printf("Error\t%v", err)
		return fmt.Errorf("error adding files in folder %s", folder)
	}
	output, err := runCommand(folder, "git", "commit", "-m", `"Updating git & image references"`)
	if strings.Contains(output, "nothing to commit") {
		log.Printf("Info\tNothing to commit in folder %s", folder)
		return nil
	}
	if err != nil {
		log.Printf("Error\t%v", err)
		return fmt.Errorf("error committing changes in folder %s", folder)
	}
	// _, err = runCommand(folder, "git", "push")
	// if err != nil {
	// 	log.Printf("Error\t%v", err)
	// 	return fmt.Errorf("error pushing changes in folder %s", folder)
	// }
	return nil
}
