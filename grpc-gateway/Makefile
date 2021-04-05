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
	docker run -d --network=plgdnet \
		-p 9084:9084 \
		--name=$(SERVICE_NAME) \
		--hostname=$(SERVICE_NAME) \
		-v $(shell pwd)/../.tmp/certs:/data/certs \
		-v $(shell pwd)/grpc-gateway.yaml:/data/grpc-gateway.yaml \
		plgd/$(SERVICE_NAME):$(LATEST_TAG) --config=/data/grpc-gateway.yaml

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
	protoc -I=. -I=${GOPATH}/src --go_out=${GOPATH}/src $(shell pwd)/pb/clientConfiguration.proto
	protoc -I=. -I=${GOPATH}/src --go_out=${GOPATH}/src $(shell pwd)/pb/devices.proto
	protoc -I=. -I=${GOPATH}/src --go-grpc_out=${GOPATH}/src $(shell pwd)/pb/service.proto
	protoc -I=. -I=${GOPATH}/src --go_out=${GOPATH}/src $(shell pwd)/pb/errdetails/errorDetails.proto

.PHONY: build-servicecontainer build push proto/generate


