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

push: build-servicecontainer
	docker push plgd/$(SERVICE_NAME):$(VERSION_TAG)
	docker push plgd/$(SERVICE_NAME):$(LATEST_TAG)

proto/generate:
	protoc -I=. -I=${GOPATH}/src --go_out=${GOPATH}/src $(shell pwd)/pb/clientConfiguration.proto
	protoc -I=. -I=${GOPATH}/src --go_out=${GOPATH}/src $(shell pwd)/pb/devices.proto
	protoc -I=. -I=${GOPATH}/src --go_out=${GOPATH}/src $(shell pwd)/pb/getPendingCommands.proto
	protoc -I=. -I=${GOPATH}/src --go_out=${GOPATH}/src $(shell pwd)/pb/getDevicesMetadata.proto
	protoc -I=. -I=${GOPATH}/src --go_out=${GOPATH}/src $(shell pwd)/pb/updateDeviceMetadata.proto
	protoc -I=. -I=${GOPATH}/src -I=${GOPATH}/src/github.com/googleapis/googleapis -I=${GOPATH}/src/github.com/grpc-ecosystem/grpc-gateway --go-grpc_out=${GOPATH}/src $(shell pwd)/pb/service.proto
	protoc -I=. -I=${GOPATH}/src --go_out=${GOPATH}/src $(shell pwd)/pb/errdetails/errorDetails.proto
	protoc -I=. -I=${GOPATH}/src -I=${GOPATH}/src/github.com/googleapis/googleapis -I=${GOPATH}/src/github.com/grpc-ecosystem/grpc-gateway --openapiv2_out=${GOPATH}/src \
    	--openapiv2_opt logtostderr=true \
    	$(shell pwd)/pb/service.proto
	protoc -I=. -I=${GOPATH}/src -I=${GOPATH}/src/github.com/googleapis/googleapis -I=${GOPATH}/src/github.com/grpc-ecosystem/grpc-gateway --grpc-gateway_out ${GOPATH}/src \
    	--grpc-gateway_opt logtostderr=true \
    	--grpc-gateway_opt paths=source_relative \
   		$(shell pwd)/pb/service.proto

.PHONY: build-servicecontainer build push proto/generate


