SHELL = /bin/bash
SERVICE_NAME = $(notdir $(CURDIR))
LATEST_TAG ?= vnext
VERSION_TAG ?= $(LATEST_TAG)-$(shell git rev-parse --short=7 --verify HEAD)
GOPATH ?= $(shell go env GOPATH)
WORKING_DIRECTORY := $(shell pwd)
REPOSITORY_DIRECTORY := $(shell cd .. && pwd)

default: build

define build-docker-image
	cd .. && docker build \
		--network=host \
		--build-arg DIRECTORY=$(SERVICE_NAME) \
		--build-arg NAME=$(SERVICE_NAME) \
		--tag plgd/$(SERVICE_NAME):$(VERSION_TAG) \
		--tag plgd/$(SERVICE_NAME):$(LATEST_TAG) \
		--target $(1) \
		-f tools/docker/Dockerfile \
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
	protoc -I=. -I=$(REPOSITORY_DIRECTORY) -I=$(GOPATH)/src --go_out=$(GOPATH)/src $(WORKING_DIRECTORY)/pb/hubConfiguration.proto
	protoc-go-inject-tag -input=$(WORKING_DIRECTORY)/pb/hubConfiguration.pb.go
	protoc -I=. -I=$(REPOSITORY_DIRECTORY) -I=$(GOPATH)/src --go_out=$(GOPATH)/src $(WORKING_DIRECTORY)/pb/devices.proto
	protoc -I=. -I=$(REPOSITORY_DIRECTORY) -I=$(GOPATH)/src --go_out=$(GOPATH)/src $(WORKING_DIRECTORY)/pb/events.proto
	protoc -I=. -I=$(REPOSITORY_DIRECTORY) -I=$(GOPATH)/src --go_out=$(GOPATH)/src $(WORKING_DIRECTORY)/pb/getDevicesMetadata.proto
	protoc -I=. -I=$(REPOSITORY_DIRECTORY) -I=$(GOPATH)/src --go_out=$(GOPATH)/src $(WORKING_DIRECTORY)/pb/getPendingCommands.proto
	protoc -I=. -I=$(REPOSITORY_DIRECTORY) -I=$(GOPATH)/src --go_out=$(GOPATH)/src $(WORKING_DIRECTORY)/pb/cancelCommands.proto
	protoc -I=. -I=$(REPOSITORY_DIRECTORY) -I=$(GOPATH)/src --go_out=$(GOPATH)/src $(WORKING_DIRECTORY)/pb/updateDeviceMetadata.proto
	protoc -I=. -I=$(REPOSITORY_DIRECTORY) -I=$(GOPATH)/src -I=$(GOOGLEAPIS_PATH) -I=$(GRPCGATEWAY_MODULE_PATH) --go-grpc_out=$(GOPATH)/src $(WORKING_DIRECTORY)/pb/service.proto
	protoc -I=. -I=$(REPOSITORY_DIRECTORY) -I=$(GOPATH)/src -I=$(GOOGLEAPIS_PATH) -I=$(GRPCGATEWAY_MODULE_PATH) --openapiv2_out=$(REPOSITORY_DIRECTORY) \
		--openapiv2_opt logtostderr=true \
		$(WORKING_DIRECTORY)/pb/service.proto
	protoc -I=. -I=$(REPOSITORY_DIRECTORY) -I=$(GOPATH)/src -I=$(GOOGLEAPIS_PATH) -I=$(GRPCGATEWAY_MODULE_PATH) --grpc-gateway_out $(GOPATH)/src \
		--grpc-gateway_opt logtostderr=true \
		--grpc-gateway_opt paths=source_relative \
		$(WORKING_DIRECTORY)/pb/service.proto

.PHONY: build-servicecontainer build push proto/generate


