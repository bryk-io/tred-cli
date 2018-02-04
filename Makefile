default: build
BINARY_NAME=tred
LD_FLAGS="\
-X github.com/bryk-io/tred-cli/cmd.buildCode=`git log --pretty=format:'%H' -n1` \
-X github.com/bryk-io/tred-cli/cmd.releaseTag=0.1.0 \
"

# Build for the default architecture in use
build:
	go build -v -ldflags $(LD_FLAGS) -o $(BINARY_NAME)

# Build for linux systems
linux:
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -v -ldflags $(LD_FLAGS) -o $(BINARY_NAME)_linux

# Download and compile all dependencies and intermediary products
clean:
	dep ensure -v

# Install the binary to '$GOPATH/bin'
install:
	go build -v -ldflags $(LD_FLAGS) -i -o ${GOPATH}/bin/$(BINARY_NAME)