SHELL = /bin/bash
SIMULATOR_NAME_SUFFIX ?= $(shell hostname)

SUBDIRS := resource-aggregate authorization resource-directory cloud2cloud-connector cloud2cloud-gateway coap-gateway grpc-gateway certificate-authority portal-webapi bundle http-gateway
.PHONY: $(SUBDIRS) push proto/generate clean build test env make-mongo make-nats make-ca cloud-build

default: build

cloud-build:
	docker build \
		--network=host \
		--tag cloud-build \
		.

make-ca:
	docker pull ocfcloud/step-ca:vnext
	if [ "${TRAVIS_OS_NAME}" == "linux" ]; then \
		sudo sh -c 'echo net.ipv4.ip_unprivileged_port_start=0 > /etc/sysctl.d/50-unprivileged-ports.conf'; \
		sudo sysctl --system; \
	fi
	mkdir -p ./.tmp/step-ca/data/secrets
	echo "password" > ./.tmp/step-ca/data/secrets/password
	docker run \
		-it \
		-v "$(shell pwd)"/.tmp/step-ca/data:/home/step --user $(shell id -u):$(shell id -g) \
		ocfcloud/step-ca:vnext \
		/bin/bash -c "step ca init -dns localhost -address=:10443 -provisioner=test@localhost -name test -password-file ./secrets/password && step ca provisioner add acme --type ACME && step ca provisioner add ocf.gw --type ACME"
	docker run \
		-d \
		--network=host \
		--name=step-ca-test \
		-v /etc/nsswitch.conf:/etc/nsswitch.conf \
		-v "$(shell pwd)"/.tmp/step-ca/data:/home/step --user $(shell id -u):$(shell id -g) \
		ocfcloud/step-ca:vnext

make-nats:
	sleep 1
	docker exec -it step-ca-test /bin/bash -c "mkdir -p certs/nats && step ca certificate localhost certs/nats/nats.crt certs/nats/nats.key --provisioner acme"
	docker run \
	    -d \
		--network=host \
		--name=nats \
		-v $(shell pwd)/.tmp/step-ca/data/certs:/certs \
		nats --tls --tlsverify --tlscert=/certs/nats/nats.crt --tlskey=/certs/nats/nats.key --tlscacert=/certs/root_ca.crt

make-mongo:
	sleep 1
	mkdir -p $(shell pwd)/.tmp/mongo
	docker exec -it step-ca-test /bin/bash -c "mkdir -p certs/mongo && step ca certificate localhost certs/mongo/mongo.crt certs/mongo/mongo.key --provisioner acme && cat certs/mongo/mongo.crt >> certs/mongo/mongo.key"
	docker run \
	    -d \
		--network=host \
		--name=mongo \
		-v $(shell pwd)/.tmp/mongo:/data/db \
		-v $(shell pwd)/.tmp/step-ca/data/certs:/certs --user $(shell id -u):$(shell id -g) \
		mongo --tlsMode requireTLS --tlsCAFile /certs/root_ca.crt --tlsCertificateKeyFile certs/mongo/mongo.key

env: clean make-ca make-nats make-mongo
	docker build ./device-simulator --network=host -t device-simulator --target service
	docker run -d --name=devsim --network=host -t device-simulator devsim-$(SIMULATOR_NAME_SUFFIX)

test: env cloud-build
	docker run \
		--network=host \
		-v $(shell pwd)/.tmp/step-ca/data/certs/root_ca.crt:/root_ca.crt \
		-e DIAL_ACME_CA_POOL=/root_ca.crt \
		-e DIAL_ACME_DOMAINS="localhost" \
		-e DIAL_ACME_DIRECTORY_URL="https://localhost:10443/acme/acme/directory" \
		-e LISTEN_ACME_CA_POOL=/root_ca.crt \
		-e LISTEN_ACME_DOMAINS="localhost" \
		-e LISTEN_ACME_DEVICE_ID="adebc667-1f2b-41e3-bf5c-6d6eabc68cc6" \
		-e LISTEN_ACME_DIRECTORY_URL="https://localhost:10443/acme/acme/directory" \
		-e TEST_COAP_GW_OVERWRITE_LISTEN_ACME_DIRECTORY_URL="https://localhost:10443/acme/ocf.gw/directory" \
		--mount type=bind,source="$(shell pwd)",target=/shared \
		cloud-build \
		go test -p 1 -v ./... -covermode=atomic -coverprofile=/shared/coverage.txt

build: cloud-build $(SUBDIRS)

clean:
	docker rm -f step-ca-test || true
	docker rm -f mongo || true
	docker rm -f nats || true
	docker rm -f devsim || true
	rm -rf ./.tmp/step-ca || true
	rm -rf ./.tmp/mongo || true

proto/generate: $(SUBDIRS)
push: $(SUBDIRS)

$(SUBDIRS):
	$(MAKE) -C $@ $(MAKECMDGOALS)
