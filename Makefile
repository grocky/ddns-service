PROJECT_NAME=ddns-service
APP_VERSION := $(shell git describe --always --long --dirty)

BUCKET_NAME=grocky-services
APP_ARCHIVE=$(PROJECT_NAME)-$(APP_VERSION).zip

BUILD_DIR=bin
BUILD_BIN=${BUILD_DIR}/${PROJECT_NAME}_linux_${APP_VERSION}

LOCAL_OS := $(shell uname | tr '[:upper:]' '[:lower:]')

##### Targets ######

clean:
	@rm -rf $(BUILD_DIR)

_s3-bucket: $(BUILD_DIR)/s3-bucket
$(BUILD_DIR)/s3-bucket:
	aws s3api create-bucket --region=us-east-1 --bucket=$(BUCKET_NAME)
	touch $(BUILD_DIR)/s3-bucket

_ensure-package: $(BUILD_DIR)
$(BUILD_DIR):
	@mkdir -p $@

publish: package _s3-bucket _upload-archive
_upload-archive: $(BUILD_DIR)/publish-$(APP_VERSION)
$(BUILD_DIR)/publish-$(APP_VERSION):
	@aws s3 cp $(BUILD_DIR)/$(APP_ARCHIVE) s3://$(BUCKET_NAME)/$(APP_ARCHIVE)

.PHONY: deploy
deploy: publish
	cd terraform; terraform apply -var 'app_version=$(APP_VERSION)' -auto-approve

### Go Impl ###

install:
	go get github.com/aws/aws-lambda-go/events
	go get github.com/aws/aws-lambda-go/lambda
	go get github.com/stretchr/testify/assert

.PHONY: build-local ${BUILD_DIR}/${PROJECT_NAME}_${LOCAL_OS}
build-local: ${BUILD_DIR}/${PROJECT_NAME}_${LOCAL_OS}_${APP_VERSION}
${BUILD_DIR}/${PROJECT_NAME}_${LOCAL_OS}_${APP_VERSION}:
	env GOOS=${LOCAL_OS} GOARCH=amd64 go build -o ${BUILD_DIR}/${PROJECT_NAME}_${LOCAL_OS} main.go

.PHONY: build ${BUILD_DIR}/${PROJECT_NAME}_linux
build: ${BUILD_BIN}
${BUILD_BIN}:
	env GOOS=linux GOARCH=amd64 go build -o $@ main.go

package: _ensure-package $(BUILD_DIR)/$(APP_ARCHIVE)
$(BUILD_DIR)/$(APP_ARCHIVE):
	zip -j $@ ${BUILD_BIN}
