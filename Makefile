.PHONY: all
.DEFAULT_GOAL := help
BINARY_NAME=tred
VERSION_TAG=0.4.2

# Custom linker tags
LD_FLAGS="\
-X github.com/bryk-io/tred-cli/cmd.buildCode=`git log --pretty=format:'%H' -n1` \
-X github.com/bryk-io/tred-cli/cmd.releaseTag=$(VERSION_TAG)"

## help: Prints this help message
help:
	@echo "Commands available"
	@sed -n 's/^##//p' ${MAKEFILE_LIST} | column -t -s ':' | sed -e 's/^/ /' | sort

## updates: List available updates for direct dependencies
updates:
	# https://github.com/golang/go/wiki/Modules#how-to-upgrade-and-downgrade-dependencies
	go list -u -f '{{if (and (not (or .Main .Indirect)) .Update)}}{{.Path}}: {{.Version}} -> {{.Update.Version}}{{end}}' -m all 2> /dev/null

## clean: Verify dependencies and remove intermediary products
clean:
	@-rm -rf vendor
	go clean
	go mod tidy
	go mod verify

## test: Run all tests excluding the vendor dependencies
test:
	# Static analysis
	golangci-lint run -v ./...
	go-consistent -v ./...

	# Unit tests
	go test -race -cover -v ./...

## build: Build for the default architecture in use
build:
	go build -v -ldflags $(LD_FLAGS) -o $(BINARY_NAME)

## install: Install the binary to '$GOPATH/bin'
install:
	go build -v -ldflags $(LD_FLAGS) -i -o ${GOPATH}/bin/$(BINARY_NAME)

## build-for: Build the availabe binaries for the specified 'os' and 'arch'
build-for:
	CGO_ENABLED=0 GOOS=$(os) GOARCH=$(arch) \
	go build -v -ldflags $(LD_FLAGS) \
	-o $(dest)$(BINARY_NAME)_$(VERSION_TAG)_$(os)_$(arch)$(suffix)

## release: Prepare the artifacts for a new tagged release
release:
	@-rm -rf release-$(VERSION_TAG)
	mkdir release-$(VERSION_TAG)
	make build-for os=linux arch=amd64 dest=release-$(VERSION_TAG)/
	make build-for os=darwin arch=amd64 dest=release-$(VERSION_TAG)/
	make build-for os=windows arch=amd64 suffix=".exe" dest=release-$(VERSION_TAG)/
	make build-for os=windows arch=386 suffix=".exe" dest=release-$(VERSION_TAG)/

## ci-conf: Update CI/CD configuration file
ci-conf:
	drone lint .drone.yml
	@DRONE_SERVER=${BRYK_DRONE_SERVER} DRONE_TOKEN=${BRYK_DRONE_TOKEN} drone sign --save bryk-io/tred-cli
