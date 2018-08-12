PROJECT_NAME=ddns-service
APP_VERSION := $(shell cat package.json | jq '.version' -r)

BUCKET_NAME=grocky-services
APP_ARCHIVE=$(PROJECT_NAME)-$(APP_VERSION).zip

BUILD_DIR=dist

##### Targets ######

build: _ensure-build _archive-source

_archive-source: $(BUILD_DIR)/$(APP_ARCHIVE)
$(BUILD_DIR)/$(APP_ARCHIVE):
	zip $@ index.js

_ensure-build: $(BUILD_DIR)
$(BUILD_DIR):
	@mkdir -p $@

_s3-bucket: $(BUILD_DIR)/s3-bucket
$(BUILD_DIR)/s3-bucket:
	aws s3api create-bucket --region=us-east-1 --bucket=$(BUCKET_NAME)
	touch $(BUILD_DIR)/s3-bucket

deploy: build _s3-bucket _upload-archive

_upload-archive: $(BUILD_DIR)/deploy-$(APP_VERSION)
$(BUILD_DIR)/deploy-$(APP_VERSION):
	aws s3 cp $(BUILD_DIR)/$(APP_ARCHIVE) s3://$(BUCKET_NAME)/$(APP_ARCHIVE)
	@touch $@

clean:
	@rm -rf $(BUILD_DIR)

