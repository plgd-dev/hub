SHELL = /bin/bash
SERVICE_NAME = $(notdir $(CURDIR))
LATEST_TAG = vnext
VERSION_TAG = vnext-$(shell git rev-parse --short=7 --verify HEAD)

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

push: build-servicecontainer
	docker push plgd/$(SERVICE_NAME):$(VERSION_TAG)
	docker push plgd/$(SERVICE_NAME):$(LATEST_TAG)

proto/generate:
	protoc -I=. -I=${GOPATH}/src --gogofaster_out=${GOPATH}/src $(shell pwd)/pb/clientConfiguration.proto
	protoc -I=. -I=${GOPATH}/src --gogofaster_out=${GOPATH}/src $(shell pwd)/pb/devices.proto
	protoc -I=. -I=${GOPATH}/src --go_out=plugins=grpc:${GOPATH}/src --grpchan_out=${GOPATH}/src $(shell pwd)/pb/service.proto
	protoc -I=. -I=${GOPATH}/src --go_out=${GOPATH}/src $(shell pwd)/pb/errdetails/errorDetails.proto

.PHONY: build-servicecontainer build push proto/generate


