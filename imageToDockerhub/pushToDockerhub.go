package main

// Create repo info from:  https://stackoverflow.com/questions/29844420/how-can-i-create-docker-private-repository-via-docker-hub-api
// Dockerhub token must be Read/Write/Delete

// But `docker push` creates the image namespace in dockerhub, so we don't need to create it first

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"strings"
)

type DockerAuthRequest struct {
	Username string `json:"username"`
	Token    string `json:"password"`
}

type DockerAuthResponse struct {
	Token string `json:"token"`
}

type NewRepoReq struct {
	Name        string `json:"name"`
	Namespace   string `json:"namespace"`
	Description string `json:"description"`
	IsPrivate   bool   `json:"is_private"`
}

func getToken() (token string, err error) {
	authURL := "https://hub.docker.com/v2/users/login/"
	authReq := DockerAuthRequest{
		Username: os.Getenv("DOCKERHUB_USER"),
		Token:    os.Getenv("DOCKERHUB_TOKEN"),
	}

	jsonData, err := json.Marshal(authReq)
	if err != nil {
		return "", fmt.Errorf("error marshaling JSON: %w", err)
	}
	req, err := http.NewRequest("POST", authURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return "", fmt.Errorf("error creating request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("error sending request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("error reading response body: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("authentication failed, status code: %d", resp.StatusCode)
	}

	var authResp DockerAuthResponse
	if err := json.Unmarshal(body, &authResp); err != nil {
		return "", fmt.Errorf("error unmarshaling response: %w", err)
	}

	return authResp.Token, nil
}

func createDockerhubRepo(repo NewRepoReq, token string) error {
	apiURL := "https://hub.docker.com/v2/repositories/"

	// Marshal the repository data into JSON
	jsonData, err := json.Marshal(repo)
	if err != nil {
		fmt.Println("Error marshaling JSON:", err)
		return err
	}

	// Create a new request
	req, err := http.NewRequest("POST", apiURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("error creating request: %s", err)
	}

	// Set headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("JWT %s", token))

	// Send the request
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("error sending request: %s", err)
	}
	defer resp.Body.Close()

	// Read the response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("error reading response body: %s", err)
	}
	var result map[string]interface{}
	json.Unmarshal(body, &result)

	// Check the response
	if resp.StatusCode != http.StatusCreated {
		if strings.Contains(result["message"].(string), "already exists") {
			log.Printf("Repository already exists: %s", repo.Name)
			return nil
		} else {
			return fmt.Errorf("failed to create repository, status code: %d %s", resp.StatusCode, body)
		}
	}
	log.Printf("Repository created: %s", repo.Name)
	return nil
}

func migrateImage(oldImageName string, newImageName string, tag string) error {
	if tag == "" {
		tag = "latest"
	}
	oldFull := fmt.Sprintf("libapps-admin.uncw.edu:8000/randall-dev/%s:%s", oldImageName, tag)
	newFull := fmt.Sprintf("%s/%s:%s", os.Getenv("DOCKERHUB_ORG"), newImageName, tag)
	// log.Printf("oldFull: %s", oldFull)
	// log.Printf("newFull: %s", newFull)
	if err := pullDockerImage(oldFull); err != nil {
		return err
	}
	if err := renameImage(oldFull, newFull); err != nil {
		return err
	}
	if err := loginToRegistry(); err != nil {
		return err
	}
	if err := pushImage(newFull); err != nil {
		return err
	}
	return nil
}

func pullDockerImage(imageName string) error {
	log.Printf("Pulling image %s\n", imageName)
	cmd := exec.Command("docker", "pull", imageName)
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to pull Docker image: %s, output: %s", err, output)
	}
	log.Printf("Pulled image %s\n", imageName)
	return nil
}

func renameImage(oldImage string, newImage string) error {
	cmd := exec.Command("docker", "tag", oldImage, newImage)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to tag image %s as %s: %w", oldImage, newImage, err)
	}
	log.Printf("Renamed %s to %s", oldImage, newImage)
	return nil
}

func pushImage(newImage string) error {
	cmd := exec.Command("docker", "push", newImage)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to push Docker image to registry: %w", err)
	}
	log.Printf("Pushed image %s", newImage)
	return nil
}

func loginToRegistry() error {
	username, token := os.Getenv("DOCKERHUB_USER"), os.Getenv("DOCKERHUB_TOKEN")
	cmd := exec.Command("docker", "login", "--username", username, "--password", token)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to login to Docker registry: %w", err)
	}
	log.Printf("Logged in to Docker registry as %s\n", username)
	return nil
}

func migrateImages(projects []Project) error {
	successed, faileds := []string{}, []string{}
	token, err := getToken()
	if err != nil {
		return fmt.Errorf("failed to get token: %v", err)
	}
	for _, project := range projects {
		for _, image := range project.Images {
			oldImageName := image.Name
			newImageName := strings.Replace(oldImageName, "/", "-", -1)
			newRepo := NewRepoReq{
				Name:        newImageName,
				Namespace:   os.Getenv("DOCKERHUB_ORG"),
				Description: "",
				IsPrivate:   true,
			}
			if err := createDockerhubRepo(newRepo, token); err != nil {
				log.Printf("Failed to create repo: %v", err)
				faileds = append(faileds, project.Name)
				continue
			}
			for tag, ok := range image.Tags {
				if !ok {
					log.Printf("tag %s not found in image %s", tag, image.Name)
					continue
				}
				if err := migrateImage(oldImageName, newImageName, tag); err != nil {
					log.Printf("Failed to migrate image: %v", err)
					faileds = append(faileds, image.Name)
					continue
				}
			}
			log.Printf("Migrated image: %s", oldImageName)
			successed = append(successed, image.Name)
		}
	}
	log.Printf("Successed migrate images: %v", successed)
	log.Printf("Failed to migrate images: %v", faileds)
	return nil
}
