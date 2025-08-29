# Makefile for WHAM!

# Use 'go' as the default, but allow overriding it from the command line
# (e.g., `make GO=/path/to/custom/go build`). This is the most standard and robust approach.
GO ?= go

# Versioning. Use git describe to get a version string.
# This will be like "v1.2.3" on a tag, or "v1.2.3-4-g5c0f8d7" between tags.
VERSION ?= $(shell git describe --tags --always --dirty)

# Default target executed when you just run `make`.
all: build ## Build the binary (default)

# Build the WHAM! binary.
build: ## Build the wham binary
	@if [ -z "$$(command -v $(GO))" ]; then \
		echo "\033[91mError: '$(GO)' command not found.\033[0m" >&2; \
		echo "Please ensure Go is installed and its 'bin' directory is in your PATH." >&2; \
		echo "If you installed Go in a custom location, you can specify the path manually, e.g.:" >&2; \
		echo "  \033[36mmake GO=/path/to/your/go/bin/go build\033[0m" >&2; \
		exit 1; \
	fi
	@{ \
		PKG=$$($(GO) list -m); \
		COMMIT=$$(git rev-parse --short HEAD); \
		BUILD_DATE=$$(date -u +'%Y-%m-%dT%H:%M:%SZ'); \
		LDFLAGS="-s -w -X '$${PKG}/cmd.Version=${VERSION}' -X '$${PKG}/cmd.Commit=$${COMMIT}' -X '$${PKG}/cmd.BuildDate=$${BUILD_DATE}'"; \
		echo "==> Building WHAM! version ${VERSION}..."; \
		$(GO) build -ldflags "$${LDFLAGS}" -o wham .; \
	}

# Install WHAM! to your GOBIN.
install: ## Install wham to your GOBIN
	@if [ -z "$$(command -v $(GO))" ]; then \
		echo "\033[91mError: '$(GO)' command not found.\033[0m" >&2; \
		echo "Please ensure Go is installed and its 'bin' directory is in your PATH." >&2; \
		echo "If you installed Go in a custom location, you can specify the path manually, e.g.:" >&2; \
		echo "  \033[36mmake GO=/path/to/your/go/bin/go install\033[0m" >&2; \
		exit 1; \
	fi
	@{ \
		PKG=$$($(GO) list -m); \
		COMMIT=$$(git rev-parse --short HEAD); \
		BUILD_DATE=$$(date -u +'%Y-%m-%dT%H:%M:%SZ'); \
		LDFLAGS="-s -w -X '$${PKG}/cmd.Version=${VERSION}' -X '$${PKG}/cmd.Commit=$${COMMIT}' -X '$${PKG}/cmd.BuildDate=$${BUILD_DATE}'"; \
		echo "==> Installing WHAM! to your GOBIN..."; \
		$(GO) install -ldflags "$${LDFLAGS}"; \
	}

# Clean up the built binary.
clean: ## Clean up build artifacts
	@echo "==> Cleaning..."
	@rm -f wham

# A self-documenting help target.
help: ## Show this help message
	@echo "Usage: make [target]"
	@echo ""
	@echo "Targets:"
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-20s\033[0m %s\n", $$1, $$2}'

# Phony targets are not files.
.PHONY: all build install clean help
