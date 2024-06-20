package main

import (
	"fmt"
	"log"
	"os"
	"path"
	"path/filepath"
	"time"

	"github.com/joho/godotenv"
)

type filechange struct {
	filename    string
	needle      string
	replacement string
}

func doBranch(folder string, branch Branch) error {
	log.Printf("Starting branch\t%v", branch.Name)
	err := checkoutBranch(folder, branch)
	if err != nil {
		log.Printf("Error\t%v", err)
		return err
	}
	filechanges := []filechange{
		{filename: "docker-compose.yml", needle: `image: libapps-admin.uncw.edu:8000/randall-dev/(.*?)`, replacement: "image: uncw-library/%s"},
		{filename: "README.md", needle: `libapps-admin.uncw.edu:8000/randall-dev/(.*?)`, replacement: "uncw-library/%s"},
		{filename: "README.md", needle: `libapps-admin.uncw.edu/randall-dev/(.*?)`, replacement: "github.com/uncw-library/%s"},
	}

	err = gitFetchPull(folder)
	if err != nil {
		return err
	}

	for _, fc := range filechanges {
		fullpath := path.Join(folder, fc.filename)
		err := editFile(fullpath, fc.needle, fc.replacement)
		if err != nil {
			return fmt.Errorf("error editing file %s, %v", fullpath, err)
		}
	}
	err = commitAndPushBranch(folder)
	if err != nil {
		log.Printf("Error\t%v", err)
		return err
	}
	return nil
}

func doFolder(folder string, project Project) error {
	log.Printf("Starting\tfolder: %v", folder)

	err := gitClone(folder, project)
	if err != nil {
		return err
	}

	// do each branch
	log.Printf("%+v", project)
	for _, branch := range project.Branches {
		err = doBranch(folder, branch)
		if err != nil {
			return err
		}
	}

	// // return folder to the default branch
	var defaultBranch Branch
	for _, branch := range project.Branches {
		if branch.Default {
			defaultBranch = branch
		}
	}
	log.Printf("Returning to starting branch: %s\n", defaultBranch.Name)
	err = checkoutBranch(folder, defaultBranch)
	if err != nil {
		return err
	}

	return nil
}

func setup() *os.File {
	// set up logging
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	err := os.Mkdir("logs", 0755)
	if err != nil && !os.IsExist(err) {
		log.Fatalf("Failed to create directory: %v", err)
	}
	logpath := path.Join("logs", fmt.Sprintf("logs-%v.log", time.Now().Format("20060102_150405")))
	logFile, err := os.OpenFile(logpath, os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		log.Fatalf("Failed to open log file: %v", err)
	}
	log.SetOutput(logFile)

	// read .env file
	err = godotenv.Load("../.env")
	if err != nil {
		log.Fatalf("Failed to load .env file: %v", err)
	}
	return logFile
}

func doTheWork(targetDir string) (successes []string, erroreds []string) {
	// do the work
	successes, erroreds = []string{}, []string{}
	libappsProjects, err := fetchLibappsProjects()
	if err != nil {
		log.Fatalf("Error fetching libapps projects: %v", err)
	}

	for _, project := range libappsProjects {
		dest := filepath.Join(targetDir, project.Name)

		// DEBUG:  ONLY DO ONE REPO
		// if dest != "/Users/armstrongg/Desktop/all_gitlab_cloned/d8-staff" {
		// 	continue
		// }

		err = doFolder(dest, project)
		if err != nil {
			erroreds = append(erroreds, dest)
			log.Printf("Error\t%v", err)
			continue
		}
		successes = append(successes, dest)
	}
	return successes, erroreds
}

func main() {
	logFile := setup()
	defer logFile.Close()

	if len(os.Args) < 2 {
		log.Fatal("Usage: repo_sed <targetDir>")
	}
	targetDir := os.Args[1]

	successes, erroreds := doTheWork(targetDir)
	log.Printf("Successes\t%v", successes)
	log.Printf("Erroreds\t%v", erroreds)
}
