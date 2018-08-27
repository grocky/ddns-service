PROJECT_NAME=ddns-service
APP_VERSION := $(shell cat package-js.json | jq '.version' -r)

BUCKET_NAME=grocky-services
APP_ARCHIVE=$(PROJECT_NAME)-$(APP_VERSION).zip

BUILD_DIR=bin

LOCAL_OS := $(shell uname | tr '[:upper:]' '[:lower:]')

##### Targets ######

_s3-bucket: $(BUILD_DIR)/s3-bucket
$(BUILD_DIR)/s3-bucket:
	aws s3api create-bucket --region=us-east-1 --bucket=$(BUCKET_NAME)
	touch $(BUILD_DIR)/s3-bucket

_ensure-package-js: $(BUILD_DIR)
$(BUILD_DIR):
	@mkdir -p $@

### Go Impl ###
.PHONY: build-local ${BUILD_DIR}/${PROJECT_NAME}_${LOCAL_OS}
build-local: ${BUILD_DIR}/${PROJECT_NAME}_${LOCAL_OS}
${BUILD_DIR}/${PROJECT_NAME}_${LOCAL_OS}:
	env GOOS=${LOCAL_OS} GOARCH=amd64 go build -o ${BUILD_DIR}/${PROJECT_NAME}_${LOCAL_OS} main.go

.PHONY: build ${BUILD_DIR}/${PROJECT_NAME}_linux
build: ${BUILD_DIR}/${PROJECT_NAME}_linux
${BUILD_DIR}/${PROJECT_NAME}_linux:
	env GOOS=linux GOARCH=amd64 go build -o ${BUILD_DIR}/${PROJECT_NAME}_linux main.go

### NODE Impl ###
.PHONY: package-js
package-js: _ensure-package _archive-source

_archive-source: $(BUILD_DIR)/$(APP_ARCHIVE)
$(BUILD_DIR)/$(APP_ARCHIVE):
	zip -r $@ index.js src node_modules

publish-js: package-js _s3-bucket _upload-archive

_upload-archive: $(BUILD_DIR)/publish-js-$(APP_VERSION)
$(BUILD_DIR)/publish-js-$(APP_VERSION):
	aws s3 cp $(BUILD_DIR)/$(APP_ARCHIVE) s3://$(BUCKET_NAME)/$(APP_ARCHIVE)
	@touch $@

.PHONY: deploy
deploy-js: package-js publish-js
	cd terraform; terraform apply -var 'app_version=$(APP_VERSION)' -auto-approve

clean:
	@rm -rf $(BUILD_DIR)

