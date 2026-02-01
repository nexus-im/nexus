# Go Backend Makefile

BINARY_NAME=nexus-server
BUILD_DIR=bin
GO_FILES=$(shell find . -name '*.go' -not -path "./vendor/*")

.PHONY: all build run test lint clean help

all: lint test build

build: ## Build the Go binary
	@echo "Building..."
	@mkdir -p $(BUILD_DIR)
	@CGO_ENABLED=0 go build -o $(BUILD_DIR)/$(BINARY_NAME) .

run: build ## Build and run the server (default port 8081)
	@echo "Running..."
	@$(BUILD_DIR)/$(BINARY_NAME) -addr :8081

test: ## Run unit tests
	@echo "Testing..."
	@go test -v ./...

lint: ## Run linter (using golangci-lint if installed, else go vet)
	@echo "Linting..."
	@if command -v golangci-lint >/dev/null; then \
		golangci-lint run; \
	else \
		echo "golangci-lint not found, running go vet..."; \
		go vet ./...; \
	fi

clean: ## Remove build artifacts
	@echo "Cleaning..."
	@rm -rf $(BUILD_DIR)

help: ## Show this help message
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%%-15s\033[0m %%s\n", $$1, $$2}'
