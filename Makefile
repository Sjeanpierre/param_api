# Generate tarball with new build of param_api
#


all: clean build

clean:
	@rm -f ./bin/param-api-latest

build:
	@echo Building param-api version $(VERSION)
	@env CGO_ENABLED=0 GOOS=linux go build -a -tags netgo -ldflags '-w' -o ./bin/param-api-latest *.go

.PHONY: all clean build
