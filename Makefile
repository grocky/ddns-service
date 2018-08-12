PROJECT_NAME=ddns-service
APP_VERSION := $(shell cat package.json | jq '.version' -r)

BUCKET_NAME=grocky-ddns-service
APP_ARCHIVE=$(PROJECT_NAME)-$(APP_VERSION).zip

BUILD_DIR=dist

##### Targets ######

build: ensure-build $(BUILD_DIR)/$(APP_ARCHIVE)

$(BUILD_DIR)/$(APP_ARCHIVE):
	zip $@ index.js

ensure-build: $(BUILD_DIR)

$(BUILD_DIR):
	@mkdir -p $@

s3-bucket:
	aws s3api create-bucket --region=us-east-1 --bucket=$(BUCKET_NAME)
	touch s3-bucket

deploy: s3-bucket
	aws s3 cp example.zip s3://$(BUCKET_NAME)/v$(APP_VERSION)/example.zip
	touch deploy

clean:
	@rm -rf $(BUILD_DIR)

