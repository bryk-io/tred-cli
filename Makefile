.PHONY: *
.DEFAULT_GOAL := help

# Project setup
BINARY_NAME=tredctl

# State values
GIT_COMMIT_DATE=$(shell TZ=UTC git log -n1 --pretty=format:'%cd' --date='format-local:%Y-%m-%dT%H:%M:%SZ')
GIT_COMMIT_HASH=$(shell git log -n1 --pretty=format:'%H')
GIT_TAG=$(shell git describe --abbrev=0 --match='v*' --always | cut -c 1-8)

# Linker tags
# https://golang.org/cmd/link/
LD_FLAGS += -s -w
LD_FLAGS += -X github.com/bryk-io/tred-cli/cmd.coreVersion=$(GIT_TAG)
LD_FLAGS += -X github.com/bryk-io/tred-cli/cmd.buildTimestamp=$(GIT_COMMIT_DATE)
LD_FLAGS += -X github.com/bryk-io/tred-cli/cmd.buildCode=$(GIT_COMMIT_HASH)

## help: Prints this help message
help:
	@echo "Commands available"
	@sed -n 's/^##//p' ${MAKEFILE_LIST} | column -t -s ':' | sed -e 's/^/ /' | sort

## updates: List available updates for direct dependencies
updates:
	# https://github.com/golang/go/wiki/Modules#how-to-upgrade-and-downgrade-dependencies
	go list -u -f '{{if (and (not (or .Main .Indirect)) .Update)}}{{.Path}}: {{.Version}} -> {{.Update.Version}}{{end}}' -m all 2> /dev/null

## scan: Look for known vulnerabilities in the project dependencies
# https://github.com/sonatype-nexus-community/nancy
scan:
	@nancy -quiet go.sum

## clean: Verify dependencies and remove intermediary products
clean:
	@-rm -rf vendor
	go clean
	go mod tidy
	go mod verify

## lint: Static analysis
lint:
	golangci-lint run -v ./...

## test: Run unit tests excluding the vendor dependencies
test:
	go test -v -race -failfast -coverprofile=coverage.report ./...
	go tool cover -html coverage.report -o coverage.html

## build: Build for the default architecture in use
build:
	go build -v -ldflags '$(LD_FLAGS)' -o $(BINARY_NAME)

## install: Install the binary to '$GOPATH/bin'
install:
	go build -v -ldflags '$(LD_FLAGS)' -i -o ${GOPATH}/bin/$(BINARY_NAME)

## build-for: Build the available binaries for the specified 'os' and 'arch'
build-for:
	CGO_ENABLED=0 GOOS=$(os) GOARCH=$(arch) \
	go build -v -ldflags '$(LD_FLAGS)' \
	-o $(BINARY_NAME)_$(os)_$(arch)$(suffix)

## release: Prepare the artifacts for a new tagged release
release:
	goreleaser release --skip-validate --skip-publish --rm-dist
