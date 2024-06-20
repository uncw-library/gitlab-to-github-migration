import json
import logging
import os
import subprocess

import requests

import constants


def get_all_gitlab_projects():
    logging.info("Getting all GitLab projects")

    projects = []
    # namespaces = [
    #     "randall-scripts",
    #     "randall-d6-d7",
    #     "randall3",
    #     "randallstaff",
    #     "randallwebvm",
    #     "randall-laravel",
    #     "randall-archived",
    # ]
    namespaces = ["randall-drupal"]
    for namespace in namespaces:
        url = f"https://gitlab.com/api/v4/groups/{namespace}/projects"
        headers = {"Private-Token": constants.GITLAB_COM_TOKEN}
        params = {"per_page": 100, "page": 1}

        while True:
            try:
                response = requests.get(url, headers=headers, params=params, verify=False)
            except Exception as e:
                raise Exception(f"Failed to get projects from GitLab: {e}")
            if response.status_code != 200:
                raise Exception(f"Failed to get projects from GitLab: {response.text}")

            results = response.json()
            projects.extend(results)
            if not results:
                break
            params["page"] += 1

    projects.sort(key=lambda x: x["name"])
    logging.info("writing gitlab_com_projects_skeleton.txt")
    os.makedirs(os.path.join(constants.APP_ROOT, "output"), exist_ok=True)
    with open(os.path.join(constants.APP_ROOT, "output", "gitlab_com_projects_skeleton.txt"), "w") as f:
        f.write(json.dumps(projects, indent=4))

    return projects


def get_bare_gitlab_repo(gitlab_project):
    logging.info(f"Downloading raw gitlab: {gitlab_project['name']}")
    http_url_to_repo = gitlab_project.get("http_url_to_repo")
    result = subprocess.run(["git", "clone", "--bare", http_url_to_repo], capture_output=True, text=True)
    if result.returncode != 0:
        raise Exception(f"Failed to clone repository. {result.stderr}")
