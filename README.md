# LoaderHub
Inspo:
https://duckdb.org/2022/10/12/modern-data-stack-in-a-box.html

## Overview
This app is a combination of Go, Python, and PostgreSQL, and requires the following prerequisites to get started with development or usage:

## Prerequisites
Before getting started with the development of this app, make sure you have the following installed on your system:
- [Golang](https://golang.org/doc/install)
- [Docker](https://docs.docker.com/get-docker/)
- [Kubernetes](https://kubernetes.io/docs/tasks/tools/install-kubectl/)
- [Tilt](https://docs.tilt.dev/install.html)

## Local Kubernetes Cluster
If you don't have a local kubernetes cluster set up, you can use the following:

- [Minikube](https://kubernetes.io/docs/tasks/tools/install-minikube/): A tool that makes it easy to run Kubernetes locally. Minikube runs a single-node Kubernetes cluster inside a Virtual Machine (VM) on your laptop.

- [Docker for Desktop](https://docs.docker.com/desktop/get-started/#kubernetes): Includes a standalone Kubernetes server and client, as well as Docker CLI integration.

- [Kind](https://kind.sigs.k8s.io/): A tool for running local Kubernetes clusters using Docker containers as the nodes.

- [Microk8s](https://microk8s.io/): A fast, lightweight, and easy-to-install distribution of Kubernetes that runs natively on Ubuntu.

- [Kubeadm](https://kubernetes.io/docs/reference/setup-tools/kubeadm/): A toolkit that helps users bootstrap a best-practice Kubernetes cluster in an easy and repeatable way.

- [k9s](https://k9scli.io/): A terminal based UI to interact with your Kubernetes clusters.

# Setup Personal Access Token
To use the GitHub API, you will need to create a personal access token. You can do this by following the steps below:
- Go to your GitHub account settings
- Click on Developer settings
- Click on Personal access tokens
- Click on Generate new token
- Give your token a name
- Select the scopes you want to give your token
- Click on Generate token
- Copy the token and save it somewhere safe like a password manager

## Setup Environment Variables with 1password
I reccomend setting up a new vault with 1password for local development items.

Install 1password cli
```bash 
brew install --cask 1password/tap/1password-cli
```
> **_NOTE:_**  If on a Linux machine be sure to disable the biometrics unlock with  
`$ export $OP_BIOMETRIC_UNLOCK_ENABLED=false`

login to 1password [1password CLI GitHub integration](https://developer.1password.com/docs/cli/shell-plugins/github/#step-1-create-a-github-personal-access-token)

```bash
op signin
```
set up environment variables

```bash 
export GITHUB_ACCESS_TOKEN=op://development/github/personal_access_token_dataworkflow
```
start the app with tilt

```bash
op run -- tilt up 
```

## Database
This app uses PostgreSQL for data storage. You can either use an existing database, or create a new one using [Docker](https://hub.docker.com/_/postgres).

## Deployment
The app is deployed using [Kubernetes](https://kubernetes.io/docs/concepts/workloads/controllers/deployment/) and managed using [Tilt](https://tilt.dev/docs/). The deployment and management files are located in the `ops/dev-stack` directory.

## Setup
Clone this repository to your local machine
Install the required software and tools
Configure the environment variables in the respective config files under ops/dev-stack/
Build the Docker images using the Dockerfiles in the Dockerfiles directory
Deploy the app using Tilt by running tilt up in the root directory
Prerequisites
To get started with the app, you will need to have the following installed on your machine:

You will also need to have access to a local kubernetes cluster. If you do not have one, you can use Minikube for testing purposes. 



These are some of the options available, and you can choose the one that works best for you based on your requirements and preferences.
### Containers
#### tips n tricks

If you need to run a container and you want to be able to run commands inside it, you can use the following command:

```sh
docker run -it --rm --entrypoint /bin/ash alpine/git
docker run -it --rm -v /Users/thinkstudio/repos/database_thing/ops/dev-stack/dbt/lakehouse_demo:/opt/venv/lakehouse_demo --entrypoint /bin/ash dbt
```

### Dockerfiles

The Dockerfiles for the different components of the app are located in the Dockerfiles directory. The components are:

go_loader
psql_db
pushup
py_app
ops/dev-stack

The ops/dev-stack directory contains the configuration files for the different components of the app. The components are:
go_loader: configuration, deployment and service files
postgres_db: configuration, deployment and service files, as well as job files
py_app: deployment and service files

#### 
### Tiltfile
The Tiltfile is a configuration file used by Tilt to manage the development environment. To use Tilt, you will need to install it on your machine. Tilt download link.

### debug docker images

https://github.com/GoogleContainerTools/distroless
```
python3 quality_checks/pydeequ/dyno_deequ.py
python3 quality_checks/great_expectations/dyno_gx.py
$ docker run --entrypoint=sh --rm -ti py_app:latest
$ docker run --entrypoint=sh -ti my_debug_image
```