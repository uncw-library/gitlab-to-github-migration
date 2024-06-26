package main

import (
	"fmt"
	"log"
	"os"
	"path"
	"time"

	"github.com/joho/godotenv"
)

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
	err := godotenv.Load("../.env")
	if err != nil {
		log.Fatalf("Failed to load .env file: %v", err)
	}
}

// func devHack(file string) ([]Project, error) {
// 	data, err := ioutil.ReadFile(file)
// 	if err != nil {
// 		return nil, fmt.Errorf("failed to read file: %v", err)
// 	}
// 	var projects []Project
// 	if err := json.Unmarshal(data, &projects); err != nil {
// 		return nil, fmt.Errorf("failed to unmarshal JSON: %v", err)
// 	}
// 	return projects, nil
// }

func main() {
	logFile := setupLogging()
	defer logFile.Close()
	setupConfig()

	projects, err := fetchLibappsProjects()
	if err != nil {
		log.Fatalf("Failed to fetch projects: %v", err)
	}

	// debug: skip fetching projects
	// projects, err := devHack("manualProjects.json")
	// if err != nil {
	// 	log.Fatalf("Failed to fetch projects: %v", err)
	// }

	if err := migrateImages(projects); err != nil {
		log.Fatalf("Failed to migrate repos: %v", err)
	}
	log.Print("Done")
}
