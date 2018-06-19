# Generate tarball with new build of param_api
#


all: clean build build-docker

clean:
	@rm -f ./bin/param-api-latest

build:
	@echo Building param-api version $(VERSION)
	@env CGO_ENABLED=0 GOOS=linux go build -a -tags netgo -ldflags '-w' -o ./bin/param-api-latest *.go

build-docker:
	@echo Building docker tag $(TAG) in $(AWS_REGION)
	@env AWS_REGION=$(AWS_REGION) TAG=$(TAG) docker-compose -f docker/docker-compose.yml build

.PHONY: all clean build
