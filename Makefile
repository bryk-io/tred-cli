default: build
BINARY_NAME=tred

# Build for the default architecture in use
build:
	go build -v -o $(BINARY_NAME)

# Build linux systems
build-linux:
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -v -o $(BINARY_NAME)_linux

# Download and compile all dependencies and intermediary products
clean:
	rm -rf vendor glide.lock
	glide cache-clear
	glide install
