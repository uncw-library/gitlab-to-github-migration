package main

import (
	"bufio"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
)

type Image struct {
	Name string          `json:"name"`
	Tags map[string]bool `json:"tags"`
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
	ID                int      `json:"id"`
	Name              string   `json:"name"`
	URL               string   `json:"http_url_to_repo"`
	Archived          bool     `json:"archived"`
	Visibility        string   `json:"visibility"`
	PathWithNamespace string   `json:"path_with_namespace"`
	Links             Links    `json:"_links"`
	Branches          []Branch `json:"branches"`
	Images            []Image  `json:"images"`
}

func fetchLibappsProjects() ([]Project, error) {
	imagesFromGrep := getUniqueFromGreppedImages()

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
		// if projects[i].Name != "wildcard-proxy" {
		// 	log.Print("Skipping all but wildcard-proxy")
		// 	continue
		// }

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
		err = enrichImages(&projects[i], token, imagesFromGrep)
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

func enrichImages(project *Project, token Token, imagesFromGrep []Image) error {
	// if project.Name != "wildcard-proxy" {
	// 	log.Print("Skipping all but wildcard-proxy")
	// 	return nil
	// }

	log.Printf("Enriching images for %s", project.Name)
	// preload the image names found on the server into the project
	for _, v := range imagesFromGrep {
		grepProjectName := strings.Split(v.Name, "/")[0]
		if grepProjectName == project.Name {
			project.Images = append(project.Images, v)
		}
	}
	for i, pImage := range project.Images {
		// log.Printf("Looping through images: %s", pImage.Name)
		for _, gImage := range imagesFromGrep {
			// log.Printf("Looping through grepped images: %s", gImage.Name)
			grepProjectName := strings.Split(gImage.Name, "/")[0]
			// log.Printf("grepProjectName: %s", grepProjectName)
			if pImage.Name == grepProjectName {
				// log.Printf("Found matching image: %s %s", pImage.Name, gImage.Name)
				for gTag := range gImage.Tags {
					project.Images[i].Tags[gTag] = true
				}
			}
		}
	}

	// merge in the results from the docker API
	result := getImageFromDockerAPI(project, token)
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
		var resultTags []string
		if tagsInterface == nil {
			log.Printf("nil tagsInsterface for project %s", project.Name)
			return nil
		} else {
			resultTagsSlice := tagsInterface.([]interface{})
			for _, tag := range resultTagsSlice {
				tagStr, ok := tag.(string)
				if !ok {
					continue
				}
				resultTags = append(resultTags, tagStr)
			}
		}
		// try to find an existing image name, then add the tags
		for i, pImage := range project.Images {
			if pImage.Name == project.Name {
				for _, tag := range resultTags {
					project.Images[i].Tags[tag] = true
				}
				return nil
			}
		}

		// if no existing image name, then make a new image + tags & attach it.
		newTags := map[string]bool{}
		for _, tag := range resultTags {
			newTags[tag] = true
		}
		image := Image{
			Name: project.Name,
			Tags: newTags,
		}
		project.Images = append(project.Images, image)
	}

	return nil
}

func getImageFromDockerAPI(project *Project, token Token) map[string]interface{} {
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
	return result
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

func getUniqueFromGreppedImages() []Image {
	// log.Printf("Getting unique images from grepped images")
	images := []Image{}
	projectsFile, err := os.Open("grepped_docker_images.txt")
	if err != nil {
		log.Fatal("Need file 'grepped_docker_images.txt' in appdir with names of all images currently running")
	}
	defer projectsFile.Close()
	// only add unique image to greppedImages
	scanner := bufio.NewScanner(projectsFile)
	for scanner.Scan() {
		line := scanner.Text()

		// if !strings.Contains(line, "wildcard-proxy") {
		// 	log.Print("skipping line without wildcard-proxy")
		// 	continue
		// }

		if !strings.Contains(line, "libapps-admin.uncw.edu") {
			log.Printf("Skipping line without libapps-admin.uncw.edu: %s", line)
			continue
		}
		line = strings.TrimSpace(line)
		line = strings.ReplaceAll(line, "'", "")
		line = strings.ReplaceAll(line, "\"", "")

		imageTag := strings.Replace(line, "libapps-admin.uncw.edu:8000/randall-dev/", "", -1)
		split := strings.Split(imageTag, ":")
		var name, tag string
		if len(split) < 2 {
			name, tag = split[0], ""
		} else {
			name, tag = split[0], split[1]
		}

		breakout := false
		for _, image := range images {
			if image.Name == name {
				image.Tags[tag] = true
				breakout = true
			}
		}
		if breakout {
			break
		}
		newImage := Image{Name: name, Tags: map[string]bool{tag: true}}
		images = append(images, newImage)
	}
	return images
}
