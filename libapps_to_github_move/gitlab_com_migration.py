#!/usr/bin/env python3

import os
import logging
import shutil

import constants
import github
import log
import gitlab_com


def remove_cloned_folder(project_name):
    os.chdir(constants.REPOS_ROOT)
    if os.path.exists(project_name):
        logging.info(f"Removing existing folder: {project_name}")
        shutil.rmtree(project_name)
    else:
        logging.info(f"No folder to delete: {project_name}")


def move_a_project(gitlab_project, github_projects, force_overwrite=False):
    project_name = gitlab_project.get("name")
    github_has_project = github.exists_github_repo(project_name, github_projects)
    if not github_has_project:
        github.make_github_repo(gitlab_project)
    if github_has_project and force_overwrite:
        logging.info("Forcing overwrite of github repo: {project_name}")
    if github_has_project and not force_overwrite:
        raise Exception(f"Not overwriting existing Github repo: {project_name}")

    os.makedirs(constants.REPOS_ROOT, exist_ok=True)
    os.chdir(constants.REPOS_ROOT)
    remove_cloned_folder(f"{project_name}.git")
    gitlab_com.get_bare_gitlab_repo(gitlab_project)
    github.push_to_github(project_name)
    github.set_github_repo_to_private(project_name)
    github.configure_github_primary_branch(gitlab_project, github_has_project)
    remove_cloned_folder(f"{project_name}.git")

    worked = True
    return worked


def do_one_repo(gitlab_project, github_projects):
    logging.info(f"Starting do_one_repo for {gitlab_project.get('name')}")
    # if the repo exists on gitlab & github, special handling needed.  Flag them:  "force_overwrite" == True
    force_overwrite = False
    if gitlab_project.get("name") in constants.DUPLICATE_REPOS:
        logging.info(f"duplicate repo found: {gitlab_project.get('name')}")
        force_overwrite = True

    try:
        worked = move_a_project(gitlab_project, github_projects, force_overwrite=force_overwrite)
    except Exception as e:
        worked = False
        logging.error(e)
    return worked


def do_gitlab_com():
    completed, failed = [], []
    os.chdir(constants.APP_ROOT)

    gitlab_com_projects = gitlab_com.get_all_gitlab_projects()
    github_projects = github.get_all_github_projects()

    for gitlab_project in gitlab_com_projects:
        worked = do_one_repo(gitlab_project, github_projects)
        project_name = gitlab_project.get("name")
        if worked:
            completed.append(project_name)
        else:
            failed.append(project_name)

    logging.info(f"completed: {completed}")
    logging.info(f"failed: {failed}")


if __name__ == "__main__":
    log.setup_logging()
    logging.info(f"starting {os.path.basename(__file__)}")
    do_gitlab_com()
