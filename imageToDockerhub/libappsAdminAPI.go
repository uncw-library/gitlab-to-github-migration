package main

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
)

type Image struct {
	Name string `json:"name"`
	Tags []string
}

type Branch struct {
	Name    string `json:"name"`
	Default bool   `json:"default"`
}

type Links struct {
	RepoBranches string `json:"repo_branches"`
}

type Token struct {
	Token string `json:"token"`
}

type Project struct {
	ID                int    `json:"id"`
	Name              string `json:"name"`
	URL               string `json:"http_url_to_repo"`
	Archived          bool   `json:"archived"`
	Visibility        string `json:"visibility"`
	PathWithNamespace string `json:"path_with_namespace"`
	Links             Links  `json:"_links"`
	Branches          []Branch
	Images            []Image
}

func fetchLibappsProjects() ([]Project, error) {
	page := 1
	projects := []Project{}
	for {
		pageProjects, err := fetchLibappsPage(page)
		if err != nil {
			return nil, err
		}
		if len(pageProjects) == 0 {
			break
		}
		projects = append(projects, pageProjects...)
		page++
	}

	for i := range projects {
		log.Printf("Enriching %s", projects[i].Name)
		err := enrichBranches(&projects[i])
		if err != nil {
			log.Printf("Failed to enrich branch: %v", err)
			continue
		}
		token, err := getDockerRegistryToken(projects[i])
		if err != nil {
			log.Printf("Failed to get docker registry token: %v", err)
			continue
		}
		err = enrichImages(&projects[i], token)
		if err != nil {
			return projects, err
		}
	}

	writeProjectsToFile(projects, "libapps-admin_projects.json")

	return projects, nil
}

func fetchLibappsPage(page int) ([]Project, error) {
	url := fmt.Sprintf("https://libapps-admin.uncw.edu/api/v4/projects?page=%d&per_page=100", page)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	privateToken := os.Getenv("LIBAPPS_ADMIN_TOKEN")
	req.Header.Set("PRIVATE-TOKEN", privateToken)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var projects []Project
	if err := json.Unmarshal(body, &projects); err != nil {
		return nil, err
	}

	return projects, nil
}

func writeProjectsToFile(projects []Project, filename string) error {
	projectsFile, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer projectsFile.Close()
	encoder := json.NewEncoder(projectsFile)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(projects); err != nil {
		return err
	}
	return nil
}

func enrichBranches(project *Project) error {
	req, err := http.NewRequest("GET", project.Links.RepoBranches, nil)
	if err != nil {
		return err
	}
	privateToken := os.Getenv("LIBAPPS_ADMIN_TOKEN")
	req.Header.Set("PRIVATE-TOKEN", privateToken)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		log.Printf("Failed to fetch branches for project %s: %v", project.Name, err)
		return err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	if err := json.Unmarshal(body, &project.Branches); err != nil {
		return err
	}

	return nil
}

func enrichImages(project *Project, token Token) error {
	client := &http.Client{}

	url := fmt.Sprintf("http://localhost:5000/v2/%s/tags/list", project.PathWithNamespace)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		log.Fatal(err)
	}

	req.Header.Add("Accept", "application/vnd.docker.distribution.manifest.v2+json")
	req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", token.Token))

	resp, err := client.Do(req)
	if err != nil {
		log.Fatal(err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	log.Printf("body: %s", body)
	if err != nil {
		log.Printf("Failed to read body: %v", err)
		log.Fatal(err)
	}

	var result map[string]interface{}
	json.Unmarshal(body, &result)

	if errorsInterface, ok := result["errors"]; ok {
		// Handle errors
		for _, errorInterface := range errorsInterface.([]interface{}) {
			errorMap := errorInterface.(map[string]interface{})
			if errorMap["code"] == "NAME_UNKNOWN" {
				log.Printf("No image found for project %s", project.Name)
				return nil
			}
		}
		log.Printf("errors: %v", errorsInterface)
		return nil
	}

	if tagsInterface, ok := result["tags"]; ok {
		// Handle tags
		if tagsInterface == nil {
			project.Images = append(project.Images, Image{Name: project.Name, Tags: []string{""}})
			return nil
		}
		tagsSlice := tagsInterface.([]interface{})
		tags := make([]string, len(tagsSlice))
		for i, tag := range tagsSlice {
			tags[i] = tag.(string)
		}
		image := Image{
			Name: result["name"].(string),
			Tags: tags,
		}
		log.Printf("tags %v", tags)
		project.Images = append(project.Images, image)
	}
	return nil
}

func getDockerRegistryToken(project Project) (Token, error) {
	var token Token
	url := fmt.Sprintf("https://libapps-admin.uncw.edu/jwt/auth?client_id=docker&offline_token=true&service=container_registry&scope=repository:%s:push,pull", project.PathWithNamespace)

	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	client := &http.Client{Transport: tr}
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		log.Fatal(err)
	}
	req.SetBasicAuth(os.Getenv("GITLAB_USER"), os.Getenv("GITLAB_PASS"))

	resp, err := client.Do(req)
	if err != nil {
		log.Fatal(err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Fatal(err)
	}
	err = json.Unmarshal(body, &token)
	if err != nil {
		log.Fatal(err)
	}
	return token, nil
}
