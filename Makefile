# SAGE ADK Makefile

# Variables
BINARY_NAME=sage-adk
BUILD_DIR=build
GO=go
GOFLAGS=-v
VERSION=$(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
LDFLAGS=-ldflags "-X main.version=${VERSION}"

# Colors for output
RED=\033[0;31m
GREEN=\033[0;32m
YELLOW=\033[0;33m
BLUE=\033[0;34m
NC=\033[0m # No Color

.PHONY: help
help: ## Show this help message
	@echo '$(BLUE)SAGE ADK - Makefile Commands$(NC)'
	@echo ''
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "$(GREEN)%-20s$(NC) %s\n", $$1, $$2}'

.PHONY: all
all: clean deps lint test build ## Run all tasks (clean, deps, lint, test, build)

.PHONY: setup
setup: ## Initial setup - install dependencies and tools
	@echo "$(BLUE)Installing Go dependencies...$(NC)"
	$(GO) mod download
	$(GO) mod verify
	@echo "$(GREEN)Setup complete!$(NC)"

.PHONY: deps
deps: ## Download dependencies
	@echo "$(BLUE)Downloading dependencies...$(NC)"
	$(GO) mod download
	$(GO) mod tidy

.PHONY: build
build: ## Build the main binary
	@echo "$(BLUE)Building $(BINARY_NAME)...$(NC)"
	@mkdir -p $(BUILD_DIR)/bin
	$(GO) build $(GOFLAGS) $(LDFLAGS) -o $(BUILD_DIR)/bin/$(BINARY_NAME) ./cmd/adk
	@echo "$(GREEN)Build complete: $(BUILD_DIR)/bin/$(BINARY_NAME)$(NC)"

.PHONY: build-all
build-all: ## Build all binaries
	@echo "$(BLUE)Building all binaries...$(NC)"
	@mkdir -p $(BUILD_DIR)/bin
	$(GO) build $(GOFLAGS) $(LDFLAGS) -o $(BUILD_DIR)/bin/$(BINARY_NAME) ./cmd/adk
	@echo "$(GREEN)All builds complete!$(NC)"

.PHONY: install
install: ## Install the binary to $GOPATH/bin
	@echo "$(BLUE)Installing $(BINARY_NAME)...$(NC)"
	$(GO) install $(LDFLAGS) ./cmd/adk
	@echo "$(GREEN)Install complete!$(NC)"

.PHONY: test
test: ## Run tests
	@echo "$(BLUE)Running tests...$(NC)"
	$(GO) test -v -race ./...

.PHONY: test-short
test-short: ## Run short tests only
	@echo "$(BLUE)Running short tests...$(NC)"
	$(GO) test -v -short ./...

.PHONY: test-coverage
test-coverage: ## Run tests with coverage
	@echo "$(BLUE)Running tests with coverage...$(NC)"
	$(GO) test -v -race -coverprofile=coverage.txt -covermode=atomic ./...
	$(GO) tool cover -html=coverage.txt -o coverage.html
	@echo "$(GREEN)Coverage report: coverage.html$(NC)"

.PHONY: test-integration
test-integration: ## Run integration tests
	@echo "$(BLUE)Running integration tests...$(NC)"
	$(GO) test -v -tags=integration ./test/integration/...

.PHONY: bench
bench: ## Run benchmarks
	@echo "$(BLUE)Running benchmarks...$(NC)"
	$(GO) test -bench=. -benchmem ./...

.PHONY: lint
lint: ## Run linters
	@echo "$(BLUE)Running linters...$(NC)"
	@which golangci-lint > /dev/null || (echo "$(RED)golangci-lint not installed$(NC)" && exit 1)
	golangci-lint run ./...

.PHONY: fmt
fmt: ## Format code
	@echo "$(BLUE)Formatting code...$(NC)"
	$(GO) fmt ./...
	gofmt -s -w .

.PHONY: vet
vet: ## Run go vet
	@echo "$(BLUE)Running go vet...$(NC)"
	$(GO) vet ./...

.PHONY: clean
clean: ## Clean build artifacts
	@echo "$(BLUE)Cleaning...$(NC)"
	rm -rf $(BUILD_DIR)
	rm -f coverage.txt coverage.html
	$(GO) clean
	@echo "$(GREEN)Clean complete!$(NC)"

.PHONY: run
run: ## Run the application
	@echo "$(BLUE)Running $(BINARY_NAME)...$(NC)"
	$(GO) run ./cmd/adk

.PHONY: examples
examples: ## Build all examples
	@echo "$(BLUE)Building all examples...$(NC)"
	@mkdir -p $(BUILD_DIR)/bin/examples
	@for dir in examples/*/; do \
		if [ -f "$$dir/main.go" ]; then \
			example=$$(basename $$dir); \
			echo "Building $$example..."; \
			cd "$$dir" && $(GO) build $(GOFLAGS) -o ../../$(BUILD_DIR)/bin/examples/$$example . || exit 1; \
		fi \
	done
	@echo "$(GREEN)All examples built in $(BUILD_DIR)/bin/examples/$(NC)"

.PHONY: example-%
example-%: ## Build specific example (e.g., make example-observability)
	@echo "$(BLUE)Building example: $*...$(NC)"
	@mkdir -p $(BUILD_DIR)/bin/examples
	@if [ -d "examples/$*" ]; then \
		cd examples/$* && $(GO) build $(GOFLAGS) -o ../../$(BUILD_DIR)/bin/examples/$* .; \
		echo "$(GREEN)Example built: $(BUILD_DIR)/bin/examples/$*$(NC)"; \
	else \
		echo "$(RED)Example $* not found$(NC)"; \
		exit 1; \
	fi

.PHONY: run-example-%
run-example-%: ## Run specific example (e.g., make run-example-observability)
	@echo "$(BLUE)Running example: $*...$(NC)"
	@if [ -f "$(BUILD_DIR)/bin/examples/$*" ]; then \
		$(BUILD_DIR)/bin/examples/$*; \
	else \
		echo "$(YELLOW)Building $* first...$(NC)"; \
		$(MAKE) example-$* && $(BUILD_DIR)/bin/examples/$*; \
	fi

.PHONY: run-example-simple
run-example-simple: ## Run simple example
	@echo "$(BLUE)Running simple agent example...$(NC)"
	cd examples/simple-agent && $(GO) run main.go

.PHONY: run-example-sage
run-example-sage: ## Run SAGE-enabled example
	@echo "$(BLUE)Running SAGE-enabled agent example...$(NC)"
	cd examples/sage-enabled-agent && $(GO) run main.go

.PHONY: docker-build
docker-build: ## Build Docker image
	@echo "$(BLUE)Building Docker image...$(NC)"
	docker build -t sage-adk:$(VERSION) .
	docker tag sage-adk:$(VERSION) sage-adk:latest
	@echo "$(GREEN)Docker build complete!$(NC)"

.PHONY: docker-run
docker-run: ## Run Docker container
	@echo "$(BLUE)Running Docker container...$(NC)"
	docker run --rm -p 8080:8080 --env-file .env sage-adk:latest

.PHONY: docs
docs: ## Generate documentation
	@echo "$(BLUE)Generating documentation...$(NC)"
	@which godoc > /dev/null || (echo "$(RED)godoc not installed. Run: go install golang.org/x/tools/cmd/godoc@latest$(NC)" && exit 1)
	@echo "$(GREEN)Documentation server starting at http://localhost:6060$(NC)"
	godoc -http=:6060

.PHONY: generate
generate: ## Run go generate
	@echo "$(BLUE)Running go generate...$(NC)"
	$(GO) generate ./...

.PHONY: mod-update
mod-update: ## Update all dependencies
	@echo "$(BLUE)Updating dependencies...$(NC)"
	$(GO) get -u ./...
	$(GO) mod tidy

.PHONY: mod-vendor
mod-vendor: ## Vendor dependencies
	@echo "$(BLUE)Vendoring dependencies...$(NC)"
	$(GO) mod vendor

.PHONY: check
check: fmt vet lint test ## Run all checks (fmt, vet, lint, test)

.PHONY: pre-commit
pre-commit: fmt vet lint test-short ## Run pre-commit checks

.PHONY: release-dry
release-dry: ## Dry run release (test)
	@echo "$(BLUE)Running release dry run...$(NC)"
	@which goreleaser > /dev/null || (echo "$(RED)goreleaser not installed$(NC)" && exit 1)
	goreleaser release --snapshot --clean

.PHONY: release
release: ## Create a release
	@echo "$(BLUE)Creating release...$(NC)"
	@which goreleaser > /dev/null || (echo "$(RED)goreleaser not installed$(NC)" && exit 1)
	goreleaser release --clean

.PHONY: version
version: ## Show version
	@echo "$(GREEN)Version: $(VERSION)$(NC)"

.PHONY: info
info: ## Show project info
	@echo "$(BLUE)SAGE ADK Information$(NC)"
	@echo "Version:     $(VERSION)"
	@echo "Go Version:  $(shell $(GO) version)"
	@echo "Build Dir:   $(BUILD_DIR)"
	@echo "Binary Name: $(BINARY_NAME)"
