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
	protoc -I=. -I=$(REPOSITORY_DIRECTORY) -I=$(GOPATH)/src -I=./pb --go_out=$(GOPATH)/src $(WORKING_DIRECTORY)/pb/devices.proto
	protoc -I=. -I=$(REPOSITORY_DIRECTORY) -I=$(GOPATH)/src -I=./pb --go_out=$(GOPATH)/src $(WORKING_DIRECTORY)/pb/events.proto
	protoc -I=. -I=$(REPOSITORY_DIRECTORY) -I=$(GOPATH)/src -I=./pb --go_out=$(GOPATH)/src $(WORKING_DIRECTORY)/pb/service.proto
	protoc -I=. -I=$(REPOSITORY_DIRECTORY) -I=$(GOPATH)/src -I=./pb --go-grpc_out=$(GOPATH)/src $(WORKING_DIRECTORY)/pb/service.proto

.PHONY: build-servicecontainer build push clean proto/generate
