SHELL = /bin/bash
SERVICE_NAME = $(notdir $(CURDIR))
LATEST_TAG ?= vnext
VERSION_TAG ?= $(LATEST_TAG)-$(shell git rev-parse --short=7 --verify HEAD)

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

build-test: build-servicecontainer
	docker run  -d --network=bridge \
		-p 5684:5684 -p 5683:5683 -p 9085:9085 -p 9086:9086 -p 9088:9088 -p 9089:9089 -p 9090:9090 \
		--name=$(SERVICE_NAME) \
		--hostname=$(SERVICE_NAME) \
		-e FQDN=localhost \
		-v $(shell pwd)/.tmp/data/log:/data/log \
		-v $(shell pwd)/.tmp/data/certs:/data/certs \
		-v $(shell pwd)/.tmp/data/oauth/keys:/data/oauth/keys \
		plgd/$(SERVICE_NAME):$(LATEST_TAG)

build-clean:
ifneq ($(shell docker ps | grep $(SERVICE_NAME) | tr -s ' ' | cut -d ' ' -f 1),)
	docker stop $(shell docker ps | grep $(SERVICE_NAME) | tr -s ' ' | cut -d ' ' -f 1)
endif
ifneq ($(shell docker ps -a | grep $(SERVICE_NAME) | tr -s ' ' | cut -d ' ' -f 1),)
	docker rm $(shell docker ps -a | grep $(SERVICE_NAME) | tr -s ' ' | cut -d ' ' -f 1)
endif

push: build-servicecontainer
	docker push plgd/$(SERVICE_NAME):$(VERSION_TAG)
	docker push plgd/$(SERVICE_NAME):$(LATEST_TAG)

proto/generate:

.PHONY: build-servicecontainer build push proto/generate


