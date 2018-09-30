default: help
FILES_LIST=`find . -iname '*.go' | grep -v 'vendor'`
GO_PKG_LIST=`go list ./... | grep -v 'vendor'`
BINARY_NAME=tred
LD_FLAGS="\
-X github.com/bryk-io/tred-cli/cmd.buildCode=`git log --pretty=format:'%H' -n1` \
-X github.com/bryk-io/tred-cli/cmd.releaseTag=0.2.0 \
"

build: ## Build for the default architecture in use
	go build -v -ldflags $(LD_FLAGS) -o $(BINARY_NAME)

linux: ## Build for linux systems
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -v -ldflags $(LD_FLAGS) -o $(BINARY_NAME)_linux

clean: ## Download and compile all dependencies and intermediary products
	dep ensure -v

install: ## Install the binary to '$GOPATH/bin'
	go build -v -ldflags $(LD_FLAGS) -i -o ${GOPATH}/bin/$(BINARY_NAME)

test: ## Run all tests excluding the vendor dependencies
	# Formatting
	go vet $(GO_PKG_LIST)
	gofmt -s -l $(FILES_LIST)
	golint -set_exit_status $(GO_PKG_LIST)
	misspell $(FILES_LIST)

	# Static analysis
	ineffassign $(FILES_LIST)
	gosec $(GO_PKG_LIST)
	gocyclo -over 15 `find . -iname '*.go' | grep -v 'vendor' | grep -v '_test.go' | grep -v 'pb.go'`

help: ## Display available make targets
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[33m%-16s\033[0m %s\n", $$1, $$2}'