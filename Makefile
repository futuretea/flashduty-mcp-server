# FlashDuty MCP Server Makefile

.DEFAULT_GOAL := help

PACKAGE = $(shell go list -m)
GIT_COMMIT_HASH = $(shell git rev-parse HEAD)
GIT_VERSION = $(shell git describe --tags --always --dirty)
BUILD_TIME = $(shell date -u '+%Y-%m-%dT%H:%M:%SZ')
BINARY_NAME = flashduty-mcp-server
LD_FLAGS = -s -w \
	-X '$(PACKAGE)/pkg/core/version.Version=$(GIT_VERSION)' \
	-X '$(PACKAGE)/pkg/core/version.GitCommit=$(GIT_COMMIT_HASH)' \
	-X '$(PACKAGE)/pkg/core/version.BuildDate=$(BUILD_TIME)'
COMMON_BUILD_ARGS = -ldflags "$(LD_FLAGS)"

GOLANGCI_LINT = $(shell pwd)/_output/tools/bin/golangci-lint
GOLANGCI_LINT_VERSION ?= v2.2.2

.PHONY: help
help: ## Display this help
	@awk 'BEGIN {FS = ":.*##"; printf "\nUsage:\n  make \033[36m<target>\033[0m\n"} /^[a-zA-Z_0-9\/\.-]+:.*?##/ { printf "  \033[36m%-21s\033[0m %s\n", $$1, $$2 } /^##@/ { printf "\n\033[1m%s\033[0m\n", substr($$0, 5) } ' $(MAKEFILE_LIST)

.PHONY: clean
clean: ## Clean up all build artifacts
	rm -rf '$(BINARY_NAME)'

.PHONY: build
build: tidy ## Build the project
	CGO_ENABLED=0 go build $(COMMON_BUILD_ARGS) -o $(BINARY_NAME) ./cmd/flashduty-mcp-server

.PHONY: test
test: ## Run the tests
	go test -count=1 -v ./...

.PHONY: format
format: ## Format the code
	go fmt ./...

.PHONY: tidy
tidy: ## Tidy up the go modules
	go mod tidy

.PHONY: golangci-lint
golangci-lint: ## Download and install golangci-lint if not already installed
		@[ -f $(GOLANGCI_LINT) ] || { \
    	set -e ;\
    	curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $(shell dirname $(GOLANGCI_LINT)) $(GOLANGCI_LINT_VERSION) ;\
    	}

.PHONY: lint
lint: golangci-lint ## Lint the code
	$(GOLANGCI_LINT) run --verbose --print-resources-usage

.PHONY: version
version: ## Show version information
	./$(BINARY_NAME) version
