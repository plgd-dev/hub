SHELL = /bin/bash
SERVICE_NAME = $(notdir $(CURDIR))
LATEST_TAG ?= vnext
VERSION_TAG ?= $(LATEST_TAG)-$(shell git rev-parse --short=7 --verify HEAD)
GOPATH ?= $(shell go env GOPATH)
WORKING_DIRECTORY := $(shell pwd)

default: build

define build-docker-image
	docker build \
		--network=host \
		--tag plgd/$(SERVICE_NAME):$(VERSION_TAG) \
		--tag plgd/$(SERVICE_NAME):$(LATEST_TAG) \
		--target $(1) \
		.
endef

build-servicecontainer:
	$(call build-docker-image,service)

build: build-servicecontainer

push: build-servicecontainer
	docker push plgd/$(SERVICE_NAME):$(VERSION_TAG)
	docker push plgd/$(SERVICE_NAME):$(LATEST_TAG)

GOOGLEAPIS_MODULE_PATH := $(shell find $(GOPATH)/pkg/mod/github.com/googleapis -mindepth 1 -maxdepth 1 -iname 'googleapi*' -print -quit)
GRPCGATEWAY_MODULE_PATH := $(shell go list -m -f '{{.Dir}}' github.com/grpc-ecosystem/grpc-gateway/v2 | head -1)

proto/generate:
	protoc -I=. -I=$(GOPATH)/src --go_out=$(GOPATH)/src $(WORKING_DIRECTORY)/pb/clientConfiguration.proto
	protoc -I=. -I=$(GOPATH)/src --go_out=$(GOPATH)/src $(WORKING_DIRECTORY)/pb/devices.proto
	protoc -I=. -I=$(GOPATH)/src --go_out=$(GOPATH)/src $(WORKING_DIRECTORY)/pb/events.proto
	protoc -I=. -I=$(GOPATH)/src --go_out=$(GOPATH)/src $(WORKING_DIRECTORY)/pb/getDevicesMetadata.proto
	protoc -I=. -I=$(GOPATH)/src --go_out=$(GOPATH)/src $(WORKING_DIRECTORY)/pb/getPendingCommands.proto
	protoc -I=. -I=$(GOPATH)/src --go_out=$(GOPATH)/src $(WORKING_DIRECTORY)/pb/updateDeviceMetadata.proto
	protoc -I=. -I=$(GOPATH)/src -I=$(GOOGLEAPIS_MODULE_PATH) -I=$(GRPCGATEWAY_MODULE_PATH) --go-grpc_out=$(GOPATH)/src $(WORKING_DIRECTORY)/pb/service.proto
	protoc -I=. -I=$(GOPATH)/src -I=$(GOOGLEAPIS_MODULE_PATH) -I=$(GRPCGATEWAY_MODULE_PATH) --openapiv2_out=$(GOPATH)/src \
		--openapiv2_opt logtostderr=true \
		$(WORKING_DIRECTORY)/pb/service.proto
	protoc -I=. -I=$(GOPATH)/src -I=$(GOOGLEAPIS_MODULE_PATH) -I=$(GRPCGATEWAY_MODULE_PATH) --grpc-gateway_out $(GOPATH)/src \
		--grpc-gateway_opt logtostderr=true \
		--grpc-gateway_opt paths=source_relative \
		$(WORKING_DIRECTORY)/pb/service.proto

.PHONY: build-servicecontainer build push proto/generate


