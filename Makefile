# Build variables
BINARY_NAME=skillsync
VERSION?=0.1.0
BUILD_DIR=bin
COMMIT=$(shell git rev-parse --short HEAD)
BUILD_DATE=$(shell date -u +"%Y-%m-%dT%H:%M:%SZ")
LDFLAGS=-ldflags "-X github.com/klauern/skillsync/internal/cli.Version=$(VERSION) -X github.com/klauern/skillsync/internal/cli.Commit=$(COMMIT) -X github.com/klauern/skillsync/internal/cli.BuildDate=$(BUILD_DATE)"

.PHONY: help
help: ## Display this help message
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-20s\033[0m %s\n", $$1, $$2}'

.PHONY: build
build: ## Build the binary
	@mkdir -p $(BUILD_DIR)
	go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME) ./cmd/skillsync

.PHONY: install
install: ## Install the binary to GOPATH/bin
	go install $(LDFLAGS) ./cmd/skillsync

.PHONY: uninstall
uninstall: ## Remove installed binary from GOPATH/bin
	rm -f $(shell [ -n "$(GOBIN)" ] && echo "$(GOBIN)" || echo "$(shell go env GOPATH)/bin")/$(BINARY_NAME)$(shell go env GOEXE)

.PHONY: test
test: ## Run tests
	go test -v -race -coverprofile=coverage.out ./...

.PHONY: test-coverage
test-coverage: test ## Run tests and display coverage
	go tool cover -html=coverage.out

.PHONY: bench
bench: ## Run benchmark tests
	go test -bench=. -benchmem -benchtime=3s ./...

.PHONY: bench-cpu
bench-cpu: ## Run benchmarks with CPU profiling
	go test -bench=. -benchmem -cpuprofile=cpu.prof ./...
	@echo "View CPU profile with: go tool pprof cpu.prof"

.PHONY: bench-mem
bench-mem: ## Run benchmarks with memory profiling
	go test -bench=. -benchmem -memprofile=mem.prof ./...
	@echo "View memory profile with: go tool pprof mem.prof"

.PHONY: lint
lint: ## Run golangci-lint
	golangci-lint run ./...

.PHONY: fmt
fmt: ## Format code
	gofumpt -w .
	goimports -w -local github.com/klauern/skillsync .

.PHONY: vet
vet: ## Run go vet
	go vet ./...

.PHONY: tidy
tidy: ## Tidy and verify dependencies
	go mod tidy
	go mod verify

.PHONY: install-tools
install-tools: ## Install development tools
	@echo "Installing gofumpt..."
	@go install mvdan.cc/gofumpt@latest
	@echo "Installing goimports..."
	@go install golang.org/x/tools/cmd/goimports@latest
	@echo "Installing golangci-lint..."
	@go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	@echo "All tools installed successfully!"

.PHONY: clean
clean: ## Remove build artifacts
	rm -rf $(BUILD_DIR)
	rm -f coverage.out coverage.html

.PHONY: audit
audit: tidy fmt vet lint test ## Run all quality checks

.PHONY: run
run: build ## Build and run
	./$(BUILD_DIR)/$(BINARY_NAME)

.PHONY: all
all: audit build ## Run audit and build

.DEFAULT_GOAL := help
