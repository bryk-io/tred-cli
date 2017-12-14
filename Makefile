default: build
BINARY_NAME=tred

# Build for the default architecture in use
build:
	go build -v -o $(BINARY_NAME)

# Build for linux systems
linux:
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -v -o $(BINARY_NAME)_linux

# Download and compile all dependencies and intermediary products
clean:
	dep ensure -v

# Install the binary to '$GOPATH/bin'
install:
	go build -v -i -o ${GOPATH}/bin/$(BINARY_NAME)