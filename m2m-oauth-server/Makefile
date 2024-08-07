SHELL = /bin/bash
SERVICE_NAME = m2m-oauth-server
LATEST_TAG ?= vnext
BRANCH_TAG ?= $(shell git rev-parse --abbrev-ref HEAD | sed 's/[^a-zA-Z0-9]/-/g')
ifneq ($(BRANCH_TAG),main)
	LATEST_TAG = $(BRANCH_TAG)
endif
VERSION_TAG ?= $(LATEST_TAG)-$(shell git rev-parse --short=7 --verify HEAD)
BUILD_COMMIT_DATE ?= $(shell date -u +%FT%TZ --date=@`git show --format='%ct' HEAD --quiet`)
BUILD_SHORT_COMMIT ?= $(shell git show --format=%h HEAD --quiet)
BUILD_DATE ?= $(shell date -u +%FT%TZ)
BUILD_VERSION ?= $(shell git tag --sort version:refname | tail -1 | sed -e "s/^v//")

default: build

define build-docker-image
	cd ../.. && \
		mkdir -p .tmp/docker/$(SERVICE_NAME) && \
		awk '{gsub("@NAME@","$(SERVICE_NAME)")} {gsub("@DIRECTORY@","m2m-oauth-server")} {print}' tools/docker/Dockerfile.in > .tmp/docker/$(SERVICE_NAME)/Dockerfile && \
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

proto/generate:

.PHONY: build-servicecontainer build push proto/generate
