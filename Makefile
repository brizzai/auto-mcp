.PHONY: build run test clean docker release release-snapshot

# Build configuration
BINARY := auto-mcp
BUILD_DIR := build
CMD_DIR := cmd/auto-mcp

# Docker configuration
DOCKER_IMAGE := auto-mcp

# Build the binary
build:
	go build -o $(BUILD_DIR)/$(BINARY) ./$(CMD_DIR)

# Run the application directly without building
run:
	go run ./$(CMD_DIR)

run-example:
	go run ./$(CMD_DIR) --swagger-file=./examples/swagger/example_swagger.json --mode=sse

deps:
	go install gotest.tools/gotestsum@latest
	go install github.com/goreleaser/goreleaser@latest
	go mod tidy

# Test and code quality
test:
	go test -v ./...

coverage:
	gotestsum --format=testname -- -coverprofile=coverage.out -covermode=atomic ./...

lint:
	golangci-lint run ./...

# Cleanup
clean:
	go clean
	rm -f $(BUILD_DIR)/$(BINARY)

# Docker operations
docker-build:
	docker build -t $(DOCKER_IMAGE) -f Dockerfile .

docker-run:
	docker run --rm -p 8080:8080 $(DOCKER_IMAGE)

# GoReleaser commands
release:
	goreleaser release --clean

release-snapshot:
	goreleaser release --snapshot --clean

.DEFAULT_GOAL := build
