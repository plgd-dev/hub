SHELL = /bin/bash
SIMULATOR_NAME_SUFFIX ?= $(shell hostname)
CLOUD_SID ?= adebc667-1f2b-41e3-bf5c-6d6eabc68cc6
LATEST_TAG ?= vnext
ifeq ($(strip $(LATEST_TAG)),)
BUILD_TAG := vnext
else
BUILD_TAG := $(LATEST_TAG)
endif
BRANCH_TAG ?= $(shell git rev-parse --abbrev-ref HEAD | sed 's/[^a-zA-Z0-9]/-/g')
ifneq ($(BRANCH_TAG),main)
	BUILD_TAG = $(BRANCH_TAG)
endif
GOPATH ?= $(shell go env GOPATH)
WORKING_DIRECTORY := $(shell pwd)
USER_ID := $(shell id -u)
GROUP_ID := $(shell id -g)
TEST_CHECK_RACE ?= false
ifeq ($(TEST_CHECK_RACE),true)
GO_BUILD_ARG := -race
else
GO_BUILD_ARG := $(GO_BUILD_ARG)
endif
TEST_TIMEOUT ?= 45m
TEST_COAP_GATEWAY_UDP_ENABLED ?= true
TEST_COAP_GATEWAY_LOG_LEVEL ?= info
TEST_COAP_GATEWAY_LOG_DUMP_BODY ?= false
TEST_RESOURCE_AGGREGATE_LOG_LEVEL ?= info
TEST_RESOURCE_AGGREGATE_LOG_DUMP_BODY ?= false
TEST_GRPC_GATEWAY_LOG_LEVEL ?= info
TEST_GRPC_GATEWAY_LOG_DUMP_BODY ?= false
TEST_IDENTITY_STORE_LOG_LEVEL ?= info
TEST_IDENTITY_STORE_LOG_DUMP_BODY ?= false
TEST_SNIPPET_SERVICE_LOG_LEVEL ?= info
TEST_SNIPPET_SERVICE_LOG_DUMP_BODY ?= false
TEST_MEMORY_COAP_GATEWAY_NUM_DEVICES ?= 1
TEST_MEMORY_COAP_GATEWAY_NUM_RESOURCES ?= 1
TEST_MEMORY_COAP_GATEWAY_EXPECTED_RSS_IN_MB ?= 50
TEST_MEMORY_COAP_GATEWAY_RESOURCE_DATA_SIZE ?= 200
TEST_DATABASE ?= mongodb
TEST_LEAD_RESOURCE_TYPE_FILTER ?=
TEST_LEAD_RESOURCE_TYPE_REGEX_FILTER ?=
TEST_LEAD_RESOURCE_TYPE_USE_UUID ?= false
# supported values: ECDSA-SHA256, ECDSA-SHA384, ECDSA-SHA512
CERT_TOOL_SIGN_ALG ?= ECDSA-SHA256
# supported values: P256, P384, P521
CERT_TOOL_ELLIPTIC_CURVE ?= P256
CERT_TOOL_IMAGE = ghcr.io/plgd-dev/hub/cert-tool:vnext

SUBDIRS := bundle certificate-authority cloud2cloud-connector cloud2cloud-gateway coap-gateway grpc-gateway resource-aggregate resource-directory http-gateway identity-store snippet-service m2m-oauth-server test/oauth-server tools/cert-tool
.PHONY: $(SUBDIRS) push proto/generate clean build test env mongo nats certificates hub-build http-gateway-www simulators

default: build

hub-test:
	docker build \
		--network=host \
		--tag hub-test \
		-f Dockerfile.test \
		.

certificates:
	mkdir -p $(WORKING_DIRECTORY)/.tmp/certs
	docker run \
		--rm -v $(WORKING_DIRECTORY)/.tmp/certs:/certs \
		--user $(USER_ID):$(GROUP_ID) \
		${CERT_TOOL_IMAGE} \
		--cmd.generateRootCA --outCert=/certs/root_ca.crt --outKey=/certs/root_ca.key --cert.subject.cn=RootCA \
				--cert.signatureAlgorithm=$(CERT_TOOL_SIGN_ALG) --cert.ellipticCurve=$(CERT_TOOL_ELLIPTIC_CURVE)
	docker run \
		--rm -v $(WORKING_DIRECTORY)/.tmp/certs:/certs \
		--user $(USER_ID):$(GROUP_ID) \
		${CERT_TOOL_IMAGE} \
		--cmd.generateCertificate --outCert=/certs/http.crt --outKey=/certs/http.key	--cert.subject.cn=localhost \
				--cert.san.domain=localhost --cert.san.ip=127.0.0.1 --signerCert=/certs/root_ca.crt --signerKey=/certs/root_ca.key \
				--cert.signatureAlgorithm=$(CERT_TOOL_SIGN_ALG) --cert.ellipticCurve=$(CERT_TOOL_ELLIPTIC_CURVE)
	docker run \
		--rm -v $(WORKING_DIRECTORY)/.tmp/certs:/certs \
		--user $(USER_ID):$(GROUP_ID) \
		${CERT_TOOL_IMAGE} \
		--cmd.generateIdentityCertificate=$(CLOUD_SID) --outCert=/certs/coap.crt --outKey=/certs/coap.key \
				--cert.san.domain=localhost --cert.san.ip=127.0.0.1 --signerCert=/certs/root_ca.crt --signerKey=/certs/root_ca.key \
				--cert.signatureAlgorithm=$(CERT_TOOL_SIGN_ALG) --cert.ellipticCurve=$(CERT_TOOL_ELLIPTIC_CURVE)
	cat $(WORKING_DIRECTORY)/.tmp/certs/http.crt > $(WORKING_DIRECTORY)/.tmp/certs/mongo.key
	cat $(WORKING_DIRECTORY)/.tmp/certs/http.key >> $(WORKING_DIRECTORY)/.tmp/certs/mongo.key

privateKeys:
	mkdir -p $(WORKING_DIRECTORY)/.tmp/privKeys
	openssl genrsa -out $(WORKING_DIRECTORY)/.tmp/privKeys/idTokenKey.pem 4096
	openssl ecparam -name prime256v1 -genkey -noout -out $(WORKING_DIRECTORY)/.tmp/privKeys/accessTokenKey.pem
	openssl ecparam -name prime256v1 -genkey -noout -out $(WORKING_DIRECTORY)/.tmp/privKeys/m2mAccessTokenKey.pem

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

scylla/clean:
	docker rm -f scylla || true
	sudo rm -rf $(WORKING_DIRECTORY)/.tmp/scylla || true

scylla: scylla/clean
	mkdir -p $(WORKING_DIRECTORY)/.tmp/scylla/data $(WORKING_DIRECTORY)/.tmp/scylla/commitlog $(WORKING_DIRECTORY)/.tmp/scylla/hints $(WORKING_DIRECTORY)/.tmp/scylla/view_hints $(WORKING_DIRECTORY)/.tmp/scylla/etc
	docker run --rm \
		-v $(WORKING_DIRECTORY)/.tmp/scylla/etc:/etc-scylla-tmp \
		--entrypoint /bin/cp \
		scylladb/scylla \
		/etc/scylla/scylla.yaml /etc-scylla-tmp/scylla.yaml
	sudo chown $(shell whoami) $(WORKING_DIRECTORY)/.tmp/scylla/etc/scylla.yaml

	yq -i '.server_encryption_options.internode_encryption="all"' $(WORKING_DIRECTORY)/.tmp/scylla/etc/scylla.yaml
	yq -i '.server_encryption_options.certificate="/certs/http.crt"' $(WORKING_DIRECTORY)/.tmp/scylla/etc/scylla.yaml
	yq -i '.server_encryption_options.keyfile="/certs/http.key"' $(WORKING_DIRECTORY)/.tmp/scylla/etc/scylla.yaml
	yq -i '.server_encryption_options.truststore="/certs/root_ca.crt"' $(WORKING_DIRECTORY)/.tmp/scylla/etc/scylla.yaml
	yq -i '.server_encryption_options.require_client_auth=true' $(WORKING_DIRECTORY)/.tmp/scylla/etc/scylla.yaml

	yq -i '.client_encryption_options.enabled=true' $(WORKING_DIRECTORY)/.tmp/scylla/etc/scylla.yaml
	yq -i '.client_encryption_options.certificate="/certs/http.crt"' $(WORKING_DIRECTORY)/.tmp/scylla/etc/scylla.yaml
	yq -i '.client_encryption_options.keyfile="/certs/http.key"' $(WORKING_DIRECTORY)/.tmp/scylla/etc/scylla.yaml
	yq -i '.client_encryption_options.truststore="/certs/root_ca.crt"' $(WORKING_DIRECTORY)/.tmp/scylla/etc/scylla.yaml
	yq -i '.client_encryption_options.require_client_auth=true' $(WORKING_DIRECTORY)/.tmp/scylla/etc/scylla.yaml

	yq -i '.api_port=11000' $(WORKING_DIRECTORY)/.tmp/scylla/etc/scylla.yaml
	yq -i '.prometheus_port=0' $(WORKING_DIRECTORY)/.tmp/scylla/etc/scylla.yaml

	yq -i 'del(.native_transport_port)' $(WORKING_DIRECTORY)/.tmp/scylla/etc/scylla.yaml
	yq -i '.native_transport_port_ssl=9142' $(WORKING_DIRECTORY)/.tmp/scylla/etc/scylla.yaml

	yq -i 'del(.native_shard_aware_transport_port)' $(WORKING_DIRECTORY)/.tmp/scylla/etc/scylla.yaml
	yq -i '.native_shard_aware_transport_port_ssl=19142' $(WORKING_DIRECTORY)/.tmp/scylla/etc/scylla.yaml

	docker run \
		-d \
		--network=host \
		--name=scylla \
		-v $(WORKING_DIRECTORY)/.tmp/scylla/etc/scylla.yaml:/etc/scylla/scylla.yaml \
		-v $(WORKING_DIRECTORY)/.tmp/scylla:/var/lib/scylla \
		-v $(WORKING_DIRECTORY)/.tmp/certs:/certs \
		scylladb/scylla --developer-mode 1 --listen-address 127.0.0.1

	while true; do \
		i=$$((i+1)); \
		if openssl s_client -connect 127.0.0.1:9142 -cert $(WORKING_DIRECTORY)/.tmp/certs/http.crt -key $(WORKING_DIRECTORY)/.tmp/certs/http.key <<< "Q" 2>/dev/null > /dev/null; then \
			break; \
		fi; \
		echo "Try to reconnect to scylla(127.0.0.1:9142) $$i"; \
		sleep 1; \
	done

.PHONY: scylla

mongo: certificates
	mkdir -p $(WORKING_DIRECTORY)/.tmp/mongo
	docker run \
		-d \
		--network=host \
		--name=mongo \
		-v $(WORKING_DIRECTORY)/.tmp/mongo:/data/db \
		-v $(WORKING_DIRECTORY)/.tmp/certs:/certs --user $(USER_ID):$(GROUP_ID) \
		mongo --tlsMode requireTLS --wiredTigerCacheSizeGB 1 --tlsCAFile  /certs/root_ca.crt --tlsCertificateKeyFile /certs/mongo.key

http-gateway-www:
	@mkdir -p $(WORKING_DIRECTORY)/.tmp/usr/local/www
	@cp -r $(WORKING_DIRECTORY)/http-gateway/web/public/* $(WORKING_DIRECTORY)/.tmp/usr/local/www/

# standard device
DEVICE_SIMULATOR_NAME := devsim
DEVICE_SIMULATOR_IMG := ghcr.io/iotivity/iotivity-lite/cloud-server-debug:vnext
# device with /oic/res observable
# note: iotivity-lite runs only grpc-gateway tests and this second device is not started; thus 
# the grpc-gateway are expected to succeed with a single non-oic/rec observable device
DEVICE_SIMULATOR_RES_OBSERVABLE_NAME := devsim-resobs
DEVICE_SIMULATOR_RES_OBSERVABLE_IMG := ghcr.io/iotivity/iotivity-lite/cloud-server-discovery-resource-observable-debug:vnext

# Pull latest device simulator with given name and run it
#
# Parameters:
#   $(1): name, used for:
#          - name of working directory for the device simulator (.tmp/$(1))
#          - name of the docker container
#          - name of the simulator ("$(1)-$(SIMULATOR_NAME_SUFFIX)")
define RUN-DOCKER-DEVICE
	mkdir -p "$(WORKING_DIRECTORY)/.tmp/$(1)" ; \
	mkdir -p "$(WORKING_DIRECTORY)/.tmp/$(1)/cloud_server_creds" ; \
	docker pull $(2) ; \
	docker run \
		-d \
		--privileged \
		--name=$(1) \
		--network=host \
		-v $(WORKING_DIRECTORY)/.tmp/$(1):/tmp \
		-v $(WORKING_DIRECTORY)/.tmp/$(1)/cloud_server_creds:/cloud_server_creds \
		$(2) \
		$(1)-$(SIMULATOR_NAME_SUFFIX)
endef

define CLEAN-DOCKER-DEVICE
	sudo rm -rf $(WORKING_DIRECTORY)/.tmp/$(1) || true
endef

define REMOVE-DOCKER-DEVICE
	docker stop --time 300 $(1) || true
	docker rm -f $(1) || true
endef

simulators/remove:
	$(call REMOVE-DOCKER-DEVICE,$(DEVICE_SIMULATOR_NAME))
	$(call REMOVE-DOCKER-DEVICE,$(DEVICE_SIMULATOR_RES_OBSERVABLE_NAME))
.PHONY: simulators/remove

simulators/clean: simulators/remove
	$(call CLEAN-DOCKER-DEVICE,$(DEVICE_SIMULATOR_NAME))
	$(call CLEAN-DOCKER-DEVICE,$(DEVICE_SIMULATOR_RES_OBSERVABLE_NAME))
.PHONY: simulators/clean

simulators: simulators/clean
	$(call RUN-DOCKER-DEVICE,$(DEVICE_SIMULATOR_NAME),$(DEVICE_SIMULATOR_IMG))
	$(call RUN-DOCKER-DEVICE,$(DEVICE_SIMULATOR_RES_OBSERVABLE_NAME),$(DEVICE_SIMULATOR_RES_OBSERVABLE_IMG))
.PHONY: simulators

BRIDGE_DEVICE_SRC_DIR = $(WORKING_DIRECTORY)/test/bridge-device
BRIDGE_DEVICE_IMAGE = ghcr.io/plgd-dev/device/bridge-device:vnext
BRIDGE_DEVICE_NAME = bridgedev
BRIDGE_DEVICE_ID ?= 8f596b43-29c0-4147-8b40-e99268ab30f7
BRIDGE_DEVICE_RESOURCES_PER_DEVICE ?= 3
BRIDGE_DEVICES_COUNT ?= 3

define SET-BRIDGE-DEVICE-CONFIG
	yq -i '.apis.coap.id = "$(BRIDGE_DEVICE_ID)"' $(1)
	yq -i '.apis.coap.externalAddresses=["127.0.0.1:15683","[::1]:15683"]' $(1)
	yq -i '.cloud.enabled=true' $(1)
	yq -i '.cloud.cloudID="$(CLOUD_SID)"' $(1)
	yq -i '.cloud.tls.caPoolPath="$(2)/certs/root_ca.crt"' $(1)
	yq -i '.cloud.tls.keyPath="$(2)/certs/coap.key"' $(1)
	yq -i '.cloud.tls.certPath="$(2)/certs/coap.crt"' $(1)
	yq -i '.numGeneratedBridgedDevices=$(BRIDGE_DEVICES_COUNT)' $(1)
	yq -i '.numResourcesPerDevice=$(BRIDGE_DEVICE_RESOURCES_PER_DEVICE)' $(1)
	yq -i '.thingDescription.enabled=true' $(1)
	yq -i '.thingDescription.file="$(2)/bridge/bridge-device.jsonld"' $(1)
endef

# config-docker.yaml -> copy of configuration with paths valid inside docker container
# config-test.yaml -> copy of configuration with paths valid on host machine
simulators/bridge/env: simulators/bridge/clean certificates
	mkdir -p $(WORKING_DIRECTORY)/.tmp/bridge
	cp $(BRIDGE_DEVICE_SRC_DIR)/bridge-device.jsonld $(WORKING_DIRECTORY)/.tmp/bridge/
	cp $(BRIDGE_DEVICE_SRC_DIR)/config.yaml $(WORKING_DIRECTORY)/.tmp/bridge/config-docker.yaml
	$(call SET-BRIDGE-DEVICE-CONFIG,$(WORKING_DIRECTORY)/.tmp/bridge/config-docker.yaml,)
	cp $(BRIDGE_DEVICE_SRC_DIR)/config.yaml $(WORKING_DIRECTORY)/.tmp/bridge/config-test.yaml
	$(call SET-BRIDGE-DEVICE-CONFIG,$(WORKING_DIRECTORY)/.tmp/bridge/config-test.yaml,$(WORKING_DIRECTORY)/.tmp)

.PHONY: simulators/bridge/env

define RUN-BRIDGE-DOCKER-DEVICE
	docker pull $(BRIDGE_DEVICE_IMAGE) ; \
	docker run \
		-d \
		--name=$(BRIDGE_DEVICE_NAME) \
		--network=host \
		-v $(WORKING_DIRECTORY)/.tmp/certs:/certs \
		-v $(WORKING_DIRECTORY)/.tmp/bridge:/bridge \
		$(BRIDGE_DEVICE_IMAGE) -config /bridge/config-docker.yaml
endef

simulators/bridge: simulators/bridge/env
	$(call RUN-BRIDGE-DOCKER-DEVICE)

.PHONY: simulators/bridge

simulators/bridge/clean:
	rm -rf $(WORKING_DIRECTORY)/.tmp/bridge || :
	$(call REMOVE-DOCKER-DEVICE,$(BRIDGE_DEVICE_NAME))

.PHONY: simulators/bridge/clean

simulators: simulators/bridge
simulators/clean: simulators/bridge/clean

env/test/mem: clean certificates nats mongo privateKeys scylla
.PHONY: env/test/mem

env: env/test/mem http-gateway-www simulators
.PHONY: env

define RUN-DOCKER
	docker run \
		--rm \
		--network=host \
		-v $(WORKING_DIRECTORY)/.tmp/bridge:/bridge \
		-v $(WORKING_DIRECTORY)/.tmp/certs:/certs \
		-v $(WORKING_DIRECTORY)/.tmp/coverage:/coverage \
		-v $(WORKING_DIRECTORY)/.tmp/report:/report \
		-v $(WORKING_DIRECTORY)/.tmp/privKeys:/privKeys \
		-v $(WORKING_DIRECTORY)/.tmp/usr/local/www:/usr/local/www \
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
		-e M2M_OAUTH_SERVER_PRIVATE_KEY=/privKeys/m2mAccessTokenKey.pem \
		-e TEST_HTTP_GW_WWW_ROOT=/usr/local/www \
		-e TEST_BRIDGE_DEVICE_CONFIG=/bridge/config-docker.yaml \
		hub-test \
		$(1) ;
endef

define RUN-TESTS-IN-DIRECTORY
	echo "Executing tests from $(1) directory"; \
	START_TIME=$$(date +%s); \
	COVERAGE_FILE=/coverage/$$(echo $(1) | sed -e "s/[\.\/]//g").coverage.txt ; \
	JSON_REPORT_FILE=$(WORKING_DIRECTORY)/.tmp/report/$$(echo $(1) | sed -e "s/[\.\/]//g").report.json ; \
	if [ -n "$${JSON_REPORT}" ]; then \
		$(call RUN-DOCKER, /bin/sh -c "$(2) go test -timeout=45m -race -p 1 -v $(1)... -covermode=atomic -coverprofile=$${COVERAGE_FILE} -json > $${JSON_REPORT_FILE}") \
	else \
		$(call RUN-DOCKER, /bin/sh -c "$(2) go test -timeout=45m -race -p 1 -v $(1)... -covermode=atomic -coverprofile=$${COVERAGE_FILE}") \
	fi ; \
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

# Run test with name $1 in over $2 directory with args $3 and env $4 variables
define RUN-TESTS
	echo "Executing tests $1"; \
	START_TIME=$$(date +%s); \
	COVERAGE_FILE=/coverage/$(1).coverage.txt ; \
	JSON_REPORT_FILE=$(WORKING_DIRECTORY)/.tmp/report/$(1).report.json ; \
	if [ -n "$${JSON_REPORT}" ]; then \
		$(call RUN-DOCKER, /bin/sh -c "$(4) go test $(3) $(2) -coverpkg=./... -covermode=atomic -coverprofile=$${COVERAGE_FILE} -json > $${JSON_REPORT_FILE}") \
	else \
		$(call RUN-DOCKER, /bin/sh -c "$(4) go test $(3) $(2) -coverpkg=./... -covermode=atomic -coverprofile=$${COVERAGE_FILE}") \
	fi ; \
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


define RUN-TESTS-UDP
	$(call RUN-TESTS,grpc-gateway-dtls,./grpc-gateway/service,-timeout=$(TEST_TIMEOUT) $(GO_BUILD_ARG) -p 1 -v -tags=test,\
		TEST_COAP_GATEWAY_UDP_ENABLED=$(TEST_COAP_GATEWAY_UDP_ENABLED) \
		TEST_COAP_GATEWAY_LOG_LEVEL=$(TEST_COAP_GATEWAY_LOG_LEVEL) TEST_COAP_GATEWAY_LOG_DUMP_BODY=$(TEST_COAP_GATEWAY_LOG_DUMP_BODY) \
		TEST_RESOURCE_AGGREGATE_LEVEL=$(TEST_RESOURCE_AGGREGATE_LEVEL) TEST_RESOURCE_AGGREGATE_LOG_DUMP_BODY=$(TEST_RESOURCE_AGGREGATE_LOG_DUMP_BODY) \
		TEST_GRPC_GATEWAY_LOG_LEVEL=$(TEST_GRPC_GATEWAY_LOG_LEVEL) TEST_GRPC_GATEWAY_LOG_DUMP_BODY=$(TEST_GRPC_GATEWAY_LOG_DUMP_BODY) \
		TEST_IDENTITY_STORE_LOG_LEVEL=$(TEST_IDENTITY_STORE_LOG_LEVEL) TEST_IDENTITY_STORE_LOG_DUMP_BODY=$(TEST_IDENTITY_STORE_LOG_DUMP_BODY) \
		TEST_SNIPPET_SERVICE_LOG_LEVEL=$(TEST_SNIPPET_SERVICE_LOG_LEVEL) TEST_SNIPPET_SERVICE_LOG_DUMP_BODY=$(TEST_SNIPPET_SERVICE_LOG_DUMP_BODY) \
		TEST_LEAD_RESOURCE_TYPE_FILTER=$(TEST_LEAD_RESOURCE_TYPE_FILTER) TEST_LEAD_RESOURCE_TYPE_REGEX_FILTER='$(TEST_LEAD_RESOURCE_TYPE_REGEX_FILTER)' TEST_LEAD_RESOURCE_TYPE_USE_UUID=$(TEST_LEAD_RESOURCE_TYPE_USE_UUID) \
		TEST_DATABASE=$(TEST_DATABASE))
	$(call RUN-TESTS,iotivity-lite-dtls,./test/iotivity-lite/service,-timeout=$(TEST_TIMEOUT) $(GO_BUILD_ARG) -p 1 -v -tags=test,\
		TEST_COAP_GATEWAY_UDP_ENABLED=$(TEST_COAP_GATEWAY_UDP_ENABLED) \
		TEST_COAP_GATEWAY_LOG_LEVEL=$(TEST_COAP_GATEWAY_LOG_LEVEL) TEST_COAP_GATEWAY_LOG_DUMP_BODY=$(TEST_COAP_GATEWAY_LOG_DUMP_BODY) \
		TEST_RESOURCE_AGGREGATE_LEVEL=$(TEST_RESOURCE_AGGREGATE_LEVEL) TEST_RESOURCE_AGGREGATE_LOG_DUMP_BODY=$(TEST_RESOURCE_AGGREGATE_LOG_DUMP_BODY) \
		TEST_GRPC_GATEWAY_LOG_LEVEL=$(TEST_GRPC_GATEWAY_LOG_LEVEL) TEST_GRPC_GATEWAY_LOG_DUMP_BODY=$(TEST_GRPC_GATEWAY_LOG_DUMP_BODY) \
		TEST_IDENTITY_STORE_LOG_LEVEL=$(TEST_IDENTITY_STORE_LOG_LEVEL) TEST_IDENTITY_STORE_LOG_DUMP_BODY=$(TEST_IDENTITY_STORE_LOG_DUMP_BODY) \
		TEST_SNIPPET_SERVICE_LOG_LEVEL=$(TEST_SNIPPET_SERVICE_LOG_LEVEL) TEST_SNIPPET_SERVICE_LOG_DUMP_BODY=$(TEST_SNIPPET_SERVICE_LOG_DUMP_BODY) \
		TEST_LEAD_RESOURCE_TYPE_FILTER=$(TEST_LEAD_RESOURCE_TYPE_FILTER) TEST_LEAD_RESOURCE_TYPE_REGEX_FILTER='$(TEST_LEAD_RESOURCE_TYPE_REGEX_FILTER)' TEST_LEAD_RESOURCE_TYPE_USE_UUID=$(TEST_LEAD_RESOURCE_TYPE_USE_UUID) \
		TEST_DATABASE=$(TEST_DATABASE))
endef

test: env hub-test
	@mkdir -p $(WORKING_DIRECTORY)/.tmp/home
	@mkdir -p $(WORKING_DIRECTORY)/.tmp/home/certificate-authority
	@mkdir -p $(WORKING_DIRECTORY)/.tmp/report
	@$(call RUN-TESTS,hub,./...,-timeout=$(TEST_TIMEOUT) $(GO_BUILD_ARG) -p 1 -v -tags=test,\
		TEST_COAP_GATEWAY_LOG_LEVEL=$(TEST_COAP_GATEWAY_LOG_LEVEL) TEST_COAP_GATEWAY_LOG_DUMP_BODY=$(TEST_COAP_GATEWAY_LOG_DUMP_BODY) \
		TEST_RESOURCE_AGGREGATE_LEVEL=$(TEST_RESOURCE_AGGREGATE_LEVEL) TEST_RESOURCE_AGGREGATE_LOG_DUMP_BODY=$(TEST_RESOURCE_AGGREGATE_LOG_DUMP_BODY) \
		TEST_GRPC_GATEWAY_LOG_LEVEL=$(TEST_GRPC_GATEWAY_LOG_LEVEL) TEST_GRPC_GATEWAY_LOG_DUMP_BODY=$(TEST_GRPC_GATEWAY_LOG_DUMP_BODY) \
		TEST_IDENTITY_STORE_LOG_LEVEL=$(TEST_IDENTITY_STORE_LOG_LEVEL) TEST_IDENTITY_STORE_LOG_DUMP_BODY=$(TEST_IDENTITY_STORE_LOG_DUMP_BODY) \
		TEST_SNIPPET_SERVICE_LOG_LEVEL=$(TEST_SNIPPET_SERVICE_LOG_LEVEL) TEST_SNIPPET_SERVICE_LOG_DUMP_BODY=$(TEST_SNIPPET_SERVICE_LOG_DUMP_BODY) \
		TEST_LEAD_RESOURCE_TYPE_FILTER=$(TEST_LEAD_RESOURCE_TYPE_FILTER) TEST_LEAD_RESOURCE_TYPE_REGEX_FILTER='$(TEST_LEAD_RESOURCE_TYPE_REGEX_FILTER)' TEST_LEAD_RESOURCE_TYPE_USE_UUID=$(TEST_LEAD_RESOURCE_TYPE_USE_UUID) \
		TEST_DATABASE=$(TEST_DATABASE))
ifeq ($(TEST_COAP_GATEWAY_UDP_ENABLED),true)
	@$(call RUN-TESTS-UDP)
endif

.PHONY: test

test/mem: env/test/mem hub-test
	@mkdir -p $(WORKING_DIRECTORY)/.tmp/home
	@mkdir -p $(WORKING_DIRECTORY)/.tmp/home/certificate-authority
	@mkdir -p $(WORKING_DIRECTORY)/.tmp/report
	@$(call RUN-TESTS,coap-gateway-mem,./coap-gateway/service,-timeout=$(TEST_TIMEOUT) -p 1 -v -tags=test_mem,\
		TEST_MEMORY_COAP_GATEWAY_NUM_DEVICES=$(TEST_MEMORY_COAP_GATEWAY_NUM_DEVICES) \
		TEST_MEMORY_COAP_GATEWAY_NUM_RESOURCES=$(TEST_MEMORY_COAP_GATEWAY_NUM_RESOURCES) \
		TEST_MEMORY_COAP_GATEWAY_EXPECTED_RSS_IN_MB=$(TEST_MEMORY_COAP_GATEWAY_EXPECTED_RSS_IN_MB) \
		TEST_MEMORY_COAP_GATEWAY_RESOURCE_DATA_SIZE=$(TEST_MEMORY_COAP_GATEWAY_RESOURCE_DATA_SIZE) \
		TEST_COAP_GATEWAY_LOG_LEVEL=$(TEST_COAP_GATEWAY_LOG_LEVEL) TEST_COAP_GATEWAY_LOG_DUMP_BODY=$(TEST_COAP_GATEWAY_LOG_DUMP_BODY) \
		TEST_RESOURCE_AGGREGATE_LEVEL=$(TEST_RESOURCE_AGGREGATE_LEVEL) TEST_RESOURCE_AGGREGATE_LOG_DUMP_BODY=$(TEST_RESOURCE_AGGREGATE_LOG_DUMP_BODY) \
		TEST_GRPC_GATEWAY_LOG_LEVEL=$(TEST_GRPC_GATEWAY_LOG_LEVEL) TEST_GRPC_GATEWAY_LOG_DUMP_BODY=$(TEST_GRPC_GATEWAY_LOG_DUMP_BODY) \
		TEST_IDENTITY_STORE_LOG_LEVEL=$(TEST_IDENTITY_STORE_LOG_LEVEL) TEST_IDENTITY_STORE_LOG_DUMP_BODY=$(TEST_IDENTITY_STORE_LOG_DUMP_BODY)\
		TEST_SNIPPET_SERVICE_LOG_LEVEL=$(TEST_SNIPPET_SERVICE_LOG_LEVEL) TEST_SNIPPET_SERVICE_LOG_DUMP_BODY=$(TEST_SNIPPET_SERVICE_LOG_DUMP_BODY) \
		TEST_LEAD_RESOURCE_TYPE_FILTER=$(TEST_LEAD_RESOURCE_TYPE_FILTER) TEST_LEAD_RESOURCE_TYPE_REGEX_FILTER='$(TEST_LEAD_RESOURCE_TYPE_REGEX_FILTER)' TEST_LEAD_RESOURCE_TYPE_USE_UUID=$(TEST_LEAD_RESOURCE_TYPE_USE_UUID) \
		TEST_DATABASE=$(TEST_DATABASE))
.PHONY: test/mem

DIRECTORIES:=$(shell ls -d ./*/)
DIRECTORIES+=./test/iotivity-lite/

test-targets := $(addprefix test-,$(patsubst ./%/,%,$(DIRECTORIES)))

$(test-targets): %: env hub-test
	@mkdir -p $(WORKING_DIRECTORY)/.tmp/home
	@mkdir -p $(WORKING_DIRECTORY)/.tmp/home/certificate-authority
	@mkdir -p $(WORKING_DIRECTORY)/.tmp/report
	@readonly TARGET_DIRECTORY=$(patsubst test-%,./%/,$@) ; \
	if ! go list -f '{{.GoFiles}}' $$TARGET_DIRECTORY... 2>/dev/null | grep go > /dev/null 2>&1; then \
		echo "No golang files detected, directory $$TARGET_DIRECTORY skipped"; \
		exit 0; \
	fi ; \
	$(call RUN-TESTS-IN-DIRECTORY,$(patsubst test-%,./%/,$@),\
		TEST_COAP_GATEWAY_UDP_ENABLED=$(TEST_COAP_GATEWAY_UDP_ENABLED) \
		TEST_COAP_GATEWAY_LOG_LEVEL=$(TEST_COAP_GATEWAY_LOG_LEVEL) TEST_COAP_GATEWAY_LOG_DUMP_BODY=$(TEST_COAP_GATEWAY_LOG_DUMP_BODY) \
		TEST_RESOURCE_AGGREGATE_LEVEL=$(TEST_RESOURCE_AGGREGATE_LEVEL) TEST_RESOURCE_AGGREGATE_LOG_DUMP_BODY=$(TEST_RESOURCE_AGGREGATE_LOG_DUMP_BODY) \
		TEST_GRPC_GATEWAY_LOG_LEVEL=$(TEST_GRPC_GATEWAY_LOG_LEVEL) TEST_GRPC_GATEWAY_LOG_DUMP_BODY=$(TEST_GRPC_GATEWAY_LOG_DUMP_BODY) \
		TEST_IDENTITY_STORE_LOG_LEVEL=$(TEST_IDENTITY_STORE_LOG_LEVEL) TEST_IDENTITY_STORE_LOG_DUMP_BODY=$(TEST_IDENTITY_STORE_LOG_DUMP_BODY) \
		TEST_SNIPPET_SERVICE_LOG_LEVEL=$(TEST_SNIPPET_SERVICE_LOG_LEVEL) TEST_SNIPPET_SERVICE_LOG_DUMP_BODY=$(TEST_SNIPPET_SERVICE_LOG_DUMP_BODY) \
		TEST_LEAD_RESOURCE_TYPE_FILTER=$(TEST_LEAD_RESOURCE_TYPE_FILTER) TEST_LEAD_RESOURCE_TYPE_REGEX_FILTER='$(TEST_LEAD_RESOURCE_TYPE_REGEX_FILTER)' TEST_LEAD_RESOURCE_TYPE_USE_UUID=$(TEST_LEAD_RESOURCE_TYPE_USE_UUID) \
		TEST_DATABASE=$(TEST_DATABASE))

.PHONY: $(test-targets)

build: $(SUBDIRS)

clean: simulators/clean scylla/clean
	docker rm -f mongo || true
	docker rm -f nats || true
	docker rm -f nats-cloud-connector || true
	sudo rm -rf ./.tmp/certs || true
	sudo rm -rf ./.tmp/mongo || true
	sudo rm -rf ./.tmp/home || true
	sudo rm -rf ./.tmp/privateKeys || true
	sudo rm -rf ./.tmp/coverage || true
	sudo rm -rf ./.tmp/report || true
	sudo rm -rf ./.tmp/usr || true

proto/generate: $(SUBDIRS)
	protoc -I=. -I=$(GOPATH)/src --go_out=$(GOPATH)/src $(WORKING_DIRECTORY)/pkg/net/grpc/stub.proto
	protoc -I=. -I=$(GOPATH)/src --go-grpc_out=$(GOPATH)/src $(WORKING_DIRECTORY)/pkg/net/grpc/stub.proto
	mv $(WORKING_DIRECTORY)/pkg/net/grpc/stub.pb.go $(WORKING_DIRECTORY)/pkg/net/grpc/stub.pb_test.go
	mv $(WORKING_DIRECTORY)/pkg/net/grpc/stub_grpc.pb.go $(WORKING_DIRECTORY)/pkg/net/grpc/stub_grpc.pb_test.go
push: hub-build $(SUBDIRS)

$(SUBDIRS):
	$(MAKE) -C $@ $(MAKECMDGOALS) LATEST_TAG=$(BUILD_TAG)
