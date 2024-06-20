package main

import (
	"fmt"
	"log"
	"os"
	"path"
	"time"
)

func fetchAllGitlabProjectInfo() []Project {
	projects, err := fetchLibappsProjects()
	if err != nil {
		log.Fatalf("Failed to fetch projects: %v", err)
	}
	return projects
}

func doWork() {
	// get all image names
	// setup dockerhub repos
	// for each image, get image, tag, push to dockerhub
	projects := fetchAllGitlabProjectInfo()
	fmt.Printf("Fetched %v projects\n", len(projects))
}

func setupLogging() *os.File {
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
	return logFile
}

func setupConfig() {
	err := godotenv.Load()
	if err != nil {
		log.Fatalf("Failed to load .env file: %v", err)
	}
}

func main() {
	logFile := setupLogging()
	defer logFile.Close()
	setupConfig()

	doWork()
}
