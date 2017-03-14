# AWS Parameter Store API

A helpful Golang-application that provides a HTTP endpoint to retrieve parameters from AWS Parameter Store.

## Installation

**Clone the repo**

```
git clone git@github.com:Sjeanpierre/param_api.git
```

**Build the binary**

```
make build
```

**Build the Docker image**

```
AWS_REGION={SOME_REGION} TAG={SOME_DOCKER_IMAGE_TAG} docker-compose -f docker/docker-compose.yml build
```

## Usage

**Run the image**

```
AWS_REGION={SOME_REGION} TAG={SOME_DOCKER_IMAGE_TAG} docker-compose -f docker/docker-compose.yml up
```

**Retrieve App config as a single JSON document from AWS SSM parameter store**

<img width="702" alt="image" src="https://cloud.githubusercontent.com/assets/673382/23838969/fa9e3f3a-0770-11e7-895c-9159af2cc37f.png">
