SHELL = /bin/bash
SERVICE_NAME = $(notdir $(CURDIR))
LATEST_TAG ?= vnext
VERSION_TAG ?= $(LATEST_TAG)-$(shell git rev-parse --short=7 --verify HEAD)
GOPATH ?= $(shell go env GOPATH)
WORKING_DIRECTORY := $(shell pwd)
BUILD_COMMIT_DATE ?= $(shell date -u +%FT%TZ --date=@`git show --format='%ct' HEAD --quiet`)
BUILD_SHORT_COMMIT ?= $(shell git show --format=%h HEAD --quiet)
BUILD_DATE ?= $(shell date -u +%FT%TZ)
BUILD_VERSION ?= $(shell git tag --sort version:refname | tail -1 | sed -e "s/^v//")

default: build

define build-docker-image
	cd ../.. && docker build \
		--network=host \
		--tag plgd/$(SERVICE_NAME):$(VERSION_TAG) \
		--tag plgd/$(SERVICE_NAME):$(LATEST_TAG) \
		--build-arg COMMIT_DATE="$(BUILD_COMMIT_DATE)" \
		--build-arg SHORT_COMMIT="$(BUILD_SHORT_COMMIT)" \
		--build-arg DATE="$(BUILD_DATE)" \
		--build-arg VERSION="$(BUILD_VERSION)" \
		--target $(1) \
		-f tools/grpc-reflection/Dockerfile \
		.
endef

build-servicecontainer:
	$(call build-docker-image,service)

build: build-servicecontainer

push: build-servicecontainer
	docker push plgd/$(SERVICE_NAME):$(VERSION_TAG)
	docker push plgd/$(SERVICE_NAME):$(LATEST_TAG)

proto/generate:


.PHONY: build-servicecontainer build push proto/generate






