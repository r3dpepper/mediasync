.PHONY: build build-all test clean install run help

BINARY_NAME=media-server
VERSION=1.0.0
BUILD_DIR=build
GO_FILES=$(shell find . -name '*.go' -not -path "./vendor/*")

# Go parameters
GOCMD=go
GOBUILD=$(GOCMD) build
GOCLEAN=$(GOCMD) clean
GOTEST=$(GOCMD) test
GOGET=$(GOCMD) get
GOMOD=$(GOCMD) mod

# Build flags
LDFLAGS=-ldflags "-s -w -X main.version=$(VERSION)"

help: ## Show this help message
	@echo 'Usage: make [target]'
	@echo ''
	@echo 'Available targets:'
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "  %-15s %s\n", $$1, $$2}'

build: ## Build for current platform
	@echo "Building $(BINARY_NAME)..."
	@mkdir -p $(BUILD_DIR)
	$(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME) ./cmd/server

build-all: ## Build for all platforms (macOS, Linux, Windows)
	@echo "Building for all platforms..."
	@mkdir -p $(BUILD_DIR)
	
	# macOS amd64
	GOOS=darwin GOARCH=amd64 $(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-darwin-amd64 ./cmd/server
	
	# macOS arm64 (Apple Silicon)
	GOOS=darwin GOARCH=arm64 $(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-darwin-arm64 ./cmd/server
	
	# Linux amd64
	GOOS=linux GOARCH=amd64 $(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-linux-amd64 ./cmd/server
	
	# Linux arm64 (Raspberry Pi, etc.)
	GOOS=linux GOARCH=arm64 $(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-linux-arm64 ./cmd/server
	
	# Windows amd64
	GOOS=windows GOARCH=amd64 $(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-windows-amd64.exe ./cmd/server
	
	@echo "Build complete. Binaries in $(BUILD_DIR)/"

test: ## Run tests
	$(GOTEST) -v -race -coverprofile=coverage.out ./...

test-coverage: test ## Run tests with coverage report
	$(GOCMD) tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report: coverage.html"

clean: ## Clean build artifacts
	$(GOCLEAN)
	rm -rf $(BUILD_DIR)
	rm -f coverage.out coverage.html

deps: ## Download dependencies
	$(GOMOD) download
	$(GOMOD) verify

install: build ## Install to system (requires sudo on Unix)
	@echo "Installing $(BINARY_NAME)..."
ifeq ($(shell uname -s),Darwin)
	sudo cp $(BUILD_DIR)/$(BINARY_NAME) /usr/local/bin/
	sudo chmod +x /usr/local/bin/$(BINARY_NAME)
else ifeq ($(shell uname -s),Linux)
	sudo cp $(BUILD_DIR)/$(BINARY_NAME) /usr/local/bin/
	sudo chmod +x /usr/local/bin/$(BINARY_NAME)
else
	@echo "Manual installation required for Windows"
	@echo "Copy $(BUILD_DIR)/$(BINARY_NAME).exe to C:\\Windows\\System32\\"
endif

run: build ## Build and run the server
	$(BUILD_DIR)/$(BINARY_NAME) start

init: build ## Initialize configuration
	$(BUILD_DIR)/$(BINARY_NAME) init --storage=/tmp/media-test --port=8080

dev: ## Run in development mode with auto-reload (requires air)
	air

lint: ## Run linters (requires golangci-lint)
	golangci-lint run ./...

fmt: ## Format code
	$(GOCMD) fmt ./...
	gofmt -s -w $(GO_FILES)

vet: ## Run go vet
	$(GOCMD) vet ./...

tidy: ## Tidy go.mod
	$(GOMOD) tidy

update-deps: ## Update dependencies
	$(GOMOD) get -u ./...
	$(GOMOD) tidy

docker-build: ## Build Docker image
	docker build -t media-server:$(VERSION) .

docker-run: docker-build ## Run in Docker container
	docker run -p 8080:8080 -v $(PWD)/data:/data media-server:$(VERSION)

release: test build-all ## Create release (build all + run tests)
	@echo "Creating release $(VERSION)"
	@mkdir -p $(BUILD_DIR)/release
	@cd $(BUILD_DIR) && tar -czf release/$(BINARY_NAME)-$(VERSION)-darwin-amd64.tar.gz $(BINARY_NAME)-darwin-amd64
	@cd $(BUILD_DIR) && tar -czf release/$(BINARY_NAME)-$(VERSION)-darwin-arm64.tar.gz $(BINARY_NAME)-darwin-arm64
	@cd $(BUILD_DIR) && tar -czf release/$(BINARY_NAME)-$(VERSION)-linux-amd64.tar.gz $(BINARY_NAME)-linux-amd64
	@cd $(BUILD_DIR) && tar -czf release/$(BINARY_NAME)-$(VERSION)-linux-arm64.tar.gz $(BINARY_NAME)-linux-arm64
	@cd $(BUILD_DIR) && zip -q release/$(BINARY_NAME)-$(VERSION)-windows-amd64.zip $(BINARY_NAME)-windows-amd64.exe
	@echo "Release packages created in $(BUILD_DIR)/release/"

.DEFAULT_GOAL := help
