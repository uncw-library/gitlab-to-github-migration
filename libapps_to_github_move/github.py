import json
import logging
import os
import subprocess
import time

import requests

import constants


def get_all_github_projects():
    logging.info("Getting all GitHub projects")
    url = "https://api.github.com/orgs/uncw-library/repos"
    headers = {
        "Accept": "application/vnd.github+json",
        "Authorization": f"Bearer {constants.GITHUB_TOKEN}",
        "X-GitHub-Api-Version": "2022-11-28",
    }
    projects = []
    page = 1
    while True:
        params = {"page": page}
        try:
            response = requests.get(url, headers=headers, params=params)
        except Exception as e:
            raise Exception(f"Failed to get projects from GitHub: {e}")
        if response.status_code != 200:
            raise Exception("Could not get GitHub repos", response.text)

        results = response.json()
        if not results:
            break
        projects.extend(results)
        page += 1

    projects.sort(key=lambda x: x["name"])
    logging.info("writing github_projects_skeleton.txt")
    os.makedirs(os.path.join(constants.APP_ROOT, "output"), exist_ok=True)
    with open(os.path.join(constants.APP_ROOT, "output", "github_projects_skeleton.txt"), "w") as f:
        f.write(json.dumps(projects, indent=4))

    return projects


def make_github_repo(project):
    logging.info(f"Making GitHub repo for {project.get('name')}")
    headers = {
        "Accept": "application/vnd.github+json",
        "Authorization": f"Bearer {constants.GITHUB_TOKEN}",
        "X-GitHub-Api-Version": "2022-11-28",
    }
    data = {
        "name": project.get("name"),
        "description": project.get("description"),
        "private": True,
        "has_issues": False,
        "has_projects": False,
        "has_wiki": False,
    }
    response = requests.post("https://api.github.com/orgs/uncw-library/repos", headers=headers, data=json.dumps(data))
    if response.status_code != 201:
        raise Exception("Could not create repository", response.text)
    logging.info(f"response: {response.text}")


def exists_github_repo(project_name, github_projects):
    logging.info(f"Checking if GitHub repo exists: {project_name}")
    existing_github_names = [i.get("name") for i in github_projects]
    if project_name in existing_github_names:
        return True
    return False


def push_to_github(project_name):
    os.chdir(os.path.join(constants.REPOS_ROOT, f"{project_name}.git"))
    github_url = f"https://github.com/uncw-library/{project_name}"
    result = subprocess.run(["git", "push", "--mirror", github_url], capture_output=True, text=True)
    if result.returncode != 0:
        raise Exception(f"Git push failed with: {result.stderr}")
    logging.info(f"Pushed to GitHub: {project_name} {github_url}")


def set_github_repo_to_private(project_name):
    headers = {
        "Accept": "application/vnd.github.v3+json",
        "Authorization": f"Bearer {constants.GITHUB_TOKEN}",
        "X-GitHub-Api-Version": "2022-11-28",
    }
    data = {"name": project_name, "private": True}
    response = requests.patch(
        f"https://api.github.com/repos/uncw-library/{project_name}",
        headers=headers,
        json=data,
    )
    if not 200 <= response.status_code < 300:
        raise Exception("Could not update repo visibility", response.text)
    logging.info(f"Repo {project_name} set to private")


def configure_github_primary_branch(gitlab_project, github_has_project):
    # the complication is:
    # if github already has the project, we need to use the primary branch name from github
    # otherwise, we use the primary branch name from gitlab
    # So, we start by making the github primary branch == whatever the above gives us
    # Then we rename it to "main" if it was "master"
    # Then we re-make the github primary branch == "main"
    logging.info(f"github already has project? {github_has_project}")
    project_name = gitlab_project["name"]
    if not github_has_project:
        primary_branch = gitlab_project.get("default_branch")
        logging.info(f"Using gilab's primary branch: {primary_branch}")
    else:
        primary_branch = get_github_primary_branch(project_name)
        logging.info(f"Using github's primary branch: {primary_branch}")
    set_github_primary_branch(project_name, primary_branch)
    if primary_branch == "master":
        rename_github_master_to_main(project_name)
    set_github_primary_branch(project_name, "main")


def get_github_primary_branch(project_name):
    # this needs to stay a live check.  The primary branch will change during the process
    headers = {
        "Accept": "application/vnd.github.v3+json",
        "Authorization": f"Bearer {constants.GITHUB_TOKEN}",
        "X-GitHub-Api-Version": "2022-11-28",
    }
    response = requests.get(f"https://api.github.com/repos/uncw-library/{project_name}", headers=headers)

    if not 200 <= response.status_code < 300:
        raise Exception("Could not get repo info", response.text)

    json_response = response.json()
    primary_branch = json_response["default_branch"]
    logging.info(f"Github primary branch was: {primary_branch}")
    return primary_branch


def set_github_primary_branch(project_name, branch_name):
    logging.info(f"starting set_github_primary_branch: {project_name} {branch_name}")
    headers = {
        "Accept": "application/vnd.github+json",
        "Authorization": f"Bearer {constants.GITHUB_TOKEN}",
        "X-GitHub-Api-Version": "2022-11-28",
    }
    data = {"default_branch": branch_name}
    response = requests.patch(
        f"https://api.github.com/repos/uncw-library/{project_name}",
        headers=headers,
        data=json.dumps(data),
    )
    logging.info(f"response: {response}")
    logging.info(f"response.status_code: {response.status_code}")
    if not 200 <= response.status_code < 300:
        raise Exception("Request failed", response.text)

    # hard pause until branch is updated
    while True:
        if get_github_primary_branch(project_name) == branch_name:
            break
        logging.info(f"waiting for branch to update")
        time.sleep(100)

    logging.info(f"Github primary branch set to {branch_name}")


def rename_github_master_to_main(project_name):
    logging.info("starting rename_github_master_to_main")

    # hard pause until master branch is available
    while True:
        if get_github_primary_branch(project_name) == "master":
            break
            logging.info(f"waiting for branch to update")
        time.sleep(100)

    headers = {
        "Accept": "application/vnd.github+json",
        "Authorization": f"Bearer {constants.GITHUB_TOKEN}",
        "X-GitHub-Api-Version": "2022-11-28",
    }
    data = {"new_name": "main"}
    url = f"https://api.github.com/repos/uncw-library/{project_name}/branches/master/rename"
    logging.info(f"sending {url} with headers {headers} and data {data}")
    response = requests.post(url, headers=headers, data=json.dumps(data))
    logging.info(f"response.status_code: {response.status_code}")
    logging.info(f"response: {response.text}")
    if not 200 <= response.status_code < 300:
        raise Exception("Request failed", response.text)
    logging.info(f"Github branch master renamed to main")
