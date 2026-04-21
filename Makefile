# ====================================================================================
# Crossplane Provider Slack - Family-Scoped Build System
# ====================================================================================

PROJECT_NAME := crossplane-provider-slack
MODULE := github.com/avodah-inc/crossplane-provider-slack

# Registry and image configuration
REGISTRY := ghcr.io/starlightromero
BASE_IMAGE := gcr.io/distroless/static:nonroot

# Binary names
FAMILY_BINARY := family-provider
CONVERSATION_BINARY := provider-conversation
USERGROUP_BINARY := provider-usergroup

# Image names
FAMILY_IMAGE := provider-family-slack
CONVERSATION_IMAGE := provider-slack-conversation
USERGROUP_IMAGE := provider-slack-usergroup

# Version (override with VERSION=vX.Y.Z)
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")

# Go build settings
GOOS ?= linux
GOARCH ?= amd64
CGO_ENABLED := 0
GO_LDFLAGS := -s -w -X main.Version=$(VERSION)

# Output directory
OUTPUT_DIR := _output
BIN_DIR := $(OUTPUT_DIR)/bin/$(GOOS)_$(GOARCH)
XPKG_DIR := $(OUTPUT_DIR)/xpkg

# ====================================================================================
# Targets
# ====================================================================================

.PHONY: all
all: generate build

# ------------------------------------------------------------------------------------
# Code Generation
# ------------------------------------------------------------------------------------

.PHONY: generate
generate: ## Run code generation (angryjet, controller-gen)
	@echo "==> Generating code..."
	go generate ./apis/...

# ------------------------------------------------------------------------------------
# Build
# ------------------------------------------------------------------------------------

.PHONY: build
build: build.family build.conversation build.usergroup ## Build all provider binaries

.PHONY: build.family
build.family: ## Build the family provider binary
	@echo "==> Building $(FAMILY_BINARY)..."
	@mkdir -p $(BIN_DIR)
	CGO_ENABLED=$(CGO_ENABLED) GOOS=$(GOOS) GOARCH=$(GOARCH) \
		go build -ldflags "$(GO_LDFLAGS)" -o $(BIN_DIR)/$(FAMILY_BINARY) ./cmd/family-provider/

.PHONY: build.conversation
build.conversation: ## Build the conversation provider binary
	@echo "==> Building $(CONVERSATION_BINARY)..."
	@mkdir -p $(BIN_DIR)
	CGO_ENABLED=$(CGO_ENABLED) GOOS=$(GOOS) GOARCH=$(GOARCH) \
		go build -ldflags "$(GO_LDFLAGS)" -o $(BIN_DIR)/$(CONVERSATION_BINARY) ./cmd/provider-conversation/

.PHONY: build.usergroup
build.usergroup: ## Build the usergroup provider binary
	@echo "==> Building $(USERGROUP_BINARY)..."
	@mkdir -p $(BIN_DIR)
	CGO_ENABLED=$(CGO_ENABLED) GOOS=$(GOOS) GOARCH=$(GOARCH) \
		go build -ldflags "$(GO_LDFLAGS)" -o $(BIN_DIR)/$(USERGROUP_BINARY) ./cmd/provider-usergroup/

# ------------------------------------------------------------------------------------
# Docker
# ------------------------------------------------------------------------------------

.PHONY: docker-build
docker-build: docker-build.family docker-build.conversation docker-build.usergroup ## Build all container images

.PHONY: docker-build.family
docker-build.family: ## Build the family provider container image
	@echo "==> Building Docker image $(REGISTRY)/$(FAMILY_IMAGE):$(VERSION)..."
	docker build -f Dockerfile.family -t $(REGISTRY)/$(FAMILY_IMAGE):$(VERSION) .

.PHONY: docker-build.conversation
docker-build.conversation: ## Build the conversation provider container image
	@echo "==> Building Docker image $(REGISTRY)/$(CONVERSATION_IMAGE):$(VERSION)..."
	docker build -f Dockerfile.conversation -t $(REGISTRY)/$(CONVERSATION_IMAGE):$(VERSION) .

.PHONY: docker-build.usergroup
docker-build.usergroup: ## Build the usergroup provider container image
	@echo "==> Building Docker image $(REGISTRY)/$(USERGROUP_IMAGE):$(VERSION)..."
	docker build -f Dockerfile.usergroup -t $(REGISTRY)/$(USERGROUP_IMAGE):$(VERSION) .

.PHONY: docker-push
docker-push: ## Push all container images to the registry
	@echo "==> Pushing all images..."
	docker push $(REGISTRY)/$(FAMILY_IMAGE):$(VERSION)
	docker push $(REGISTRY)/$(CONVERSATION_IMAGE):$(VERSION)
	docker push $(REGISTRY)/$(USERGROUP_IMAGE):$(VERSION)

# ------------------------------------------------------------------------------------
# Crossplane Packages
# ------------------------------------------------------------------------------------

.PHONY: xpkg-build
xpkg-build: xpkg-build.family xpkg-build.conversation xpkg-build.usergroup ## Build all Crossplane packages

.PHONY: xpkg-build.family
xpkg-build.family: ## Build the family provider Crossplane package
	@echo "==> Building xpkg $(FAMILY_IMAGE)..."
	@mkdir -p $(XPKG_DIR)
	crossplane xpkg build \
		--package-root=package/family \
		--embed-runtime-image=$(REGISTRY)/$(FAMILY_IMAGE):$(VERSION) \
		-o $(XPKG_DIR)/$(FAMILY_IMAGE).xpkg

.PHONY: xpkg-build.conversation
xpkg-build.conversation: ## Build the conversation provider Crossplane package
	@echo "==> Building xpkg $(CONVERSATION_IMAGE)..."
	@mkdir -p $(XPKG_DIR)
	crossplane xpkg build \
		--package-root=package/conversation \
		--embed-runtime-image=$(REGISTRY)/$(CONVERSATION_IMAGE):$(VERSION) \
		-o $(XPKG_DIR)/$(CONVERSATION_IMAGE).xpkg

.PHONY: xpkg-build.usergroup
xpkg-build.usergroup: ## Build the usergroup provider Crossplane package
	@echo "==> Building xpkg $(USERGROUP_IMAGE)..."
	@mkdir -p $(XPKG_DIR)
	crossplane xpkg build \
		--package-root=package/usergroup \
		--embed-runtime-image=$(REGISTRY)/$(USERGROUP_IMAGE):$(VERSION) \
		-o $(XPKG_DIR)/$(USERGROUP_IMAGE).xpkg

.PHONY: xpkg-push
xpkg-push: ## Push all Crossplane packages to the registry
	@echo "==> Pushing all xpkgs..."
	crossplane xpkg push $(REGISTRY)/$(FAMILY_IMAGE):$(VERSION) \
		-f $(XPKG_DIR)/$(FAMILY_IMAGE).xpkg
	crossplane xpkg push $(REGISTRY)/$(CONVERSATION_IMAGE):$(VERSION) \
		-f $(XPKG_DIR)/$(CONVERSATION_IMAGE).xpkg
	crossplane xpkg push $(REGISTRY)/$(USERGROUP_IMAGE):$(VERSION) \
		-f $(XPKG_DIR)/$(USERGROUP_IMAGE).xpkg

# ------------------------------------------------------------------------------------
# Test and Lint
# ------------------------------------------------------------------------------------

.PHONY: test
test: ## Run all tests
	go test ./...

.PHONY: lint
lint: ## Run linters (placeholder)
	@echo "==> Running linters..."
	@echo "TODO: Configure golangci-lint"

# ------------------------------------------------------------------------------------
# Clean
# ------------------------------------------------------------------------------------

.PHONY: clean
clean: ## Remove build artifacts
	rm -rf $(OUTPUT_DIR)

# ------------------------------------------------------------------------------------
# Help
# ------------------------------------------------------------------------------------

.PHONY: help
help: ## Show this help
	@grep -E '^[a-zA-Z_.-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | \
		awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-30s\033[0m %s\n", $$1, $$2}'
