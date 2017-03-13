# Generate tarball with new build of param_api
#
# NOTE: OSX only
VERSION=$$(cat main.go | grep -i "cliVersion =" | awk {'print$$3'} | tr -d '"')


all: clean build compress report

clean:
	@rm -f /tmp/param-api-*.tar.gz
	@rm -f ./bin/param-api-latest

build:
	@echo Building param-api version $(VERSION)
	@env CGO_ENABLED=0 GOOS=linux go build -a -tags netgo -ldflags '-w' -o ./bin/param-api-$(VERSION) *.go 
	@cp ./bin/param-api-$(VERSION) ./bin/param-api-latest

compress:
	@tar czf /tmp/param-api-$(VERSION).tar.gz ./bin

report:
	@rm -f ./bin/param-api-latest
	@shasum -a 256 /tmp/param-api-$(VERSION).tar.gz

.PHONY: all clean build
