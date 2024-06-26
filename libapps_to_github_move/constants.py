import os

from dotenv import dotenv_values

APP_ROOT = os.path.dirname(os.path.realpath(__file__))
REPOS_ROOT = os.path.join(APP_ROOT, "repos")

config = dotenv_values(os.path.join(APP_ROOT, "..", ".env")
try:
    LIBAPPS_ADMIN_TOKEN = config["LIBAPPS_ADMIN_TOKEN"]
    GITHUB_TOKEN = config["GITHUB_TOKEN"]
    GITLAB_COM_TOKEN = config["GITLAB_COM_TOKEN"]
except KeyError as e:
    raise Exception(f"Missing required .env variable: {e}")

# uncomment as you prove the commits are equal
DUPLICATE_REPOS = [
    # 'vivo-docker2',
]
