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
CERT_PATH ?= $(WORKING_DIRECTORY)/.tmp/certs
USER_ID := $(shell id -u)
GROUP_ID := $(shell id -g)
TEST_CHECK_RACE ?= false
ifeq ($(TEST_CHECK_RACE),true)
GO_BUILD_ARG := -race
else
GO_BUILD_ARG := $(GO_BUILD_ARG)
endif
TEST_TIMEOUT ?= 1h
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
ifeq ($(TEST_MONGODB_VERBOSE),true)
MONGODB_ARGS := -vvvvv
else
MONGODB_ARGS :=
endif
TEST_LEAD_RESOURCE_TYPE_FILTER ?=
TEST_LEAD_RESOURCE_TYPE_REGEX_FILTER ?=
TEST_LEAD_RESOURCE_TYPE_USE_UUID ?= false
TEST_DPS_UDP_ENABLED ?= false
# supported values: ECDSA-SHA256, ECDSA-SHA384, ECDSA-SHA512
CERT_TOOL_SIGN_ALG ?= ECDSA-SHA256
# supported values: P256, P384, P521
CERT_TOOL_ELLIPTIC_CURVE ?= P256
CERT_TOOL_IMAGE = ghcr.io/plgd-dev/hub/cert-tool:vnext

default: build

hub-test:
	docker build \
		--network=host \
		--tag hub-test \
		-f Dockerfile.test \
		.

certificates:
	mkdir -p $(CERT_PATH)
	docker pull $(CERT_TOOL_IMAGE)
	docker run --rm -v $(CERT_PATH):/certs --user $(USER_ID):$(GROUP_ID) ${CERT_TOOL_IMAGE} \
		--cmd.generateRootCA --outCert=/certs/root_ca.crt --outKey=/certs/root_ca.key --cert.subject.cn=RootCA \
		--cert.signatureAlgorithm=$(CERT_TOOL_SIGN_ALG) --cert.ellipticCurve=$(CERT_TOOL_ELLIPTIC_CURVE) \
		--cert.validFrom=2000-01-01T12:00:00Z --cert.validFor=876000h
	docker run --rm -v $(CERT_PATH):/certs --user $(USER_ID):$(GROUP_ID) ${CERT_TOOL_IMAGE} \
		--cmd.generateCertificate --outCert=/certs/http.crt --outKey=/certs/http.key --cert.subject.cn=localhost \
		--cert.san.domain=localhost --cert.san.ip=127.0.0.1 --signerCert=/certs/root_ca.crt --signerKey=/certs/root_ca.key \
		--cert.signatureAlgorithm=$(CERT_TOOL_SIGN_ALG) --cert.ellipticCurve=$(CERT_TOOL_ELLIPTIC_CURVE)
	docker run --rm -v $(CERT_PATH):/certs --user $(USER_ID):$(GROUP_ID) ${CERT_TOOL_IMAGE} \
		--cmd.generateIdentityCertificate=$(CLOUD_SID) --outCert=/certs/coap.crt --outKey=/certs/coap.key \
		--cert.san.domain=localhost --cert.san.ip=127.0.0.1 --signerCert=/certs/root_ca.crt --signerKey=/certs/root_ca.key \
		--cert.signatureAlgorithm=$(CERT_TOOL_SIGN_ALG) --cert.ellipticCurve=$(CERT_TOOL_ELLIPTIC_CURVE)
	cat $(CERT_PATH)/http.crt > $(CERT_PATH)/mongo.key
	cat $(CERT_PATH)/http.key >> $(CERT_PATH)/mongo.key
	mkdir -p $(CERT_PATH)/device
	cp $(CERT_PATH)/root_ca.crt $(CERT_PATH)/device/dpsca.pem
	cp $(CERT_PATH)/root_ca.key $(CERT_PATH)/device/dpscakey.pem
	docker run --rm -v $(CERT_PATH)/device:/certs --user $(USER_ID):$(GROUP_ID) ${CERT_TOOL_IMAGE} \
		--signerCert=/certs/dpsca.pem --signerKey=/certs/dpscakey.pem  --outCert=/certs/intermediatecacrt.pem --outKey=/certs/intermediatecakey.pem \
		--cert.basicConstraints.maxPathLen=0 --cert.subject.cn="intermediateCA" --cmd.generateIntermediateCA \
		--cert.signatureAlgorithm=$(CERT_TOOL_SIGN_ALG) --cert.ellipticCurve=$(CERT_TOOL_ELLIPTIC_CURVE)
	docker run --rm -v $(CERT_PATH)/device:/certs --user $(USER_ID):$(GROUP_ID) ${CERT_TOOL_IMAGE} \
		--signerCert=/certs/intermediatecacrt.pem --signerKey=/certs/intermediatecakey.pem --outCert=/certs/mfgcrt.pem --outKey=/certs/mfgkey.pem \
		--cert.san.domain=localhost --cert.san.ip=127.0.0.1 --cert.subject.cn="mfg" --cmd.generateCertificate \
		--cert.signatureAlgorithm=$(CERT_TOOL_SIGN_ALG) --cert.ellipticCurve=$(CERT_TOOL_ELLIPTIC_CURVE)
	docker run --rm -v $(CERT_PATH)/device:/certs --user $(USER_ID):$(GROUP_ID) ${CERT_TOOL_IMAGE} \
		--cmd.generateRootCA --outCert=/certs/root_ca_alt.crt --outKey=/certs/root_ca_alt.key --cert.subject.cn=RootCA \
		--cert.signatureAlgorithm=$(CERT_TOOL_SIGN_ALG) --cert.ellipticCurve=$(CERT_TOOL_ELLIPTIC_CURVE) \
		--cert.validFrom=2000-01-01T12:00:00Z --cert.validFor=876000h

certificates/clean:
	( [ -n "$(CERT_PATH)" ] && sudo rm -rf $(CERT_PATH) ) || :

.PHONY: certificates certificates/clean

privateKeys:
	mkdir -p $(WORKING_DIRECTORY)/.tmp/privKeys
	openssl genrsa -out $(WORKING_DIRECTORY)/.tmp/privKeys/idTokenKey.pem 4096
	openssl ecparam -name prime256v1 -genkey -noout -out $(WORKING_DIRECTORY)/.tmp/privKeys/accessTokenKey.pem
	openssl ecparam -name prime256v1 -genkey -noout -out $(WORKING_DIRECTORY)/.tmp/privKeys/m2mAccessTokenKey.pem

privateKeys/clean:
	sudo rm -rf $(WORKING_DIRECTORY)/.tmp/privKeys || :

.PHONY: privateKeys privateKeys/clean

nats: certificates
	mkdir -p $(WORKING_DIRECTORY)/.tmp/jetstream/cloud
	mkdir -p $(WORKING_DIRECTORY)/.tmp/jetstream/cloud-connector
	docker run \
	    -d \
		--network=host \
		--name=nats \
		-v $(CERT_PATH):/certs \
		-v $(WORKING_DIRECTORY)/.tmp/jetstream/cloud:/data \
		--user $(USER_ID):$(GROUP_ID) \
		nats --jetstream --store_dir /data --tls --tlsverify --tlscert=/certs/http.crt --tlskey=/certs/http.key --tlscacert=/certs/root_ca.crt
	docker run \
	    -d \
		--network=host \
		--name=nats-cloud-connector \
		-v $(CERT_PATH):/certs \
		-v $(WORKING_DIRECTORY)/.tmp/jetstream/cloud-connector:/data \
		--user $(USER_ID):$(GROUP_ID) \
		nats --jetstream --store_dir /data --port 34222 --tls --tlsverify --tlscert=/certs/http.crt --tlskey=/certs/http.key --tlscacert=/certs/root_ca.crt

nats/clean:
	docker rm -f nats || :
	docker rm -f nats-cloud-connector || :
	sudo rm -rf $(WORKING_DIRECTORY)/.tmp/jetstream || :

.PHONY: nats nats/clean

scylla/clean:
	docker rm -f scylla || :
	sudo rm -rf $(WORKING_DIRECTORY)/.tmp/scylla || :

scylla: scylla/clean
	mkdir -p $(WORKING_DIRECTORY)/.tmp/scylla/etc ;
	docker run --rm \
		--user $(USER_ID):$(GROUP_ID) \
		-v $(WORKING_DIRECTORY)/.tmp/scylla/etc:/etc-scylla-tmp \
		--entrypoint /bin/cp \
		scylladb/scylla \
		/etc/scylla/scylla.yaml /etc-scylla-tmp/scylla.yaml
	sudo chown $(shell whoami):$(shell id -gn) $(WORKING_DIRECTORY)/.tmp/scylla/etc/scylla.yaml

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
		--name=scylla \
		--network=host \
		-v $(WORKING_DIRECTORY)/.tmp/scylla/etc/scylla.yaml:/etc/scylla/scylla.yaml \
		-v $(CERT_PATH):/certs \
		scylladb/scylla --developer-mode 1 --listen-address 127.0.0.1

	@MAX_RETRIES=300; \
	i=0; \
	while true; do \
		i=$$((i+1)); \
		if openssl s_client -connect 127.0.0.1:9142 -cert $(CERT_PATH)/http.crt -key $(CERT_PATH)/http.key <<< "Q" 2>/dev/null > /dev/null; then \
			break; \
		fi; \
		if [ "$$i" -ge "$$MAX_RETRIES" ]; then \
			echo "Scylla did not become ready after $$i seconds. Exiting."; \
			exit 1; \
		fi; \
		echo "Try to reconnect to scylla(127.0.0.1:9142) $$i"; \
		sleep 1; \
	done

.PHONY: scylla scylla/clean

# Pull latest mongo and start its in replica set
#
# Parameters:
#   $(1): name, used for:
#          - name of working directory for the device simulator (.tmp/$(1))
#          - name of the docker container
#   $(2): additional opts
define RUN-DOCKER-MONGO
	mkdir -p $(WORKING_DIRECTORY)/.tmp/$(1) ; \
	docker run \
		-d \
		--network=host \
		--name=$(1) \
		-v $(WORKING_DIRECTORY)/.tmp/$(1):/data/db \
		-v $(CERT_PATH):/certs --user $(USER_ID):$(GROUP_ID) \
		mongo $(2) --tlsMode requireTLS --wiredTigerCacheSizeGB 1 --tlsCAFile /certs/root_ca.crt \
			--tlsCertificateKeyFile /certs/mongo.key
endef

MONGODB_REPLICA_0 := mongo0
MONGODB_REPLICA_0_PORT := 27017
MONGODB_REPLICA_1 := mongo1
MONGODB_REPLICA_1_PORT := 27018
MONGODB_REPLICA_2 := mongo2
MONGODB_REPLICA_2_PORT := 27019

mongo: certificates
	$(call RUN-DOCKER-MONGO,$(MONGODB_REPLICA_0),$(MONGODB_ARGS) --replSet myReplicaSet --bind_ip localhost --port $(MONGODB_REPLICA_0_PORT))
	$(call RUN-DOCKER-MONGO,$(MONGODB_REPLICA_1),$(MONGODB_ARGS) --replSet myReplicaSet --bind_ip localhost --port $(MONGODB_REPLICA_1_PORT))
	$(call RUN-DOCKER-MONGO,$(MONGODB_REPLICA_2),$(MONGODB_ARGS) --replSet myReplicaSet --bind_ip localhost --port $(MONGODB_REPLICA_2_PORT))
	COUNTER=0; \
	while [[ $${COUNTER} -lt 30 ]]; do \
		echo "Checking mongodb connection ($${COUNTER}):"; \
		if docker exec $(MONGODB_REPLICA_0) mongosh --quiet --tls --tlsCAFile /certs/root_ca.crt \
			--tlsCertificateKeyFile /certs/mongo.key --eval "db.adminCommand('ping')"; then \
			break; \
		fi ; \
		sleep 1; \
		let COUNTER+=1; \
	done; \
	docker exec $(MONGODB_REPLICA_0) mongosh --tls --tlsCAFile /certs/root_ca.crt --tlsCertificateKeyFile /certs/mongo.key \
		--eval "rs.initiate({ \
		_id: \"myReplicaSet\", \
		members: [ \
			{_id: 0, host: \"localhost:$(MONGODB_REPLICA_0_PORT)\"}, \
			{_id: 1, host: \"localhost:$(MONGODB_REPLICA_1_PORT)\"}, \
			{_id: 2, host: \"localhost:$(MONGODB_REPLICA_2_PORT)\"} \
		] \
	})"

mongo/clean:
	$(call REMOVE-DOCKER-DEVICE,$(MONGODB_REPLICA_0))
	$(call CLEAN-DOCKER-DEVICE,$(MONGODB_REPLICA_0))
	sudo rm -rf ./.tmp/$(MONGODB_REPLICA_0) || :
	$(call REMOVE-DOCKER-DEVICE,$(MONGODB_REPLICA_1))
	$(call CLEAN-DOCKER-DEVICE,$(MONGODB_REPLICA_1))
	sudo rm -rf ./.tmp/$(MONGODB_REPLICA_1) || :
	$(call REMOVE-DOCKER-DEVICE,$(MONGODB_REPLICA_2))
	$(call CLEAN-DOCKER-DEVICE,$(MONGODB_REPLICA_2))
	sudo rm -rf ./.tmp/$(MONGODB_REPLICA_2) || :

.PHONY: mongo mongo/clean

mongo-no-replicas: certificates
	$(call RUN-DOCKER-MONGO,mongo,$(MONGODB_ARGS))

mongo-no-replicas/clean:
	$(call REMOVE-DOCKER-DEVICE,mongo)
	$(call CLEAN-DOCKER-DEVICE,mongo)
	sudo rm -rf ./.tmp/mongo || :

.PHONY: mongo-no-replicas mongo-no-replicas/clean

http-gateway-www:
	@mkdir -p $(WORKING_DIRECTORY)/.tmp/usr/local/www
	@cp -r $(WORKING_DIRECTORY)/http-gateway/web/public/* $(WORKING_DIRECTORY)/.tmp/usr/local/www/

http-gateway-www/clean:
	sudo rm -rf ./.tmp/usr || true

.PHONY: http-gateway-www http-gateway-www/clean

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
#   $(2): docker image
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
	sudo rm -rf $(WORKING_DIRECTORY)/.tmp/$(1) || :
endef

define REMOVE-DOCKER-DEVICE
	docker stop --time 300 $(1) || :
	docker rm -f $(1) || :
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

.PHONY: simulators simulators/remove simulators/clean

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
		-v $(CERT_PATH):/certs \
		-v $(WORKING_DIRECTORY)/.tmp/bridge:/bridge \
		$(BRIDGE_DEVICE_IMAGE) -config /bridge/config-docker.yaml
endef

simulators/bridge: simulators/bridge/env
	$(call RUN-BRIDGE-DOCKER-DEVICE)
.PHONY: simulators/bridge

simulators/bridge/remove:
	rm -rf $(WORKING_DIRECTORY)/.tmp/bridge || :
.PHONY: simulators/bridge/remove

simulators/bridge/clean: simulators/bridge/remove
	$(call REMOVE-DOCKER-DEVICE,$(BRIDGE_DEVICE_NAME))
.PHONY: simulators/bridge/clean

simulators: simulators/bridge
simulators/clean: simulators/bridge/clean

# device provisioning service
ifeq ($(TEST_DPS_UDP_ENABLED),true)
DPS_ENDPOINT ?= coaps://127.0.0.1:20030
else
DPS_ENDPOINT ?= coaps+tcp://127.0.0.1:20030
endif
DPS_DEVICE_LOG_LEVEL ?= debug
DPS_DEVICE_OC_LOG_LEVEL ?= info
DPS_DEVICE_SIMULATOR_OBT_NAME := dps-devsim-obt
DPS_DEVICE_SIMULATOR_NAME := dps-devsim
DPS_DEVICE_SIMULATOR_IMG := ghcr.io/iotivity/iotivity-lite/dps-cloud-server-debug:vnext

# Pull latest DPS device simulator with given name and run it
#
# Parameters:
#   $(1): name, used for:
#          - name of working directory for the device simulator (.tmp/$(1))
#          - name of the docker container
#          - name of the simulator ("$(1)-$(SIMULATOR_NAME_SUFFIX)")
#   $(2): docker image
#   $(3): endpoint
define RUN-DPS-DOCKER-DEVICE
	mkdir -p "$(WORKING_DIRECTORY)/.tmp/$(1)" ; \
	docker pull $(2) ; \
	docker run \
		-d \
		--privileged \
		--name=$(1) \
		--network=host \
		-v $(WORKING_DIRECTORY)/.tmp/$(1):/tmp \
		-v $(CERT_PATH)/device:/dps/pki_certs \
		$(2) \
		$(1)-$(SIMULATOR_NAME_SUFFIX) --create-conf-resource --cloud-observer-max-retry 10 --expiration-limit 10 --retry-configuration 5 \
			--log-level $(DPS_DEVICE_LOG_LEVEL) --oc-log-level $(DPS_DEVICE_OC_LOG_LEVEL) $(3)
endef

simulators/dps/remove:
	$(call REMOVE-DOCKER-DEVICE,$(DPS_DEVICE_SIMULATOR_NAME))
	$(call REMOVE-DOCKER-DEVICE,$(DPS_DEVICE_SIMULATOR_OBT_NAME))
.PHONY: simulators/dps/remove

simulators/dps/clean: simulators/dps/remove
	$(call CLEAN-DOCKER-DEVICE,$(DPS_DEVICE_SIMULATOR_NAME))
	$(call CLEAN-DOCKER-DEVICE,$(DPS_DEVICE_SIMULATOR_OBT_NAME))
.PHONY: simulators/dps/clean

simulators/dps: simulators/dps/clean
	$(call RUN-DPS-DOCKER-DEVICE,$(DPS_DEVICE_SIMULATOR_NAME),$(DPS_DEVICE_SIMULATOR_IMG),--wait-for-reset $(DPS_ENDPOINT))
	$(call RUN-DPS-DOCKER-DEVICE,$(DPS_DEVICE_SIMULATOR_OBT_NAME),$(DPS_DEVICE_SIMULATOR_IMG),"")
.PHONY: simulators/dps

simulators/clean: simulators/dps/clean
simulators: simulators/dps

env: clean certificates nats privateKeys http-gateway-www mongo simulators
env/test/mem: clean certificates nats privateKeys

ifeq ($(TEST_DATABASE),mongodb)
# the measure memory test cases seems to be much slower with replica sets running
env/test/mem: mongo-no-replicas
else
# test uses mongodb for most tests, but scylla can be enabled for some; so we always need mongodb to be running
# but scylla needs to be started only if TEST_DATABASE=cqldb
env: scylla
# test/mem uses either mongodb or scylla, so just one needs to be started
env/test/mem: scylla
endif

.PHONY: env env/test/mem

define RUN-DOCKER
	docker run \
		--rm \
		--network=host \
		-v $(CERT_PATH):/certs \
		-v $(WORKING_DIRECTORY)/.tmp/bridge:/bridge \
		-v $(WORKING_DIRECTORY)/.tmp/coverage:/coverage \
		-v $(WORKING_DIRECTORY)/.tmp/report:/report \
		-v $(WORKING_DIRECTORY)/.tmp/privKeys:/privKeys \
		-v $(WORKING_DIRECTORY)/.tmp/usr/local/www:/usr/local/www \
		-v /var/run/docker.sock:/var/run/docker.sock \
		-e TEST_CLOUD_SID=$(CLOUD_SID) \
		-e LISTEN_FILE_CA_POOL=/certs/root_ca.crt \
		-e LISTEN_FILE_CERT_DIR_PATH=/certs \
		-e LISTEN_FILE_CERT_NAME=http.crt \
		-e LISTEN_FILE_CERT_KEY_NAME=http.key \
		-e TEST_COAP_GW_CERT_FILE=/certs/coap.crt \
		-e TEST_COAP_GW_KEY_FILE=/certs/coap.key \
		-e TEST_ROOT_CA_CERT=/certs/root_ca.crt \
		-e TEST_ROOT_CA_KEY=/certs/root_ca.key \
		-e TEST_DPS_ROOT_CA_CERT_ALT=/certs/device/root_ca_alt.crt \
		-e TEST_DPS_ROOT_CA_KEY_ALT=/certs/device/root_ca_alt.key \
		-e TEST_DPS_INTERMEDIATE_CA_CERT=/certs/device/intermediatecacrt.pem \
		-e TEST_DPS_INTERMEDIATE_CA_KEY=/certs/device/intermediatecakey.pem \
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
		$(call RUN-DOCKER, /bin/sh -c "$(3) go test $(2) $(1)... -covermode=atomic -coverprofile=$${COVERAGE_FILE} -json > $${JSON_REPORT_FILE}") \
	else \
		$(call RUN-DOCKER, /bin/sh -c "$(3) go test $(2) $(1)... -covermode=atomic -coverprofile=$${COVERAGE_FILE}") \
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
		TEST_DPS_UDP_ENABLED=$(TEST_DPS_UDP_ENABLED) TEST_DATABASE=$(TEST_DATABASE))
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
	$(call RUN-TESTS-IN-DIRECTORY,$(patsubst test-%,./%/,$@),-timeout=$(TEST_TIMEOUT) $(GO_BUILD_ARG) -p 1 -v -tags=test,\
		TEST_COAP_GATEWAY_UDP_ENABLED=$(TEST_COAP_GATEWAY_UDP_ENABLED) \
		TEST_COAP_GATEWAY_LOG_LEVEL=$(TEST_COAP_GATEWAY_LOG_LEVEL) TEST_COAP_GATEWAY_LOG_DUMP_BODY=$(TEST_COAP_GATEWAY_LOG_DUMP_BODY) \
		TEST_RESOURCE_AGGREGATE_LEVEL=$(TEST_RESOURCE_AGGREGATE_LEVEL) TEST_RESOURCE_AGGREGATE_LOG_DUMP_BODY=$(TEST_RESOURCE_AGGREGATE_LOG_DUMP_BODY) \
		TEST_GRPC_GATEWAY_LOG_LEVEL=$(TEST_GRPC_GATEWAY_LOG_LEVEL) TEST_GRPC_GATEWAY_LOG_DUMP_BODY=$(TEST_GRPC_GATEWAY_LOG_DUMP_BODY) \
		TEST_IDENTITY_STORE_LOG_LEVEL=$(TEST_IDENTITY_STORE_LOG_LEVEL) TEST_IDENTITY_STORE_LOG_DUMP_BODY=$(TEST_IDENTITY_STORE_LOG_DUMP_BODY) \
		TEST_SNIPPET_SERVICE_LOG_LEVEL=$(TEST_SNIPPET_SERVICE_LOG_LEVEL) TEST_SNIPPET_SERVICE_LOG_DUMP_BODY=$(TEST_SNIPPET_SERVICE_LOG_DUMP_BODY) \
		TEST_LEAD_RESOURCE_TYPE_FILTER=$(TEST_LEAD_RESOURCE_TYPE_FILTER) TEST_LEAD_RESOURCE_TYPE_REGEX_FILTER='$(TEST_LEAD_RESOURCE_TYPE_REGEX_FILTER)' TEST_LEAD_RESOURCE_TYPE_USE_UUID=$(TEST_LEAD_RESOURCE_TYPE_USE_UUID) \
		TEST_DPS_UDP_ENABLED=$(TEST_DPS_UDP_ENABLED) TEST_DATABASE=$(TEST_DATABASE))

.PHONY: $(test-targets)

SUBDIRS := bundle certificate-authority cloud2cloud-connector cloud2cloud-gateway coap-gateway device-provisioning-service grpc-gateway resource-aggregate resource-directory http-gateway identity-store snippet-service m2m-oauth-server test/oauth-server tools/cert-tool

build: $(SUBDIRS)

clean: simulators/clean nats/clean scylla/clean mongo/clean mongo-no-replicas/clean privateKeys/clean http-gateway-www/clean
	sudo rm -rf ./.tmp/home || true
	sudo rm -rf ./.tmp/coverage || true
	sudo rm -rf ./.tmp/report || true

proto/generate: $(SUBDIRS)
	protoc -I=. -I=$(GOPATH)/src --go_out=$(GOPATH)/src $(WORKING_DIRECTORY)/pkg/net/grpc/stub.proto
	protoc -I=. -I=$(GOPATH)/src --go-grpc_out=$(GOPATH)/src $(WORKING_DIRECTORY)/pkg/net/grpc/stub.proto
	mv $(WORKING_DIRECTORY)/pkg/net/grpc/stub.pb.go $(WORKING_DIRECTORY)/pkg/net/grpc/stub.pb_test.go
	mv $(WORKING_DIRECTORY)/pkg/net/grpc/stub_grpc.pb.go $(WORKING_DIRECTORY)/pkg/net/grpc/stub_grpc.pb_test.go

push: $(SUBDIRS)

.PHONY: $(SUBDIRS) push proto/generate clean build

$(SUBDIRS):
	$(MAKE) -C $@ $(MAKECMDGOALS) LATEST_TAG=$(BUILD_TAG)
