PROJECT_NAME=ddns-service
APP_VERSION := $(shell cat package.json | jq '.version' -r)

BUCKET_NAME=grocky-services
APP_ARCHIVE=$(PROJECT_NAME)-$(APP_VERSION).zip

BUILD_DIR=dist

##### Targets ######

.PHONY: build
build: _ensure-build _archive-source

_archive-source: $(BUILD_DIR)/$(APP_ARCHIVE)
$(BUILD_DIR)/$(APP_ARCHIVE):
	zip -r $@ index.js src node_modules

_ensure-build: $(BUILD_DIR)
$(BUILD_DIR):
	@mkdir -p $@

_s3-bucket: $(BUILD_DIR)/s3-bucket
$(BUILD_DIR)/s3-bucket:
	aws s3api create-bucket --region=us-east-1 --bucket=$(BUCKET_NAME)
	touch $(BUILD_DIR)/s3-bucket

publish: build _s3-bucket _upload-archive

_upload-archive: $(BUILD_DIR)/publish-$(APP_VERSION)
$(BUILD_DIR)/publish-$(APP_VERSION):
	aws s3 cp $(BUILD_DIR)/$(APP_ARCHIVE) s3://$(BUCKET_NAME)/$(APP_ARCHIVE)
	@touch $@

.PHONY: deploy
deploy: build publish
	cd terraform; terraform apply -var 'app_version=$(APP_VERSION)' -auto-approve

clean:
	@rm -rf $(BUILD_DIR)

