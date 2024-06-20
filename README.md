# Gitlab to Github migration tools

UNCW-Library is migrating it's git repos from a Gitlab.com instance and from a self-hosted Gitlab instance.  The destination is github.com/uncw-library.  We're also migrating our Gitlab docker registry images to dockerhub.com/uncw-library.

These scripts automate that process.

# setup
libapps_to_github_move is a Python script.  Installing a pyvenv plus requests and dotenv modules.

image_to_dockerhub, repoSed, and localRepoUpdate are Go scripts.

# Script details
image_to_dockerhub finds all the images on our self-hosted Gitlab, then pushes them to dockerhub.

repoSed pulls all git repos from our self-hosted Gitlab, then updates the git url & docker image urls.  Then pushes them back to our Gitlab.

localRepoUpdate runs on each local machine.  It finds the git repos & revises their remote origin & default branch.
