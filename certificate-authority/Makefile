SHELL = /bin/bash
SERVICE_NAME = $(notdir $(CURDIR))
LATEST_TAG ?= vnext
VERSION_TAG ?= $(LATEST_TAG)-$(shell git rev-parse --short=7 --verify HEAD)
GOPATH ?= $(shell go env GOPATH)
WORKING_DIRECTORY := $(shell pwd)
REPOSITORY_DIRECTORY := $(shell cd .. && pwd)

default: build

define build-docker-image
	cd .. && \
		mkdir -p .tmp/docker/$(SERVICE_NAME) && \
		awk '{gsub("@NAME@","$(SERVICE_NAME)")} {gsub("@DIRECTORY@","$(SERVICE_NAME)")} {print}' tools/docker/Dockerfile.in > .tmp/docker/$(SERVICE_NAME)/Dockerfile && \
		docker build \
		--network=host \
		--tag plgd/$(SERVICE_NAME):$(VERSION_TAG) \
		--tag plgd/$(SERVICE_NAME):$(LATEST_TAG) \
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
	protoc -I=. -I=$(REPOSITORY_DIRECTORY) -I=$(GOPATH)/src -I=$(GOOGLEAPIS_PATH) -I=$(GRPCGATEWAY_MODULE_PATH) --openapiv2_out=$(REPOSITORY_DIRECTORY) \
		--openapiv2_opt logtostderr=true \
		$(WORKING_DIRECTORY)/pb/service.proto
	protoc -I=. -I=$(REPOSITORY_DIRECTORY) -I=$(GOPATH)/src -I=$(GOOGLEAPIS_PATH) -I=$(GRPCGATEWAY_MODULE_PATH) --grpc-gateway_out=$(REPOSITORY_DIRECTORY) \
		--grpc-gateway_opt logtostderr=true \
		--grpc-gateway_opt paths=source_relative \
		$(WORKING_DIRECTORY)/pb/service.proto
	protoc -I=. -I=$(REPOSITORY_DIRECTORY) -I=$(GOPATH)/src -I=$(GOOGLEAPIS_PATH) -I=$(GRPCGATEWAY_MODULE_PATH) --go-grpc_out=$(GOPATH)/src \
		$(WORKING_DIRECTORY)/pb/service.proto

.PHONY: build-servicecontainer build push proto/generate






