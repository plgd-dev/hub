SHELL = /bin/bash

SUBDIRS := resource-aggregate

ocf-cloud-build:
	docker build \
		--network=host \
		--tag ocf-cloud-build \
		.

make-ca:
	docker pull smallstep/step-ca
	if [ "${TRAVIS_OS_NAME}" == "linux" ]; then \
		sudo sh -c 'echo net.ipv4.ip_unprivileged_port_start=0 > /etc/sysctl.d/50-unprivileged-ports.conf'; \
		sudo sysctl --system; \
	fi
	mkdir -p ./test/step-ca/data/secrets
	echo "password" > ./test/step-ca/data/secrets/password
	docker run \
		-it \
		-v "$(shell pwd)"/test/step-ca/data:/home/step --user $(shell id -u):$(shell id -g) \
		smallstep/step-ca \
		/bin/bash -c "step ca init -dns localhost -address=:10443 -provisioner=test@localhost -name test -password-file ./secrets/password && step ca provisioner add acme --type ACME"
	docker run \
		-d \
		--network=host \
		--name=step-ca-test \
		-v /etc/nsswitch.conf:/etc/nsswitch.conf \
		-v "$(shell pwd)"/test/step-ca/data:/home/step --user $(shell id -u):$(shell id -g) \
		smallstep/step-ca

make-nats:
	sleep 1
	docker exec -it step-ca-test /bin/bash -c "mkdir -p certs/nats && step ca certificate localhost certs/nats/nats.crt certs/nats/nats.key --provisioner acme"
	docker run \
	    -d \
		--network=host \
		--name=nats \
		-v $(shell pwd)/test/step-ca/data/certs:/certs \
		nats --tls --tlsverify --tlscert=/certs/nats/nats.crt --tlskey=/certs/nats/nats.key --tlscacert=/certs/root_ca.crt

make-mongo:
	sleep 1
	mkdir -p $(shell pwd)/test/mongo
	docker exec -it step-ca-test /bin/bash -c "mkdir -p certs/mongo && step ca certificate localhost certs/mongo/mongo.crt certs/mongo/mongo.key --provisioner acme && cat certs/mongo/mongo.crt >> certs/mongo/mongo.key"
	docker run \
	    -d \
		--network=host \
		--name=mongo \
		-v $(shell pwd)/test/mongo:/data/db \
		-v $(shell pwd)/test/step-ca/data/certs:/certs --user $(shell id -u):$(shell id -g) \
		mongo --tlsMode requireTLS --tlsCAFile /certs/root_ca.crt --tlsCertificateKeyFile certs/mongo/mongo.key

env: clean make-ca make-nats make-mongo ocf-cloud-build

test: env
	docker run \
		--network=host \
		-v $(shell pwd)/test/step-ca/data/certs/root_ca.crt:/root_ca.crt \
		-e DIAL_ACME_CA_POOL=/root_ca.crt \
		-e DIAL_ACME_DOMAINS="localhost" \
		-e DIAL_ACME_DIRECTORY_URL="https://localhost:10443/acme/acme/directory" \
		-e LISTEN_ACME_CA_POOL=/root_ca.crt \
		-e LISTEN_ACME_DOMAINS="localhost" \
		-e LISTEN_ACME_DIRECTORY_URL="https://localhost:10443/acme/acme/directory" \
		--mount type=bind,source="$(shell pwd)",target=/shared \
		ocf-cloud-build \
		go test -p 1 -v ./... -covermode=atomic -coverprofile=/shared/coverage.txt

build: ocf-cloud-build $(SUBDIRS)
	$(MAKE) -C $@ $(MAKECMDGOALS)

clean: $(SUBDIRS)
	docker rm -f step-ca-test || true
	docker rm -f mongo || true
	docker rm -f nats || true
	rm -rf ./test/step-ca || true
	rm -rf ./test/mongo || true

proto/generate: $(SUBDIRS)
push: $(SUBDIRS)

$(SUBDIRS):
	$(MAKE) -C $@ $(MAKECMDGOALS)

.PHONY: $(SUBDIRS) push proto/generate clean build test env make-mongo make-nats make-ca ocf-cloud-build 