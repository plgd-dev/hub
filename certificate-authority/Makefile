SHELL = /bin/bash
SERVICE_NAME = $(notdir $(CURDIR))
LATEST_TAG ?= vnext
BRANCH_TAG ?= $(shell git rev-parse --abbrev-ref HEAD | sed 's/[^a-zA-Z0-9]/-/g')
ifneq ($(BRANCH_TAG),main)
	LATEST_TAG = $(BRANCH_TAG)
endif
VERSION_TAG ?= $(LATEST_TAG)-$(shell git rev-parse --short=7 --verify HEAD)
GOPATH ?= $(shell go env GOPATH)
WORKING_DIRECTORY := $(shell pwd)
REPOSITORY_DIRECTORY := $(shell cd .. && pwd)
BUILD_COMMIT_DATE ?= $(shell date -u +%FT%TZ --date=@`git show --format='%ct' HEAD --quiet`)
BUILD_SHORT_COMMIT ?= $(shell git show --format=%h HEAD --quiet)
BUILD_DATE ?= $(shell date -u +%FT%TZ)
BUILD_VERSION ?= $(shell git tag --sort version:refname | tail -1 | sed -e "s/^v//")

default: build

define build-docker-image
	cd .. && \
		mkdir -p .tmp/docker/$(SERVICE_NAME) && \
		awk '{gsub("@NAME@","$(SERVICE_NAME)")} {gsub("@DIRECTORY@","$(SERVICE_NAME)")} {print}' tools/docker/Dockerfile.in > .tmp/docker/$(SERVICE_NAME)/Dockerfile && \
		docker build \
		--network=host \
		--tag ghcr.io/plgd-dev/hub/$(SERVICE_NAME):$(VERSION_TAG) \
		--tag ghcr.io/plgd-dev/hub/$(SERVICE_NAME):$(LATEST_TAG) \
		--tag ghcr.io/plgd-dev/hub/$(SERVICE_NAME):$(BRANCH_TAG) \
		--build-arg COMMIT_DATE="$(BUILD_COMMIT_DATE)" \
		--build-arg SHORT_COMMIT="$(BUILD_SHORT_COMMIT)" \
		--build-arg DATE="$(BUILD_DATE)" \
		--build-arg VERSION="$(BUILD_VERSION)" \
		--target $(1) \
		-f .tmp/docker/$(SERVICE_NAME)/Dockerfile \
		.
endef

build-servicecontainer:
	$(call build-docker-image,service)

build: build-servicecontainer

push: build-servicecontainer
	docker push plgd/$(SERVICE_NAME):$(VERSION_TAG)
	docker push plgd/$(SERVICE_NAME):$(LATEST_TAG)

GOOGLEAPIS_PATH := $(REPOSITORY_DIRECTORY)/dependency/googleapis
GRPCGATEWAY_MODULE_PATH := $(shell go list -m -f '{{.Dir}}' github.com/grpc-ecosystem/grpc-gateway/v2 | head -1)

proto/generate:
	protoc -I=. -I=$(REPOSITORY_DIRECTORY) -I=$(GOPATH)/src --go_out=$(GOPATH)/src $(WORKING_DIRECTORY)/pb/cert.proto
	protoc -I=. -I=$(REPOSITORY_DIRECTORY) -I=$(GOPATH)/src --go_out=$(GOPATH)/src $(WORKING_DIRECTORY)/pb/signingRecords.proto
	protoc -I=. -I=$(REPOSITORY_DIRECTORY) -I=$(GOPATH)/src -I=$(GOOGLEAPIS_PATH) -I=$(GRPCGATEWAY_MODULE_PATH) --go_out=$(GOPATH)/src $(WORKING_DIRECTORY)/pb/service.proto
	protoc-go-inject-tag -input=$(WORKING_DIRECTORY)/pb/signingRecords.pb.go
	protoc -I=. -I=$(REPOSITORY_DIRECTORY) -I=$(GOPATH)/src -I=$(GOOGLEAPIS_PATH) -I=$(GRPCGATEWAY_MODULE_PATH) --openapiv2_out=$(REPOSITORY_DIRECTORY) \
		--openapiv2_opt logtostderr=true \
		$(WORKING_DIRECTORY)/pb/service.proto
	protoc -I=. -I=$(REPOSITORY_DIRECTORY) -I=$(GOPATH)/src -I=$(GOOGLEAPIS_PATH) -I=$(GRPCGATEWAY_MODULE_PATH) --grpc-gateway_out=$(REPOSITORY_DIRECTORY) \
		--grpc-gateway_opt logtostderr=true \
		--grpc-gateway_opt paths=source_relative \
		$(WORKING_DIRECTORY)/pb/service.proto
	protoc -I=. -I=$(REPOSITORY_DIRECTORY) -I=$(GOPATH)/src -I=$(GOOGLEAPIS_PATH) -I=$(GRPCGATEWAY_MODULE_PATH) --go-grpc_out=$(GOPATH)/src \
		$(WORKING_DIRECTORY)/pb/service.proto
	protoc  -I=. -I=$(REPOSITORY_DIRECTORY) -I=$(GOPATH)/src -I=$(GOOGLEAPIS_PATH) -I=$(GRPCGATEWAY_MODULE_PATH) --doc_out=$(WORKING_DIRECTORY)/pb --doc_opt=markdown,README.md $(WORKING_DIRECTORY)/pb/*.proto
	protoc  -I=. -I=$(REPOSITORY_DIRECTORY) -I=$(GOPATH)/src -I=$(GOOGLEAPIS_PATH) -I=$(GRPCGATEWAY_MODULE_PATH) --doc_out=$(WORKING_DIRECTORY)/pb --doc_opt=html,doc.html $(WORKING_DIRECTORY)/pb/*.proto

.PHONY: build-servicecontainer build push proto/generate






