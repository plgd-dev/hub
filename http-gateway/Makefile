SHELL = /bin/bash
SERVICE_NAME = $(notdir $(CURDIR))
LATEST_TAG ?= vnext
BRANCH_TAG ?= $(shell git rev-parse --abbrev-ref HEAD | sed 's/[^a-zA-Z0-9]/-/g')
ifneq ($(BRANCH_TAG),main)
	LATEST_TAG = $(BRANCH_TAG)
endif
VERSION_TAG ?= $(LATEST_TAG)-$(shell git rev-parse --short=7 --verify HEAD)
HTTP_BUILD_DIR ?= $(CURDIR)/web/build
BUILD_COMMIT_DATE ?= $(shell date -u +%FT%TZ --date=@`git show --format='%ct' HEAD --quiet`)
BUILD_SHORT_COMMIT ?= $(shell git show --format=%h HEAD --quiet)
BUILD_DATE ?= $(shell date -u +%FT%TZ)
BUILD_VERSION ?= $(shell git tag --sort version:refname | tail -1 | sed -e "s/^v//")

default: build

define build-docker-image
	cd .. && DOCKER_BUILDKIT=1 docker build \
		--network=host \
		--tag ghcr.io/plgd-dev/hub/$(SERVICE_NAME):$(VERSION_TAG) \
		--tag ghcr.io/plgd-dev/hub/$(SERVICE_NAME):$(LATEST_TAG) \
		--tag ghcr.io/plgd-dev/hub/$(SERVICE_NAME):$(BRANCH_TAG) \
		--build-arg COMMIT_DATE="$(BUILD_COMMIT_DATE)" \
		--build-arg SHORT_COMMIT="$(BUILD_SHORT_COMMIT)" \
		--build-arg DATE="$(BUILD_DATE)" \
		--build-arg VERSION="$(BUILD_VERSION)" \
		--target $(1) \
		-f $(SERVICE_NAME)/Dockerfile \
		.
endef

build-www:
	cd .. && mkdir -p $(HTTP_BUILD_DIR) && DOCKER_BUILDKIT=1 docker build \
		--network=host \
		--tag plgd/$(SERVICE_NAME)-ui:$(VERSION_TAG) \
		--tag plgd/$(SERVICE_NAME)-ui:$(LATEST_TAG) \
		--tag plgd/$(SERVICE_NAME)-ui:$(BRANCH_TAG) \
		-f $(SERVICE_NAME)/Dockerfile.www \
		.
	docker run --rm -v $(HTTP_BUILD_DIR):/www plgd/$(SERVICE_NAME)-ui:$(VERSION_TAG) /bin/bash -c "cp -r /web/build/* /www"

build-servicecontainer: build-www
	$(call build-docker-image,service)

build: build-servicecontainer

push: build-servicecontainer
	docker push plgd/$(SERVICE_NAME):$(VERSION_TAG)
	docker push plgd/$(SERVICE_NAME):$(LATEST_TAG)

proto/generate:

.PHONY: build-servicecontainer build push proto/generate build-www


