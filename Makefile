.PHONY: all test lint build clean

all: lint test build

# Run all tests
test:
	go test -v ./...

# Run golangci-lint (requires it to be installed locally)
# To install: go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
lint:
	golangci-lint run

# Build the binary
build:
	go build -v -o repowalk .

# Clean build artifacts
clean:
	rm -f repowalk
	rm -f repowalk.exe
