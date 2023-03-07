SHELL = /bin/bash
SERVICE_NAME = mock-oauth-server
LATEST_TAG ?= vnext
VERSION_TAG ?= $(LATEST_TAG)-$(shell git rev-parse --short=7 --verify HEAD)

default: build

define build-docker-image
	cd ../.. && \
		mkdir -p .tmp/docker/$(SERVICE_NAME) && \
		awk '{gsub("@NAME@","$(SERVICE_NAME)")} {gsub("@DIRECTORY@","test/oauth-server")} {print}' tools/docker/Dockerfile.in > .tmp/docker/$(SERVICE_NAME)/Dockerfile && \
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
.PHONY: proto/generate

.PHONY: build-servicecontainer build push proto/generate
