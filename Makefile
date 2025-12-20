GREEN  := $(shell tput -Txterm setaf 2)
NC     := $(shell tput -Txterm sgr0)

PROJECT_NAME := ddns-service

.DEFAULT_GOAL := help

# Source file dependencies
LAMBDA_SOURCES := $(shell find cmd/ddns-service-lambda internal pkg -name "*.go")
CLIENT_SOURCES := $(shell find cmd/ddns-client internal/client internal/state pkg -name "*.go")
ADMIN_SOURCES  := $(shell find cmd/ddns-admin internal -name "*.go")

help: ## Print this help message
	@awk -F ':|##' '/^[^\t].+?:.*?##/ { printf "${GREEN}%-20s${NC}%s\n", $$1, $$NF }' $(MAKEFILE_LIST) | sort

# =============================================================================
# Build
# =============================================================================

.PHONY: clean
clean: ## Clean build artifacts
	rm -rf bin dist

# --- Lambda ---

dist/ddns-service.zip: $(LAMBDA_SOURCES)
	./scripts/build-lambda.sh

.PHONY: build-lambda
build-lambda: dist/ddns-service.zip ## Build the Lambda deployment package

# --- ddns-client CLI ---

bin/ddns-client: $(CLIENT_SOURCES)
	@mkdir -p bin
	go build -o $@ ./cmd/ddns-client

.PHONY: build-client
build-client: bin/ddns-client ## Build the ddns-client CLI

bin/ddns-client-debug: $(CLIENT_SOURCES)
	@mkdir -p bin
	go build -tags=debug -o $@ ./cmd/ddns-client

.PHONY: build-client-debug
build-client-debug: bin/ddns-client-debug ## Build the ddns-client CLI with debug profiling

# --- ddns-admin CLI ---

bin/ddns-admin: $(ADMIN_SOURCES)
	@mkdir -p bin
	go build -o $@ ./cmd/ddns-admin

.PHONY: build-admin
build-admin: bin/ddns-admin ## Build the ddns-admin CLI

.PHONY: build
build: build-lambda build-client build-admin ## Build all artifacts

# =============================================================================
# Test
# =============================================================================

.PHONY: test
test: ## Run all tests
	go test ./...

# =============================================================================
# Deploy
# =============================================================================

.PHONY: deploy
deploy: build-lambda ## Build and deploy Lambda via Terraform
	cd terraform && terraform apply -auto-approve

# =============================================================================
# Terraform
# =============================================================================

.PHONY: tf-init
tf-init: ## Initialize Terraform
	cd terraform && terraform init

.PHONY: tf-plan
tf-plan: build-lambda ## Plan Terraform changes
	cd terraform && terraform plan

.PHONY: tf-apply
tf-apply: build-lambda ## Apply Terraform changes
	cd terraform && terraform apply
