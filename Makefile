SHELL = /bin/bash
SIMULATOR_NAME_SUFFIX ?= $(shell hostname)
CLOUD_SID ?= adebc667-1f2b-41e3-bf5c-6d6eabc68cc6
LATEST_TAG ?= vnext
ifeq ($(strip $(LATEST_TAG)),)
BUILD_TAG := vnext
else
BUILD_TAG := $(LATEST_TAG)
endif
GOPATH ?= $(shell go env GOPATH)
WORKING_DIRECTORY := $(shell pwd)
USER_ID := $(shell id -u)
GROUP_ID := $(shell id -g)

#$(error MY_FLAG=$(BUILD_TAG)AAA)

SUBDIRS := resource-aggregate authorization resource-directory cloud2cloud-connector cloud2cloud-gateway coap-gateway grpc-gateway certificate-authority bundle http-gateway
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
	mkdir -p $(WORKING_DIRECTORY)/.tmp/certs
	docker run \
		--network=host \
		-v $(WORKING_DIRECTORY)/.tmp/certs:/certs \
		--user $(USER_ID):$(GROUP_ID) \
		cloud-test \
		/bin/bash -c "cert-tool --cmd.generateRootCA --outCert=/certs/root_ca.crt --outKey=/certs/root_ca.key --cert.subject.cn=RootCA && cert-tool --cmd.generateCertificate --outCert=/certs/http.crt --outKey=/certs/http.key --cert.subject.cn=localhost --cert.san.domain=localhost --signerCert=/certs/root_ca.crt --signerKey=/certs/root_ca.key && cert-tool --cmd.generateIdentityCertificate=$(CLOUD_SID) --outCert=/certs/coap.crt --outKey=/certs/coap.key --cert.san.domain=localhost --signerCert=/certs/root_ca.crt --signerKey=/certs/root_ca.key"
	cat $(WORKING_DIRECTORY)/.tmp/certs/http.crt > $(WORKING_DIRECTORY)/.tmp/certs/mongo.key
	cat $(WORKING_DIRECTORY)/.tmp/certs/http.key >> $(WORKING_DIRECTORY)/.tmp/certs/mongo.key

privateKeys:
	mkdir -p $(WORKING_DIRECTORY)/.tmp/privKeys
	openssl genrsa -out $(WORKING_DIRECTORY)/.tmp/privKeys/idTokenKey.pem 4096
	openssl ecparam -name prime256v1 -genkey -noout -out $(WORKING_DIRECTORY)/.tmp/privKeys/accessTokenKey.pem

nats: certificates
	mkdir -p $(WORKING_DIRECTORY)/.tmp/jetstream/cloud
	mkdir -p $(WORKING_DIRECTORY)/.tmp/jetstream/cloud-connector
	docker run \
	    -d \
		--network=host \
		--name=nats \
		-v $(WORKING_DIRECTORY)/.tmp/certs:/certs \
		-v $(WORKING_DIRECTORY)/.tmp/jetstream/cloud:/data \
		--user $(USER_ID):$(GROUP_ID) \
		nats --jetstream --store_dir /data --tls --tlsverify --tlscert=/certs/http.crt --tlskey=/certs/http.key --tlscacert=/certs/root_ca.crt
	docker run \
	    -d \
		--network=host \
		--name=nats-cloud-connector \
		-v $(WORKING_DIRECTORY)/.tmp/certs:/certs \
		-v $(WORKING_DIRECTORY)/.tmp/jetstream/cloud-connector:/data \
		--user $(USER_ID):$(GROUP_ID) \
		nats --jetstream --store_dir /data --port 34222 --tls --tlsverify --tlscert=/certs/http.crt --tlskey=/certs/http.key --tlscacert=/certs/root_ca.crt

mongo: certificates
	mkdir -p $(WORKING_DIRECTORY)/.tmp/mongo
	docker run \
	    -d \
		--network=host \
		--name=mongo \
		-v $(WORKING_DIRECTORY)/.tmp/mongo:/data/db \
		-v $(WORKING_DIRECTORY)/.tmp/certs:/certs --user $(USER_ID):$(GROUP_ID) \
		mongo --tlsMode requireTLS --tlsCAFile /certs/root_ca.crt --tlsCertificateKeyFile /certs/mongo.key

env: clean certificates nats mongo privateKeys
	if [ "${TRAVIS_OS_NAME}" == "linux" ]; then \
		sudo sh -c 'echo 0 > /proc/sys/net/ipv6/conf/all/disable_ipv6'; \
	fi
	docker build ./device-simulator --network=host -t device-simulator --target service
	docker run -d --name=devsim --network=host -t device-simulator devsim-$(SIMULATOR_NAME_SUFFIX)

define RUN-TESTS-IN-DIRECTORY
	echo "Executing tests from $(1) directory"; \
	START_TIME=$$(date +%s); \
	docker run \
	--rm \
	--network=host \
	-v $(WORKING_DIRECTORY)/.tmp/certs:/certs \
	-v $(WORKING_DIRECTORY)/.tmp/coverage:/coverage \
	-v $(WORKING_DIRECTORY)/.tmp/privKeys:/privKeys \
	-v /var/run/docker.sock:/var/run/docker.sock \
	-e LISTEN_FILE_CA_POOL=/certs/root_ca.crt \
	-e LISTEN_FILE_CERT_DIR_PATH=/certs \
	-e LISTEN_FILE_CERT_NAME=http.crt \
	-e LISTEN_FILE_CERT_KEY_NAME=http.key \
	-e TEST_COAP_GW_CERT_FILE=/certs/coap.crt \
	-e TEST_COAP_GW_KEY_FILE=/certs/coap.key \
	-e TEST_CLOUD_SID=$(CLOUD_SID) \
	-e TEST_ROOT_CA_CERT=/certs/root_ca.crt \
	-e TEST_ROOT_CA_KEY=/certs/root_ca.key \
	-e TEST_OAUTH_SERVER_ID_TOKEN_PRIVATE_KEY=/privKeys/idTokenKey.pem \
	-e TEST_OAUTH_SERVER_ACCESS_TOKEN_PRIVATE_KEY=/privKeys/accessTokenKey.pem \
	cloud-test \
	go test -timeout=45m -race -p 1 -v $(1)... -covermode=atomic -coverprofile=/coverage/`echo $(1) | sed -e "s/[\.\/]//g"`.coverage.txt ; \
	EXIT_STATUS=$$? ; \
	if [ $${EXIT_STATUS} -ne 0 ]; then \
		exit $${EXIT_STATUS}; \
	fi ; \
	STOP_TIME=$$(date +%s) ; \
	EXECUTION_TIME=$$((STOP_TIME-START_TIME)) ; \
	echo "" ; \
	echo "Execution time: $${EXECUTION_TIME} seconds" ; \
	echo "" ;
endef

DIRECTORIES:=$(shell ls -d ./*/)

test: env
	@mkdir -p $(WORKING_DIRECTORY)/.tmp/home
	@mkdir -p $(WORKING_DIRECTORY)/.tmp/home/certificate-authority
	@for DIRECTORY in $(DIRECTORIES); do \
		if ! go list -f '{{.GoFiles}}' $$DIRECTORY... 2>/dev/null | grep go > /dev/null 2>&1; then \
			echo "No golang files detected, directory $${DIRECTORY} skipped"; \
			continue ; \
		fi ; \
		$(call RUN-TESTS-IN-DIRECTORY,$$DIRECTORY) \
	done

test-targets := $(addprefix test-,$(patsubst ./%/,%,$(DIRECTORIES)))

$(test-targets): %: env
	@mkdir -p $(WORKING_DIRECTORY)/.tmp/home
	@mkdir -p $(WORKING_DIRECTORY)/.tmp/home/certificate-authority
	@readonly TARGET_DIRECTORY=$(patsubst test-%,./%/,$@) ; \
	if ! go list -f '{{.GoFiles}}' $$TARGET_DIRECTORY... 2>/dev/null | grep go > /dev/null 2>&1; then \
		echo "No golang files detected, directory $$TARGET_DIRECTORY skipped"; \
		exit 0; \
	fi ; \
	$(call RUN-TESTS-IN-DIRECTORY,$(patsubst test-%,./%/,$@))

.PHONY: $(test-targets)

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
	protoc -I=. -I=$(GOPATH)/src --go_out=$(GOPATH)/src $(WORKING_DIRECTORY)/pkg/net/grpc/server/stub.proto
	protoc -I=. -I=$(GOPATH)/src --go-grpc_out=$(GOPATH)/src $(WORKING_DIRECTORY)/pkg/net/grpc/server/stub.proto
push: cloud-build $(SUBDIRS)

$(SUBDIRS):
	$(MAKE) -C $@ $(MAKECMDGOALS) LATEST_TAG=$(BUILD_TAG)
