.PHONY: run build test clean help

# Default target
.DEFAULT_GOAL := help

# Variables
BINARY_NAME=fetch-service
PORT=8080

help: ## Show this help message
	@echo 'Usage: make [target]'
	@echo ''
	@echo 'Available targets:'
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2}' $(MAKEFILE_LIST)

run: ## Run the service
	@echo "Starting URL Fetch Service on port $(PORT)..."
	@go run main.go

build: ## Build the binary
	@echo "Building $(BINARY_NAME)..."
	@go build -o $(BINARY_NAME) main.go
	@echo "Build complete: ./$(BINARY_NAME)"

test: ## Run integration tests (requires service to be running)
	@echo "Running integration tests..."
	@./test.sh

test-redirects: ## Run redirect-specific tests (requires service to be running)
	@echo "Running redirect tests..."
	@./test_redirects.sh

test-load: ## Run load and performance tests (requires service to be running)
	@echo "Running load tests..."
	@./test_load.sh

test-load: ## Run load tests (requires service to be running)
	@echo "Running load tests..."
	@./test_load.sh

start: build ## Build and run the service
	@echo "Starting $(BINARY_NAME)..."
	@./$(BINARY_NAME)

clean: ## Remove built binaries
	@echo "Cleaning up..."
	@rm -f $(BINARY_NAME)
	@echo "Clean complete"

fmt: ## Format Go code
	@echo "Formatting code..."
	@go fmt ./...
	@echo "Format complete"

vet: ## Run go vet
	@echo "Running go vet..."
	@go vet ./...
	@echo "Vet complete"

check: fmt vet ## Run formatting and vetting

test-unit: ## Run unit tests
	@echo "Running unit tests..."
	@go test -v

test-coverage: ## Run tests with coverage
	@echo "Running tests with coverage..."
	@go test -v -cover
	@go test -coverprofile=coverage.out
	@echo "Coverage report generated: coverage.out"
	@go tool cover -func=coverage.out

test-race: ## Run tests with race detection
	@echo "Running tests with race detection..."
	@go test -race -v

test-bench: ## Run benchmark tests
	@echo "Running benchmark tests..."
	@go test -bench=. -benchmem

test-all: test-unit test-race test-coverage ## Run all test suites

coverage-html: test-coverage ## Generate HTML coverage report
	@echo "Generating HTML coverage report..."
	@go tool cover -html=coverage.out -o coverage.html
	@echo "Open coverage.html in your browser"

# Example commands
example-post: ## Example: Submit URLs via POST
	@echo "Submitting URLs for fetching..."
	@curl -X POST http://localhost:$(PORT)/fetch \
		-H "Content-Type: application/json" \
		-d '{"urls": ["https://example.com", "https://google.com", "https://httpbin.org/html"]}'
	@echo "\n"

example-get: ## Example: Retrieve results via GET
	@echo "Retrieving fetch results..."
	@curl -s http://localhost:$(PORT)/fetch | jq '.'

example-health: ## Example: Check health status
	@echo "Checking health..."
	@curl http://localhost:$(PORT)/health
	@echo "\n"

example-summary: ## Example: Show summary statistics
	@echo "Fetch summary:"
	@curl -s http://localhost:$(PORT)/fetch | jq '{total: .total_urls, success: .success_count, failed: .failed_count, pending: .pending_count}'

