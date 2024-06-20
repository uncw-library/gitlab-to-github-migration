import json
import logging
import os
import subprocess

import requests

import constants


def get_all_libapps_admin_projects():
    logging.info("Getting all libapps_admin projects")

    projects = []
    url = "https://libapps-admin.uncw.edu/api/v4/projects"
    headers = {"Private-Token": constants.LIBAPPS_ADMIN_TOKEN}
    params = {"per_page": 100, "page": 1}

    while True:
        try:
            response = requests.get(url, headers=headers, params=params, verify=False)
        except Exception as e:
            raise Exception(f"Failed to get projects from libapps_admin: {e}")
        if response.status_code != 200:
            raise Exception(f"Failed to get projects from libapps_admin: {response.text}")

        results = response.json()
        projects.extend(results)
        if response.headers["X-Page"] == response.headers["X-Total-Pages"]:
            break
        params["page"] += 1

    projects.sort(key=lambda x: x["name"])
    logging.info("writing libapps_admin_projects_skeleton.txt")
    os.makedirs(os.path.join(constants.APP_ROOT, "output"), exist_ok=True)
    with open(os.path.join(constants.APP_ROOT, "output", "libapps_admin_projects_skeleton.txt"), "w") as f:
        f.write(json.dumps(projects, indent=4))

    return projects


def get_bare_libapps_admin_repo(libapps_admin_project):
    http_url_to_repo = libapps_admin_project.get("http_url_to_repo")
    logging.info(f"Downloading raw libapps_admin: {http_url_to_repo}")
    result = subprocess.run(["git", "clone", "--bare", http_url_to_repo], capture_output=True, text=True)
    if result.returncode != 0:
        raise Exception(f"Failed to clone repository. {result.stderr}")
