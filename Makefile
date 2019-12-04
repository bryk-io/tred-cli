default: help
BINARY_NAME=tred
VERSION_TAG=0.4.0
LD_FLAGS="\
-X github.com/bryk-io/tred-cli/cmd.buildCode=`git log --pretty=format:'%H' -n1` \
-X github.com/bryk-io/tred-cli/cmd.releaseTag=$(VERSION_TAG)"

help: ## Display available make targets
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[33m%-16s\033[0m %s\n", $$1, $$2}'

updates: ## List available updates for direct dependencies
	# https://github.com/golang/go/wiki/Modules#how-to-upgrade-and-downgrade-dependencies
	go list -u -f '{{if (and (not (or .Main .Indirect)) .Update)}}{{.Path}}: {{.Version}} -> {{.Update.Version}}{{end}}' -m all 2> /dev/null

clean: ## Download and compile all dependencies and intermediary products
	go mod tidy
	go mod download
	go mod verify

test: ## Run all tests excluding the vendor dependencies
	# Static analysis
	golangci-lint run -v ./...
	go-consistent -v ./...

	# Unit tests
	go test -race -cover -v ./...

build: ## Build for the default architecture in use
	go build -v -ldflags $(LD_FLAGS) -o $(BINARY_NAME)

install: ## Install the binary to '$GOPATH/bin'
	go build -v -ldflags $(LD_FLAGS) -i -o ${GOPATH}/bin/$(BINARY_NAME)

build-for: ## Build the availabe binaries for the specified 'os' and 'arch'
	CGO_ENABLED=0 GOOS=$(os) GOARCH=$(arch) \
	go build -v -ldflags $(LD_FLAGS) \
	-o $(dest)$(BINARY_NAME)_$(VERSION_TAG)_$(os)_$(arch)$(suffix)

release: ## Prepare the artifacts for a new release
	@-rm -rf release-$(VERSION_TAG)
	mkdir release-$(VERSION_TAG)
	make build-for os=linux arch=amd64 dest=release-$(VERSION_TAG)/
	make build-for os=darwin arch=amd64 dest=release-$(VERSION_TAG)/
	make build-for os=windows arch=amd64 suffix=".exe" dest=release-$(VERSION_TAG)/
	make build-for os=windows arch=386 suffix=".exe" dest=release-$(VERSION_TAG)/
