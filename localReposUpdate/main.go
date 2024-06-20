package main

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

func findAllGitFolders(targetDir string) []string {
	folders := findAllFolders(targetDir)
	gitFolders := filterGitRepos(folders)
	return gitFolders
}

func findAllFolders(targetDir string) []string {
	files, err := os.ReadDir(targetDir)
	if err != nil {
		log.Fatal(err)
	}
	var folders []string
	for _, file := range files {
		if file.IsDir() {
			folders = append(folders, filepath.Join(targetDir, file.Name()))
		}
	}
	return folders
}

func isGitRepo(path string) bool {
	_, err := os.Stat(path + "/.git")
	return err == nil
}

func filterGitRepos(folders []string) []string {
	var gitRepos []string
	for _, folder := range folders {
		if isGitRepo(folder) {
			gitRepos = append(gitRepos, folder)
		}
	}
	return gitRepos
}

func runCommand(folder string, command string, args ...string) (content string, err error) {
	log.Printf("Running\tfolder: %v, command: %v %v", folder, command, args)
	cmd := exec.Command(command, args...)
	cmd.Dir = folder
	out, err := cmd.Output()
	log.Printf("Command\t%s %s", command, args)
	log.Printf("Output\t%s", out)
	log.Printf("Error\t%v", err)
	content = strings.TrimSpace(string(out))
	if err != nil {
		log.Printf("Error: %v", err)
		return content, err
	}
	return content, nil
}

func runPreCommands(folder string) error {
	preCommands := [][]string{
		{"git", "fetch"},
		{"git", "status"},
		{"git", "remote", "show", "origin"},
	}
	for _, command := range preCommands {
		output, err := runCommand(folder, command[0], command[1:]...)
		if err != nil {
			return err
		}
		log.Printf("Output: %v", output)
	}
	return nil
}

func runOriginUpdate(folder string) error {
	originURL, err := runCommand(folder, "git", "config", "--get", "remote.origin.url")
	if err != nil {
		return err
	}
	newURL := strings.Replace(originURL, "libapps-admin.uncw.edu/randall-dev", "github.com/uncw-library", -1)
	output, err := runCommand(folder, "git", "remote", "set-url", "origin", newURL)
	if err != nil {
		return err
	}
	log.Printf("Output: %v", output)
	return nil
}

func runPostCommands(folder string) error {
	postCommands := [][]string{
		{"git", "branch", "-m", "master", "main"},
		{"git", "fetch", "origin"},
		{"git", "branch", "-u", "origin/main", "main"},
		{"git", "remote", "set-head", "origin", "-a"},
	}
	for _, command := range postCommands {
		output, err := runCommand(folder, command[0], command[1:]...)
		if err != nil {
			return err
		}
		log.Printf("Output: %v", output)
	}
	return nil
}

func main() {
	logFile, err := os.OpenFile(fmt.Sprintf("logs-%v.log", time.Now().Format("20060102_150405")), os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		log.Fatal(err)
	}
	defer logFile.Close()
	log.SetOutput(logFile)

	if len(os.Args) < 2 {
		log.Fatal("Usage: localReposUpdate <targetDir>")
	}
	targetDir := os.Args[1]

	gitFolders := findAllGitFolders(targetDir)
	log.Printf("Git folders: %v", gitFolders)

	successes, erroreds := []string{}, []string{}
	for _, folder := range gitFolders {
		err := runPreCommands(folder)
		if err != nil {
			erroreds = append(erroreds, folder)
			continue
		}
		err = runOriginUpdate(folder)
		if err != nil {
			erroreds = append(erroreds, folder)
			continue
		}
		err = runPostCommands(folder)
		if err != nil {
			erroreds = append(erroreds, folder)
			continue
		}
		successes = append(successes, folder)
	}
	log.Printf("Successes: %v", successes)
	log.Printf("Erroreds: %v", erroreds)
}
