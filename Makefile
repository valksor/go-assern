.PHONY: build test race quality ci-quality ci-test install clean run hooks lefthook

help: ## Outputs this help screen
	@grep -E '(^[a-zA-Z0-9_-]+:.*?##.*$$)|(^##)' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}{printf "\033[32m%-30s\033[0m %s\n", $$1, $$2}' | sed -e 's/\[32m##/[33m/'

# Build variables
BINARY_NAME := assern
BUILD_DIR := ./build
CMD_DIR := ./cmd/assern
VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
COMMIT := $(shell git rev-parse --short HEAD 2>/dev/null || echo "none")
BUILD_TIME := $(shell date -u '+%Y-%m-%dT%H:%M:%SZ')
BUILD_FLAGS := -trimpath -v
LDFLAGS := -ldflags "-s -w -X github.com/valksor/go-assern/internal/version.Version=$(VERSION) -X github.com/valksor/go-assern/internal/version.Commit=$(COMMIT) -X github.com/valksor/go-assern/internal/version.BuildTime=$(BUILD_TIME)"

# Default target
all: build ## Build the binary (default target)

build: ## Compile the binary
	@mkdir -p $(BUILD_DIR)
	CGO_ENABLED=0 go build $(BUILD_FLAGS) $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME) $(CMD_DIR)
	@echo "Built $(BUILD_DIR)/$(BINARY_NAME)"

test: ## Run tests with coverage
	${MAKE} quality
	go test -v -cover ./...

race: ## Run tests with race detector
	go test -v -race ./...

coverage: ## Run tests with race detection and coverage profile
	go test -race -covermode atomic -coverprofile=covprofile ./...

coverage-html: coverage ## Generate HTML coverage report
	@mkdir -p .coverage
	go tool cover -html=covprofile -o .coverage/coverage.html

quality: ## Run formatters, linter, vulncheck, and alias check (auto-fixes)
	${MAKE} fmt
	golangci-lint run ./... --fix
	govulncheck ./...
	${MAKE} check-alias

ci-quality: ## CI quality gate: lint + vulncheck + alias check (no auto-fix)
	golangci-lint run ./...
	govulncheck ./...
	${MAKE} check-alias

ci-test: ## CI test run with coverage (no quality/format mutation)
	go test -cover ./...

fmt: ## Format code with go fmt, goimports, and gofumpt
	go fmt ./...
	goimports -w .
	gofumpt -l -w .

install: build ## Install binary locally to GOPATH/bin
	cp $(BUILD_DIR)/$(BINARY_NAME) $(GOPATH)/bin/$(BINARY_NAME)
	@echo "Installed to $(GOPATH)/bin/$(BINARY_NAME)"

clean: ## Clean build artifacts
	rm -rf $(BUILD_DIR)
	rm -rf .coverage covprofile

run: build ## Run the binary (for development)
	$(BUILD_DIR)/$(BINARY_NAME)

run-args: build ## Run the binary with arguments (use ARGS=...)
	$(BUILD_DIR)/$(BINARY_NAME) $(ARGS)

tidy: clean ## Clean and tidy dependencies
	go mod tidy -e
	go get -d -v ./...

deps: ## Download dependencies
	go mod download

version: build ## Show version info
	$(BUILD_DIR)/$(BINARY_NAME) version

hooks: ## Configure git to use versioned hooks
	git config core.hooksPath .githooks
	@echo "Git hooks configured to use .githooks/"

lefthook: ## Install and configure Lefthook pre-commit hooks
	go install github.com/evilmartians/lefthook@latest
	lefthook install
	@echo "Lefthook installed. Pre-commit hooks active."

check-alias:
	@alias_issues="$$(./.github/alias.sh || true)"; \
	if [ -n "$$alias_issues" ]; then \
		echo "❌ Unnecessary import alias detected:"; \
		echo "$$alias_issues"; \
		exit 1; \
	fi
