.PHONY: all
.DEFAULT_GOAL := help
BINARY_NAME=tred
VERSION_TAG=0.4.3

# Linker tags
# https://golang.org/cmd/link/
LD_FLAGS += -s -w
LD_FLAGS += -X github.com/bryk-io/tred-cli/cmd.coreVersion=$(VERSION_TAG)
LD_FLAGS += -X github.com/bryk-io/tred-cli/cmd.buildTimestamp=$(shell date +'%s')
LD_FLAGS += -X github.com/bryk-io/tred-cli/cmd.buildCode=$(shell git log --pretty=format:'%H' -n1)

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
	-o $(dest)$(BINARY_NAME)_$(VERSION_TAG)_$(os)_$(arch)$(suffix)

## release: Prepare the artifacts for a new tagged release
release:
	@-rm -rf release-$(VERSION_TAG)
	mkdir release-$(VERSION_TAG)
	make build-for os=linux arch=amd64 dest=release-$(VERSION_TAG)/
	make build-for os=darwin arch=amd64 dest=release-$(VERSION_TAG)/
	make build-for os=windows arch=amd64 suffix=".exe" dest=release-$(VERSION_TAG)/
	make build-for os=windows arch=386 suffix=".exe" dest=release-$(VERSION_TAG)/
