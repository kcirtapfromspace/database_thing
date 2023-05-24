# Data Thing
Inspo:
https://duckdb.org/2022/10/12/modern-data-stack-in-a-box.html

## Overview
This is a demo app that shows how to build a modern data stack using Containers, Kubernetes, and Tilt.
We do this by exploring popular tools and techniques that reflect how to build a production stack.

We begin by populating a postgres database with fake payment data using using python app [datagen][datagen]
[Debezium][debezium] is used to capture changes to the database, and push them to a Kafka topic. From there we use a consumer to push the data to a data lakehouse in parquet format.
[Argo][argo] is used to orchestrate the data pipeline, where data models are pulled from github and run against the lakehouse using [duckdb][duckdb] in place of warehouse tooling such as Redshift, Snowflake. We use [dbt][dbt] to transform the data and export a transformed static dataset back into the lakehouse.
Data quality checks can be conduced by [dbt][dbt],  [Great Expectations][great_expectations], or [deequ][deequ] to validate the data in the lakehouse.

A curated dataset is then pushed to a data warehouse, and a dashboard is built using [rill][rill], [evidence.dev][evidence.dev], or [superset][superset] can be used to visualize the data.

## Prerequisites
Before getting started with the development of this app, make sure you have the following installed on your system:
- [Golang](https://golang.org/doc/install)
- [Python](https://www.python.org/downloads/)
- [Docker](https://docs.docker.com/get-docker/)
- [Kubernetes](https://kubernetes.io/docs/tasks/tools/install-kubectl/)
- [Tilt](https://tilt.dev/install/)

### Local Kubernetes Cluster
If you don't have a local kubernetes cluster set up, you can use the following:
Minikube: A tool that makes it easy to run Kubernetes locally. Minikube runs a single-node Kubernetes cluster inside a Virtual Machine (VM) on your laptop. Kind, K3s and Microk8s are other options that you can use. You can find more information about these tools below:


- [Minikube](https://kubernetes.io/docs/tasks/tools/install-minikube/)
- [Docker](https://www.docker.com/)
- [Podman](https://podman.io/)
- [Kind](https://kind.sigs.k8s.io/)
- [kustomize](https://kustomize.io/)
- [kubectl]

Lately i've been relying on the ctlptl tool to manage my local kubernetes cluster its by the same folks that make tilt.
check out [ctlptl](https://github.com/tilt-dev/ctlptl)
```sh
❯  brew install ctlptl
❯   ctlptl create registry ctlptl-registry
❯   ctlptl create cluster kind --registry=ctlptl-registry --port=5005
```

### Allow Argo to Pull Docker Images into the Local Cluster via a Private Registry
If you are using a local Kubernetes cluster, you will need to pull the Docker images into the cluster.
The dockerconfigjson is a means to inform the argo service account about the local registry and its creds. Current argo workflows need a (PAT) to pull from the container registry and private github repos. I found that a nice workflow is to store all that in 1password and set the environment variables with the 1password cli. I've included a sample workflow below.
```json
{"auths":{"ctlptl-registry:5000":{"username":"","password":"","email":"root@localhost","auth":""}}}
```

```yaml
k8s_yaml(blob("""
apiVersion: v1
kind: Secret
data:
    .dockerconfigjson: eyJhdXRocyI6eyJjdGxwdGwtcmVnaXN0cnk6NTAwMCI6eyJ1c2VybmFtZSI6IiIsInBhc3N3b3JkIjoiIiwiZW1haWwiOiJyb290QGxvY2FsaG9zdCIsImF1dGgiOiIifX19Cg==
metadata:
    name: ctlptl-regcred
    namespace: argo
type: kubernetes.io/dockerconfigjson
---
apiVersion: v1
kind: Secret
metadata:
    name: argo-workflow.service-account-token
    namespace: argo
    annotations:
        kubernetes.io/service-account.name: argo-workflow
type: kubernetes.io/service-account-token
---
apiVersion: v1
kind: Secret
metadata:
    name: argo-secrets
    namespace: argo
type: Opaque
stringData:
    S3_ARTIFACT_ACCESS_KEY: {s3_access}
    S3_ARTIFACT_SECRET_KEY: {s3_secret}
    GITHUB_ACCESS_TOKEN: {github_access_token}
""".format(
    s3_access=S3_ACCESS_KEY,
    s3_secret=S3_SECRET_KEY,
    github_access_token=GITHUB_ACCESS_TOKEN,
)))

```
#### Github Personal Access Token (PAT)
To use the GitHub API, you will need to create a personal access token. You can do this by following the steps below:
- Go to your GitHub account settings
- Click on Developer settings
- Click on Personal access tokens
- Click on Generate new token
- Give your token a name
- Select the scopes you want to give your token
- Click on Generate token
- Copy the token and save it somewhere safe like a password manager

#### Setup Environment Variables with 1password
I recommend setting up a new vault with 1password for local development items
Here is a sample workflow for setting up the environment variables for this app. I've included the 1password cli commands to make it easy to follow along.
``` sh
# install 1password cli
❯ brew install --cask 1password/tap/1password-cli
# login to 1password (link to 1password cli)
❯  $(op signin --acount my.1password.com)
# set up environment variables
❯ export GITHUB_ACCESS_TOKEN=op://development/github/personal_access_token_dataworkflow
# start the app with tilt
❯ op run -- tilt up
```

## Database
This app uses PostgreSQL for data storage. You can either use an existing database, or create a new one using [Docker](https://hub.docker.com/_/postgres). Its is version pinned to postgres:12-alpine as it is the smallest image available and is sufficient for our needs, but also newer versions of postgres have breaking changes that interfere with the debezium kafaka connector.

## Deployment
The app is deployed using [Kubernetes](https://kubernetes.io/docs/concepts/workloads/controllers/deployment/) and managed using [Tilt](https://tilt.dev/docs/). The deployment and management files are located in the `ops/dev-stack` directory. This handles most of custom kustomize files, and the deployment and service files for the different components of the app. External helm charts are pulled in via the tiltfile, and will have their own config files.

## Setup
Clone this repository to your local machine
Install the required software and tools
Configure the environment variables in the respective config files under ops/dev-stack/
Build the Docker images using the Dockerfiles in the Dockerfiles directory
Deploy the app using Tilt by running `tilt up` in the directory the tiltfile is located in.

### Containers
#### Dockerfiles

The Dockerfiles for the different components of the app are located in the Dockerfiles directory. If Tilt can recognize a kubernetes resource that is associated with the dockerfile it will try to build it. If not, you need mock up a dummy resource in the kustomize file to get them to build. You can see this with the 'py_app' This is a bit of a hack, but it works for local development purposes really only a measure for containers that are used within argo workflows. That is just too many layers of abstraction for tilt to deal with for local development. All of these images are now build and pushed to github, so you should not need to build them locally.

#### tips n tricks
If you need to run a container and you want to be able to run commands inside it, you can use the following command:

```sh
❯ docker run -it --rm --entrypoint /bin/ash alpine/git
❯ docker run -it --rm -v /Users/thinkstudio/repos/database_thing/ops/dev-stack/dbt/lakehouse_demo:/opt/venv/lakehouse_demo --entrypoint /bin/ash dbt
```


https://github.com/GoogleContainerTools/distroless
you can update image versions with `:debug` to get a shell in the container

```sh
❯ python3 quality_checks/pydeequ/dyno_deequ.py
❯ python3 quality_checks/great_expectations/dyno_gx.py
❯ docker run --entrypoint=sh --rm -ti py_app:latest
❯ docker run --entrypoint=sh -ti my_debug_image
```

[datagen]: docs/datagen/README.md
[debezium]: docs/debezium/README.md