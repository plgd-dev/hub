SHELL = /bin/bash
SIMULATOR_NAME_SUFFIX ?= $(shell hostname)
CLOUD_SID ?= adebc667-1f2b-41e3-bf5c-6d6eabc68cc6
LATEST_TAG ?= vnext
ifeq ($(strip $(LATEST_TAG)),)
BUILD_TAG := vnext
else
BUILD_TAG := $(LATEST_TAG)
endif

#$(error MY_FLAG=$(BUILD_TAG)AAA)

SUBDIRS := resource-aggregate authorization resource-directory cloud2cloud-connector cloud2cloud-gateway coap-gateway grpc-gateway certificate-authority portal-webapi bundle http-gateway
.PHONY: $(SUBDIRS) push proto/generate clean build test env mongo nats certificates cloud-build

default: build

cloud-build:
	docker build \
		--network=host \
		--tag cloud-build \
		.

cloud-test:
	docker build \
		--network=host \
		--tag cloud-test \
		-f Dockerfile.test \
		.

certificates: cloud-test
	mkdir -p $(shell pwd)/.tmp/certs
	docker run \
		--network=host \
		-v $(shell pwd)/.tmp/certs:/certs \
		--user $(shell id -u):$(shell id -g) \
		cloud-test \
		/bin/bash -c "cert-tool --cmd.generateRootCA --outCert=/certs/root_ca.crt --outKey=/certs/root_ca.key --cert.subject.cn=RootCA && cert-tool --cmd.generateCertificate --outCert=/certs/http.crt --outKey=/certs/http.key --cert.subject.cn=localhost --cert.san.domain=localhost --signerCert=/certs/root_ca.crt --signerKey=/certs/root_ca.key && cert-tool --cmd.generateIdentityCertificate=$(CLOUD_SID) --outCert=/certs/coap.crt --outKey=/certs/coap.key --cert.san.domain=localhost --signerCert=/certs/root_ca.crt --signerKey=/certs/root_ca.key"
	cat $(shell pwd)/.tmp/certs/http.crt > $(shell pwd)/.tmp/certs/mongo.key
	cat $(shell pwd)/.tmp/certs/http.key >> $(shell pwd)/.tmp/certs/mongo.key

privateKeys:
	mkdir -p $(shell pwd)/.tmp/privKeys
	openssl genrsa -out $(shell pwd)/.tmp/privKeys/idTokenKey.pem 4096
	openssl ecparam -name prime256v1 -genkey -noout -out $(shell pwd)/.tmp/privKeys/accessTokenKey.pem

nats: certificates
	docker run \
	    -d \
		--network=host \
		--name=nats \
		-v $(shell pwd)/.tmp/certs:/certs \
		--user $(shell id -u):$(shell id -g) \
		nats --tls --tlsverify --tlscert=/certs/http.crt --tlskey=/certs/http.key --tlscacert=/certs/root_ca.crt
	docker run \
	    -d \
		--network=host \
		--name=nats-cloud-connector \
		-v $(shell pwd)/.tmp/certs:/certs \
		--user $(shell id -u):$(shell id -g) \
		nats --port 34222 --tls --tlsverify --tlscert=/certs/http.crt --tlskey=/certs/http.key --tlscacert=/certs/root_ca.crt

mongo: certificates
	mkdir -p $(shell pwd)/.tmp/mongo
	docker run \
	    -d \
		--network=host \
		--name=mongo \
		-v $(shell pwd)/.tmp/mongo:/data/db \
		-v $(shell pwd)/.tmp/certs:/certs --user $(shell id -u):$(shell id -g) \
		mongo --tlsMode requireTLS --tlsCAFile /certs/root_ca.crt --tlsCertificateKeyFile /certs/mongo.key

env: clean certificates nats mongo privateKeys
	if [ "${TRAVIS_OS_NAME}" == "linux" ]; then \
		sudo sh -c 'echo 0 > /proc/sys/net/ipv6/conf/all/disable_ipv6'; \
	fi
	docker build ./device-simulator --network=host -t device-simulator --target service
	docker run -d --name=devsim --network=host -t device-simulator devsim-$(SIMULATOR_NAME_SUFFIX)

test: env
	mkdir -p $(shell pwd)/.tmp/home
	mkdir -p $(shell pwd)/.tmp/home/certificate-authority
	docker run \
		--network=host \
		-v $(shell pwd)/.tmp/certs:/certs \
		-v $(shell pwd)/.tmp/home:/home \
		-v $(shell pwd)/.tmp/privKeys:/privKeys \
		--user $(shell id -u):$(shell id -g) \
		-e HOME=/home \
		-e DIAL_TYPE="file" \
		-e DIAL_FILE_CA_POOL=/certs/root_ca.crt \
		-e DIAL_FILE_CERT_DIR_PATH=/certs \
		-e DIAL_FILE_CERT_NAME=http.crt \
		-e DIAL_FILE_CERT_KEY_NAME=http.key \
		-e LISTEN_TYPE="file" \
		-e LISTEN_FILE_CA_POOL=/certs/root_ca.crt \
		-e LISTEN_FILE_CERT_DIR_PATH=/certs \
		-e LISTEN_FILE_CERT_NAME=http.crt \
		-e LISTEN_FILE_CERT_KEY_NAME=http.key \
		-e TEST_COAP_GW_OVERWRITE_LISTEN_FILE_CERT_NAME=coap.crt \
		-e TEST_COAP_GW_OVERWRITE_LISTEN_FILE_KEY_NAME=coap.key \
		-e TEST_CLOUD_SID=$(CLOUD_SID) \
		-e TEST_ROOT_CA_CRT=/certs/root_ca.crt \
        -e TEST_ROOT_CA_KEY=/certs/root_ca.key \
		-e ACME_DB_DIR=/home/certificate-authority \
		-e TEST_OAUTH_SERVER_ID_TOKEN_PRIVATE_KEY=/privKeys/idTokenKey.pem \
		-e TEST_OAUTH_SERVER_ACCESS_TOKEN_PRIVATE_KEY=/privKeys/accessTokenKey.pem \
		cloud-test \
		go test -race -p 1 -v ./... -covermode=atomic -coverprofile=/home/coverage.txt

build: cloud-build $(SUBDIRS)

clean:
	docker rm -f mongo || true
	docker rm -f nats || true
	docker rm -f nats-cloud-connector || true
	docker rm -f devsim || true
	sudo rm -rf ./.tmp/certs || true
	sudo rm -rf ./.tmp/mongo || true
	sudo rm -rf ./.tmp/home || true
	sudo rm -rf ./.tmp/privateKeys || true

proto/generate: $(SUBDIRS)
push: cloud-build $(SUBDIRS) 

$(SUBDIRS):
	$(MAKE) -C $@ $(MAKECMDGOALS) LATEST_TAG=$(BUILD_TAG)
