GREEN  := $(shell tput -Txterm setaf 2)
NC     := $(shell tput -Txterm sgr0)

PROJECT_NAME := ddns-service
APP_VERSION  := $(shell git describe --always --long --dirty)

BUILD_DIR    := bin
BUCKET_NAME  := grocky-services
APP_ARCHIVE  := $(PROJECT_NAME)-$(APP_VERSION).zip

# Source file dependencies
LAMBDA_SOURCES := $(shell find cmd/ddns-service-lambda internal pkg -name "*.go")
PUBIP_SOURCES  := $(shell find cmd/pubip pkg -name "*.go")

help: ## Print this help message
	@awk -F ':|##' '/^[^\t].+?:.*?##/ { printf "${GREEN}%-20s${NC}%s\n", $$1, $$NF }' $(MAKEFILE_LIST) | \
		sort

# =============================================================================
# Build
# =============================================================================

$(BUILD_DIR):
	@mkdir -p $@

.PHONY=clean
clean: ## Clean build artifacts
	rm -rf $(BUILD_DIR)

# --- Lambda ---

$(BUILD_DIR)/$(PROJECT_NAME)-lambda: $(LAMBDA_SOURCES) | $(BUILD_DIR)
	GOOS=linux GOARCH=amd64 go build -o $@ ./cmd/ddns-service-lambda

.PHONY=build-lambda
build-lambda: $(BUILD_DIR)/$(PROJECT_NAME)-lambda ## Build the Lambda binary (linux/amd64)

# --- pubip CLI ---

$(BUILD_DIR)/pubip: $(PUBIP_SOURCES) | $(BUILD_DIR)
	go build -o $@ ./cmd/pubip

.PHONY=build-pubip
build-pubip: $(BUILD_DIR)/pubip ## Build the pubip CLI

$(BUILD_DIR)/pubip-debug: $(PUBIP_SOURCES) | $(BUILD_DIR)
	go build -tags=debug -o $@ ./cmd/pubip

.PHONY=build-pubip-debug
build-pubip-debug: $(BUILD_DIR)/pubip-debug ## Build the pubip CLI with debug profiling

.PHONY=build
build: build-lambda build-pubip ## Build all binaries

# =============================================================================
# Test
# =============================================================================

.PHONY=test
test: ## Run all tests
	go test ./...

.PHONY=test-endpoint
test-endpoint: ## Test the deployed endpoint with a GET request
	curl -s https://ddns.rockygray.com/public-ip | jq .

# =============================================================================
# Package & Deploy
# =============================================================================

$(BUILD_DIR)/$(APP_ARCHIVE): $(BUILD_DIR)/$(PROJECT_NAME)-lambda
	zip -j $@ $<

.PHONY=package
package: $(BUILD_DIR)/$(APP_ARCHIVE) ## Package Lambda binary into a zip archive

$(BUILD_DIR)/.s3-bucket:
	aws s3api create-bucket --region=us-east-1 --bucket=$(BUCKET_NAME)
	@touch $@

$(BUILD_DIR)/.published-$(APP_VERSION): $(BUILD_DIR)/$(APP_ARCHIVE) $(BUILD_DIR)/.s3-bucket
	aws s3 cp $< s3://$(BUCKET_NAME)/$(APP_ARCHIVE)
	@touch $@

.PHONY=publish
publish: $(BUILD_DIR)/.published-$(APP_VERSION) ## Upload Lambda archive to S3

.PHONY=deploy
deploy: publish ## Publish and deploy Lambda via Terraform
	cd terraform; terraform apply -var 'app_version=$(APP_VERSION)' -auto-approve

.PHONY=invoke
invoke: ## Invoke the Lambda with test-payload.json
	@mkdir -p logs
	aws lambda invoke --region=us-east-1 --function-name=$(PROJECT_NAME) --payload file://test-payload.json logs/out.txt
	@cat logs/out.txt | jq .

# =============================================================================
# Terraform
# =============================================================================

.PHONY=tf-init
tf-init: ## Initialize Terraform
	cd terraform; terraform init

.PHONY=tf-plan
tf-plan: ## Plan Terraform changes
	cd terraform; terraform plan -var 'app_version=$(APP_VERSION)'

.PHONY=tf-apply
tf-apply: ## Apply Terraform changes
	cd terraform; terraform apply -var 'app_version=$(APP_VERSION)'
