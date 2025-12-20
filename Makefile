# The Go source code directory
SRC_DIR=./pkg/...

# Go executable
GO=go

# Build the Go application
build:
	$(GO) build $(SRC_DIR)

# Run the Go application
run:
	$(GO) run $(SRC_DIR) $(ARGS)

# Run tests
test:
	$(GO) test -v ./...

# Clean the project (remove binary)
clean:
	rm -f bin/$(BINARY_NAME)
	rmdir bin

# Format Go code
fmt:
	$(GO) fmt ./...

# Lint Go code (optional, requires installation of golangci-lint)
lint:
	golangci-lint run

# Install dependencies
install:
	$(GO) mod tidy
	$(GO) build -o $(GOBIN)/dprime-oracle $(SRC_DIR)

# Build and test the project
all: clean build test

