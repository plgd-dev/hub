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

proto/generate:
	protoc -I=. -I=$(REPOSITORY_DIRECTORY) -I=$(GOPATH)/src --go_out=$(GOPATH)/src $(WORKING_DIRECTORY)/pb/commands.proto
	protoc -I=. -I=$(REPOSITORY_DIRECTORY) -I=$(GOPATH)/src --go_out=$(GOPATH)/src $(WORKING_DIRECTORY)/pb/events.proto
	protoc -I=. -I=$(REPOSITORY_DIRECTORY) -I=$(GOPATH)/src --go_out=$(GOPATH)/src $(WORKING_DIRECTORY)/pb/resources.proto
	protoc -I=. -I=$(REPOSITORY_DIRECTORY) -I=$(GOPATH)/src --go_out=$(GOPATH)/src $(WORKING_DIRECTORY)/cqrs/aggregate/test/aggregate_test.proto
	protoc -I=. -I=$(REPOSITORY_DIRECTORY) -I=$(GOPATH)/src --go_out=$(GOPATH)/src $(WORKING_DIRECTORY)/cqrs/eventbus/pb/eventbus.proto
	protoc -I=. -I=$(REPOSITORY_DIRECTORY) -I=$(GOPATH)/src --go-grpc_out=$(GOPATH)/src $(WORKING_DIRECTORY)/pb/service.proto

.PHONY: build-servicecontainer build push clean proto/generate

