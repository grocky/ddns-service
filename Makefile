GREEN  := $(shell tput -Txterm setaf 2)
NC     := $(shell tput -Txterm sgr0)

PROJECT_NAME := ddns-service

.DEFAULT_GOAL := help

# Source file dependencies
LAMBDA_SOURCES := $(shell find cmd/ddns-service-lambda internal pkg -name "*.go")
PUBIP_SOURCES  := $(shell find cmd/pubip pkg -name "*.go")
ADMIN_SOURCES  := $(shell find cmd/ddns-admin internal -name "*.go")

help: ## Print this help message
	@awk -F ':|##' '/^[^\t].+?:.*?##/ { printf "${GREEN}%-20s${NC}%s\n", $$1, $$NF }' $(MAKEFILE_LIST) | sort

# =============================================================================
# Build
# =============================================================================

.PHONY: clean
clean: ## Clean build artifacts
	rm -rf bin scripts/dist

# --- Lambda ---

scripts/dist/ddns-service.zip: $(LAMBDA_SOURCES)
	./scripts/build-lambda.sh

.PHONY: build-lambda
build-lambda: scripts/dist/ddns-service.zip ## Build the Lambda deployment package

# --- pubip CLI ---

bin/pubip: $(PUBIP_SOURCES)
	@mkdir -p bin
	go build -o $@ ./cmd/pubip

.PHONY: build-pubip
build-pubip: bin/pubip ## Build the pubip CLI

bin/pubip-debug: $(PUBIP_SOURCES)
	@mkdir -p bin
	go build -tags=debug -o $@ ./cmd/pubip

.PHONY: build-pubip-debug
build-pubip-debug: bin/pubip-debug ## Build the pubip CLI with debug profiling

# --- ddns-admin CLI ---

bin/ddns-admin: $(ADMIN_SOURCES)
	@mkdir -p bin
	go build -o $@ ./cmd/ddns-admin

.PHONY: build-admin
build-admin: bin/ddns-admin ## Build the ddns-admin CLI

.PHONY: build
build: build-lambda build-pubip build-admin ## Build all artifacts

# =============================================================================
# Test
# =============================================================================

.PHONY: test
test: ## Run all tests
	go test ./...

.PHONY: test-endpoint
test-endpoint: ## Test the deployed endpoint with a GET request
	curl -s https://ddns.grocky.net/public-ip | jq .

# =============================================================================
# Deploy
# =============================================================================

.PHONY: deploy
deploy: build-lambda ## Build and deploy Lambda via Terraform
	cd terraform && terraform apply -auto-approve

.PHONY: invoke
invoke: ## Invoke the Lambda with test-payload.json
	@mkdir -p logs
	aws lambda invoke --region=us-east-1 --function-name=$(PROJECT_NAME) --payload file://test-payload.json logs/out.txt
	@cat logs/out.txt | jq .

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
