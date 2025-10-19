# 🤖 ccmodel - AI Model Configuration Switcher

# Build variables
BINARY_NAME=ccmodel
VERSION ?= $(shell git describe --tags --always --dirty)
COMMIT := $(shell git rev-parse --short HEAD)
DATE := $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")
LDFLAGS=-ldflags "-X main.Version=$(VERSION) -X main.Commit=$(COMMIT) -X main.Date=$(DATE) -s -w"

# Go parameters
GOCMD=go
GOBUILD=$(GOCMD) build
GOCLEAN=$(GOCMD) clean
GOTEST=$(GOCMD) test
GOGET=$(GOCMD) get
GOMOD=$(GOCMD) mod

# Build directories
DIST_DIR=dist
BUILD_DIR=build

.PHONY: all build clean test check install uninstall fmt lint release help quick-test

all: test build

help: ## Show this help message
	@echo '🤖 ccmodel - AI Model Configuration Switcher'
	@echo ''
	@echo 'Available commands:'
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "  %-15s %s\n", $$1, $$2}' $(MAKEFILE_LIST)

build: ## Build binary for current platform
	@echo "🔨 Building $(BINARY_NAME) $(VERSION)..."
	@$(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME) .
	@echo "✅ Built: $(BUILD_DIR)/$(BINARY_NAME)"

build-all: ## Build for all platforms
	@echo "🔨 Building for all platforms..."
	@mkdir -p $(DIST_DIR)
	
	# macOS
	@GOOS=darwin GOARCH=amd64 $(GOBUILD) $(LDFLAGS) -o $(DIST_DIR)/$(BINARY_NAME)-darwin-amd64 .
	@GOOS=darwin GOARCH=arm64 $(GOBUILD) $(LDFLAGS) -o $(DIST_DIR)/$(BINARY_NAME)-darwin-arm64 .
	
	# Linux
	@GOOS=linux GOARCH=amd64 $(GOBUILD) $(LDFLAGS) -o $(DIST_DIR)/$(BINARY_NAME)-linux-amd64 .
	@GOOS=linux GOARCH=arm64 $(GOBUILD) $(LDFLAGS) -o $(DIST_DIR)/$(BINARY_NAME)-linux-arm64 .
	
	# Windows
	@GOOS=windows GOARCH=amd64 $(GOBUILD) $(LDFLAGS) -o $(DIST_DIR)/$(BINARY_NAME)-windows-amd64.exe .
	@GOOS=windows GOARCH=arm64 $(GOBUILD) $(LDFLAGS) -o $(DIST_DIR)/$(BINARY_NAME)-windows-arm64.exe .
	
	@echo "✅ All binaries built in $(DIST_DIR)/"

test: ## Run tests
	@echo "🧪 Running tests..."
	@$(GOTEST) -v ./...

check: test ## Run the local pre-push verification gate
	@echo "🔎 Checking staged and unstaged diff whitespace..."
	@git diff --check
	@git diff --cached --check

fmt: ## Format code
	@echo "🎨 Formatting code..."
	@$(GOCMD) fmt ./...

lint: ## Lint code
	@echo "🔍 Linting code..."
	@which golangci-lint >/dev/null || (echo "Installing golangci-lint..." && go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest)
	@golangci-lint run

clean: ## Clean build artifacts
	@echo "🧹 Cleaning..."
	@$(GOCLEAN)
	@rm -rf $(BUILD_DIR) $(DIST_DIR)

install: build ## Install binary to /usr/local/bin
	@echo "📦 Installing $(BINARY_NAME)..."
	@sudo cp $(BUILD_DIR)/$(BINARY_NAME) /usr/local/bin/
	@echo "✅ Installed to /usr/local/bin/$(BINARY_NAME)"

uninstall: ## Uninstall binary
	@echo "🗑️  Uninstalling $(BINARY_NAME)..."
	@sudo rm -f /usr/local/bin/$(BINARY_NAME)
	@echo "✅ Uninstalled"

dev: ## Run in development mode
	@echo "🚀 Starting development mode..."
	@$(GOCMD) run . list

release: build-all ## Create GitHub release
	@echo "🎉 Creating release $(VERSION)..."
	@echo "Binaries ready in $(DIST_DIR)/"
	@echo "Run: gh release create $(VERSION) $(DIST_DIR)/*"

deps: ## Install dependencies
	@echo "📦 Installing dependencies..."
	@$(GOMOD) download
	@$(GOMOD) tidy

update-deps: ## Update dependencies
	@echo "🔄 Updating dependencies..."
	@$(GOMOD) tidy
	@$(GOGET) -u ./...

# Development helpers
quick-test: ## Local smoke test with sample configs in a temporary HOME
	@echo "🧪 Quick test..."
	@set -eu; \
	tmp_home="$$(mktemp -d)"; \
	trap 'chmod -R u+w "$$tmp_home" 2>/dev/null || true; rm -rf "$$tmp_home"' EXIT; \
	mkdir -p "$$tmp_home/.claude"; \
	printf '%s\n' '{"model":"demo","env":{"DEMO_TOKEN":"placeholder"},"__cc":{"note":"stripped by ccmodel switch"}}' > "$$tmp_home/.claude/settings.demo.json"; \
	printf '%s\n' '{"model":"baseline"}' > "$$tmp_home/.claude/settings.baseline.json"; \
	HOME="$$tmp_home" GOCACHE="$$tmp_home/.cache/go-build" GOMODCACHE="$$tmp_home/go/pkg/mod" GOPATH="$$tmp_home/go" $(GOCMD) run . list >/dev/null; \
	HOME="$$tmp_home" GOCACHE="$$tmp_home/.cache/go-build" GOMODCACHE="$$tmp_home/go/pkg/mod" GOPATH="$$tmp_home/go" $(GOCMD) run . current --json >/dev/null; \
	HOME="$$tmp_home" GOCACHE="$$tmp_home/.cache/go-build" GOMODCACHE="$$tmp_home/go/pkg/mod" GOPATH="$$tmp_home/go" $(GOCMD) run . switch demo >/dev/null; \
	test -f "$$tmp_home/.claude/settings.json"; \
	! grep -q '__cc' "$$tmp_home/.claude/settings.json"; \
	HOME="$$tmp_home" GOCACHE="$$tmp_home/.cache/go-build" GOMODCACHE="$$tmp_home/go/pkg/mod" GOPATH="$$tmp_home/go" $(GOCMD) run . current --json | grep -q '"model": "demo"'; \
	test ! -e test-configs; \
	echo "✅ Quick test passed with a temporary HOME"

install-dev: build ## Install for development
	@echo "🔧 Installing for development..."
	@cp $(BUILD_DIR)/$(BINARY_NAME) ~/go/bin/
	@echo "✅ Development binary ready in ~/go/bin/"
