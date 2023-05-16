SHELL = /bin/bash
SERVICE_NAME = $(notdir $(CURDIR))
LATEST_TAG ?= vnext
BRANCH_TAG ?= $(shell git rev-parse --abbrev-ref HEAD | sed 's/[^a-zA-Z0-9]/-/g')
ifneq ($(BRANCH_TAG),main)
	LATEST_TAG = $(BRANCH_TAG)
endif
VERSION_TAG ?= $(LATEST_TAG)-$(shell git rev-parse --short=7 --verify HEAD)
HTTP_BUILD_DIR ?= $(CURDIR)/web/build

default: build

define build-docker-image
	cd .. && DOCKER_BUILDKIT=1 docker build \
		--network=host \
		--tag ghcr.io/plgd-dev/hub/$(SERVICE_NAME):$(VERSION_TAG) \
		--tag ghcr.io/plgd-dev/hub/$(SERVICE_NAME):$(LATEST_TAG) \
		--tag ghcr.io/plgd-dev/hub/$(SERVICE_NAME):$(BRANCH_TAG) \
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


