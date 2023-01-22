.PHONY: *
.DEFAULT_GOAL := help

# Project setup
BINARY_NAME=tredctl
OWNER=bryk-io
DOCKER_IMAGE=ghcr.io/$(OWNER)/$(BINARY_NAME)
PROJECT_REPO=https://github.com/$(OWNER)/$(BINARY_NAME)
MAINTAINERS=''

# State values
GIT_COMMIT_DATE=$(shell TZ=UTC git log -n1 --pretty=format:'%cd' --date='format-local:%Y-%m-%dT%H:%M:%SZ')
GIT_COMMIT_HASH=$(shell git log -n1 --pretty=format:'%H')
GIT_TAG=$(shell git describe --tags --always --abbrev=0 | cut -c 1-7)

# Linker tags
# https://golang.org/cmd/link/
LD_FLAGS += -s -w
LD_FLAGS += -X github.com/bryk-io/tred-cli/cmd.coreVersion=$(GIT_TAG:v%=%)
LD_FLAGS += -X github.com/bryk-io/tred-cli/cmd.buildTimestamp=$(GIT_COMMIT_DATE)
LD_FLAGS += -X github.com/bryk-io/tred-cli/cmd.buildCode=$(GIT_COMMIT_HASH)

help:
	@echo "Commands available"
	@sed -n 's/^##//p' ${MAKEFILE_LIST} | column -t -s ':' | sed -e 's/^/ /' | sort

## build: Build for the default architecture in use
build:
	go build -v -ldflags '$(LD_FLAGS)' -o $(BINARY_NAME)

## build-for: Build the available binaries for the specified 'os' and 'arch'
# make build-for os=linux arch=amd64
build-for:
	CGO_ENABLED=0 GOOS=$(os) GOARCH=$(arch) \
	go build -v -ldflags '$(LD_FLAGS)' \
	-o $(BINARY_NAME)_$(os)_$(arch)$(suffix)

## ca-roots: Generate the list of valid CA certificates
ca-roots:
	@docker run -dit --rm --name ca-roots debian:stable-slim
	@docker exec --privileged ca-roots sh -c "apt update"
	@docker exec --privileged ca-roots sh -c "apt install -y ca-certificates"
	@docker exec --privileged ca-roots sh -c "cat /etc/ssl/certs/* > /ca-roots.crt"
	@docker cp ca-roots:/ca-roots.crt ca-roots.crt
	@docker stop ca-roots

## deps: Download and compile all dependencies and intermediary products
deps:
	go mod tidy
	go clean

## docs: Display package documentation on local server
docs:
	@echo "Docs available at: http://localhost:8080/"
	godoc -http=:8080 -goroot=${GOPATH} -play

## docker: Build docker image
# https://github.com/opencontainers/image-spec/blob/master/annotations.md
docker:
	make build-for os=linux arch=amd64
	mv $(BINARY_NAME)_linux_amd64 $(BINARY_NAME)
	@-docker rmi $(DOCKER_IMAGE):$(GIT_TAG:v%=%)
	@docker build \
	"--label=org.opencontainers.image.title=$(BINARY_NAME)" \
	"--label=org.opencontainers.image.authors=$(MAINTAINERS)" \
	"--label=org.opencontainers.image.created=$(GIT_COMMIT_DATE)" \
	"--label=org.opencontainers.image.revision=$(GIT_COMMIT_HASH)" \
	"--label=org.opencontainers.image.version=$(GIT_TAG:v%=%)" \
	"--label=org.opencontainers.image.source=$(PROJECT_REPO)" \
	--rm -t $(DOCKER_IMAGE):$(GIT_TAG:v%=%) .
	@rm $(BINARY_NAME)

## install: Install the binary to GOPATH and keep cached all compiled artifacts
install:
	go build -v -ldflags '$(LD_FLAGS)' -i -o ${GOPATH}/bin/$(BINARY_NAME)

## lint: Static analysis
lint:
	golangci-lint run -v ./...

## release: Prepare artifacts for a new tagged release
release:
	goreleaser check
	goreleaser release --skip-validate --skip-publish --rm-dist

## scan-deps: Look for known vulnerabilities in the project dependencies
# https://github.com/sonatype-nexus-community/nancy
scan-deps:
	@go list -json -deps ./... | nancy sleuth --skip-update-check

## scan-secrets: Scan project code for accidentally leaked secrets
scan-secrets:
	@docker run --platform linux/amd64 --rm \
	-v $(shell pwd):/proj \
	dxa4481/trufflehog file:///proj \
	-x .exclude-secrets-scan.txt \
	--regex \
	--entropy false

## test: Run unit tests excluding the vendor dependencies
test:
	go test -race -v -failfast -coverprofile=coverage.report ./...
	go tool cover -html coverage.report -o coverage.html

## updates: List available updates for direct dependencies
# https://github.com/golang/go/wiki/Modules#how-to-upgrade-and-downgrade-dependencies
updates:
	@GOWORK=off go list -u -f '{{if (and (not (or .Main .Indirect)) .Update)}}{{.Path}}: {{.Version}} -> {{.Update.Version}}{{end}}' -mod=mod -m all 2> /dev/null
