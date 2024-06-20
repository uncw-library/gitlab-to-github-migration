package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
)

type Image struct {
	Name string `json:"name"`
}

type Branch struct {
	Name    string `json:"name"`
	Default bool   `json:"default"`
}

type Links struct {
	RepoBranches string `json:"repo_branches"`
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
		enrichBranches(&projects[i])
		enrichImages(&projects[i])
	}

	writeProjectsToFile(projects, "libapps-admin_projects.json")

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

func enrichImages(project *Project) error {
	url := fmt.Sprintf("https://libapps-admin.uncw.edu/api/v4/projects/%s/registry/repositories", project.ID)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return err
	}
	privateToken := os.Getenv("LIBAPPS_ADMIN_TOKEN")
	req.Header.Add("PRIVATE-TOKEN", privateToken)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	var images []Image
	err = json.Unmarshal(body, &images)
	if err != nil {
		return err
	}
	return nil
}
