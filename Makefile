SHELL = /bin/bash
SIMULATOR_NAME_SUFFIX ?= $(shell hostname)

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
		/bin/bash -c "cert-tool --cmd.generateRootCA --outCert=/certs/root_ca.crt --outKey=/certs/root_ca.key --cert.subject.cn=RootCA && cert-tool --cmd.generateCertificate --outCert=/certs/http.crt --outKey=/certs/http.key --cert.subject.cn=localhost --cert.san.domain=localhost --signerCert=/certs/root_ca.crt --signerKey=/certs/root_ca.key && cert-tool --cmd.generateIdentityCertificate=adebc667-1f2b-41e3-bf5c-6d6eabc68cc6 --outCert=/certs/coap.crt --outKey=/certs/coap.key --cert.san.domain=localhost --signerCert=/certs/root_ca.crt --signerKey=/certs/root_ca.key"
	cat $(shell pwd)/.tmp/certs/http.crt > $(shell pwd)/.tmp/certs/mongo.key
	cat $(shell pwd)/.tmp/certs/http.key >> $(shell pwd)/.tmp/certs/mongo.key

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

env: clean certificates nats mongo
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
		-e ACME_DB_DIR=/home/certificate-authority \
		cloud-test \
		go test -race -p 1 -v ./... -covermode=atomic -coverprofile=/home/coverage.txt
	cp $(shell pwd)/.tmp/home/coverage.txt $(shell pwd)/coverage.txt

build: cloud-build $(SUBDIRS)

clean:
	docker rm -f mongo || true
	docker rm -f nats || true
	docker rm -f nats-cloud-connector || true
	docker rm -f devsim || true
	rm -rf ./.tmp/certs || true
	rm -rf ./.tmp/mongo || true
	rm -rf ./.tmp/home || true

proto/generate: $(SUBDIRS)
push: cloud-build $(SUBDIRS)

$(SUBDIRS):
	$(MAKE) -C $@ $(MAKECMDGOALS)
