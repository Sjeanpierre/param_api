# AWS Parameter Store API

A helpful Golang-application that provides a HTTP endpoint to retrieve parameters from AWS Parameter Store.

> STOP WORRING ABOUT HAVING TO REVISION CONTROL ENV VARIABLES, STOP HAVING TO UPLOAD SECRETS TO S3!

## Design

For AWS paramater store, the entry should be in the format:
```
landscape.environment.application_name.KEY_NAME
```

To revision the same entry, edit it, bump the version number in the description, and change the value. 

**NOTE:** 

`AWS Param Store keeps the history of changed values. If you want to use an old version, you simply have to search for the key name with a description of the version you want to access.`

<img width="702" alt="image" src="https://cloud.githubusercontent.com/assets/1714316/23956875/119c064a-095b-11e7-9396-6382b9d49bf4.png">

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

**Push the Docker image**

`You can upload this to ECR, Dockerhub, or any other prefered docker image revision control.`

## Implementation Options

| Type  | Description |
| ------------- | ------------- |
| Standalone  | You can set this application up by building the Docker image for it and running it on a single server. You will need to allow http requests to that server on port 8080  |
| Multi-Container  | You can setup this application through a multi-container approach through AWS Elastic Beanstalk, AWS ECS, or on any server using docker-compose with multiple container definitions. |

`The param-store-api will return JSON data with the environment variables( key/value ). It's up to you if you want to perform the HTTP request via command line and then export the values, or if you want your application make the HTTP request and interpret the variabls.`

## Usage

**Run the image**

```
AWS_REGION={SOME_REGION} TAG={SOME_DOCKER_IMAGE_TAG} docker-compose -f docker/docker-compose.yml up
```

**Retrieve App config as a single JSON document from AWS SSM parameter store**

<img width="702" alt="image" src="https://cloud.githubusercontent.com/assets/673382/23838969/fa9e3f3a-0770-11e7-895c-9159af2cc37f.png">
