FROM golang:1.18.1-alpine AS build
RUN apk add --no-cache curl git build-base
WORKDIR $GOPATH/src/github.com/plgd-dev/hub
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN ( cd /usr/local/go && patch -p1 < $GOPATH/src/github.com/plgd-dev/hub/tools/docker/patches/shrink_tls_conn.patch )
ARG root_directory=$GOPATH/src/github.com/plgd-dev/hub

#coap-gateway
ARG service=coap-gateway
WORKDIR $root_directory/$service
RUN go build -ldflags "-linkmode external -extldflags -static" -o /go/bin/$service ./cmd/service

#grpc-gateway
ARG service=grpc-gateway
WORKDIR $root_directory/$service
RUN go build -ldflags "-linkmode external -extldflags -static" -o /go/bin/$service ./cmd/service

#http-gateway
ARG service=http-gateway
WORKDIR $root_directory/$service
RUN go build -ldflags "-linkmode external -extldflags -static" -o /go/bin/$service ./cmd/service

#resource-directory
ARG service=resource-directory
WORKDIR $root_directory/$service
RUN go build -ldflags "-linkmode external -extldflags -static" -o /go/bin/$service ./cmd/service

#resource-aggregate
ARG service=resource-aggregate
WORKDIR $root_directory/$service
RUN go build -ldflags "-linkmode external -extldflags -static" -o /go/bin/$service ./cmd/service

#identity-store
ARG service=identity-store
WORKDIR $root_directory/$service
RUN go build -ldflags "-linkmode external -extldflags -static" -o /go/bin/$service ./cmd/service

#certificate-authority
ARG service=certificate-authority
WORKDIR $root_directory/$service
RUN go build -ldflags "-linkmode external -extldflags -static" -o /go/bin/$service ./cmd/service

#oauth-server
ARG service=oauth-server
WORKDIR $root_directory/test/$service
RUN go build -ldflags "-linkmode external -extldflags -static" -o /go/bin/$service ./cmd/service

#cloud2cloud-gateway
ARG service=cloud2cloud-gateway
WORKDIR $root_directory/$service
RUN go build -ldflags "-linkmode external -extldflags -static" -o /go/bin/$service ./cmd/service

#cloud2cloud-connector
ARG service=cloud2cloud-connector
WORKDIR $root_directory/$service
RUN go build -ldflags "-linkmode external -extldflags -static" -o /go/bin/$service ./cmd/service


#certificate-generator
ARG service=kit
WORKDIR /
RUN cd $GOPATH/pkg/mod/github.com/plgd-dev/kit/v2* && go build -ldflags "-linkmode external -extldflags -static" -o /go/bin/certificate-generator ./cmd/certificate-generator

#nats
WORKDIR $root_directory
RUN apkArch="$(apk --print-arch)"; \
    case "$apkArch" in \
        armhf) ARCH='arm' ;; \
        aarch64) ARCH='arm64' ;; \
        x86) ARCH='386' ;; \
        x86_64) ARCH='amd64' ;; \
        *) echo >&2 "error: unsupported architecture: $apkArch"; exit 1 ;; \
    esac; \
    curl -L https://github.com/nats-io/nats-server/releases/download/v2.3.1/nats-server-v2.3.1-linux-${ARCH}.zip -o ./nats-server.zip ; \
    curl -L https://github.com/nats-io/natscli/releases/download/0.0.24/nats-0.0.24-linux-${ARCH}.zip -o ./nats.zip
RUN mkdir -p ./nats-server
RUN unzip ./nats-server.zip -d ./nats-server
RUN cp ./nats-server/*/nats-server /go/bin/nats-server

RUN mkdir -p ./nats
RUN unzip ./nats.zip -d ./nats
RUN cp ./nats/*/nats /go/bin/nats

FROM node:12 AS build-web
COPY --from=build /go/src/github.com/plgd-dev/hub/http-gateway/web /web
RUN cd /web && npm install && npm run build

FROM ubuntu:20.04 as service
RUN apt update
RUN apt install -y wget gnupg iproute2 systemctl openssl nginx ca-certificates netcat
RUN wget -qO - https://www.mongodb.org/static/pgp/server-4.4.asc | apt-key add -
RUN ARCH="$(dpkg --print-architecture)" ; \
    wget https://github.com/mikefarah/yq/releases/download/v4.6.3/yq_linux_${ARCH} -O /usr/bin/yq && chmod +x /usr/bin/yq ; \
    echo "deb [ arch=${ARCH} ] https://repo.mongodb.org/apt/ubuntu focal/mongodb-org/4.4 multiverse" | tee /etc/apt/sources.list.d/mongodb-org-4.4.list
RUN apt update
RUN apt-get install -y mongodb-org-server mongodb-org
COPY --from=build /go/bin/coap-gateway /usr/local/bin/coap-gateway
COPY --from=build /go/src/github.com/plgd-dev/hub/coap-gateway/config.yaml /configs/coap-gateway.yaml
COPY --from=build /go/bin/grpc-gateway /usr/local/bin/grpc-gateway
COPY --from=build /go/src/github.com/plgd-dev/hub/grpc-gateway/config.yaml /configs/grpc-gateway.yaml
COPY --from=build /go/bin/http-gateway /usr/local/bin/http-gateway
COPY --from=build /go/src/github.com/plgd-dev/hub/http-gateway/config.yaml /configs/http-gateway.yaml
COPY --from=build /go/bin/resource-directory /usr/local/bin/resource-directory
COPY --from=build /go/src/github.com/plgd-dev/hub/resource-directory/config.yaml /configs/resource-directory.yaml
COPY --from=build /go/bin/resource-aggregate /usr/local/bin/resource-aggregate
COPY --from=build /go/src/github.com/plgd-dev/hub/resource-aggregate/config.yaml /configs/resource-aggregate.yaml
COPY --from=build /go/bin/identity-store /usr/local/bin/identity-store
COPY --from=build /go/src/github.com/plgd-dev/hub/identity-store/config.yaml /configs/identity-store.yaml
COPY --from=build /go/bin/certificate-authority /usr/local/bin/certificate-authority
COPY --from=build /go/src/github.com/plgd-dev/hub/certificate-authority/config.yaml /configs/certificate-authority.yaml
COPY --from=build /go/bin/certificate-generator /usr/local/bin/certificate-generator
COPY --from=build /go/bin/nats-server /usr/local/bin/nats-server
COPY --from=build /go/bin/nats /usr/local/bin/nats
COPY --from=build /go/src/github.com/plgd-dev/hub/bundle/jetstream.json /configs/jetstream.json
COPY --from=build /go/bin/oauth-server /usr/local/bin/oauth-server
COPY --from=build /go/src/github.com/plgd-dev/hub/test/oauth-server/config.yaml /configs/oauth-server.yaml
COPY --from=build-web /web/build /usr/local/var/www
COPY --from=build /go/bin/cloud2cloud-gateway /usr/local/bin/cloud2cloud-gateway
COPY --from=build /go/src/github.com/plgd-dev/hub/cloud2cloud-gateway/config.yaml /configs/cloud2cloud-gateway.yaml
COPY --from=build /go/bin/cloud2cloud-connector /usr/local/bin/cloud2cloud-connector
COPY --from=build /go/src/github.com/plgd-dev/hub/cloud2cloud-connector/config.yaml /configs/cloud2cloud-connector.yaml
COPY --from=build /go/src/github.com/plgd-dev/hub/bundle/run.sh /usr/local/bin/run.sh
COPY --from=build /go/src/github.com/plgd-dev/hub/bundle/nginx /nginx

# global
ENV FQDN="localhost"
ENV LOG_DEBUG=false
ENV JETSTREAM=false

# global - open telemetry collector client
ENV OPEN_TELEMETRY_EXPORTER_ENABLED=false
ENV OPEN_TELEMETRY_EXPORTER_ADDRESS="localhost:4317"
ENV OPEN_TELEMETRY_EXPORTER_CERT_FILE="/certs/otel/cert.crt"
ENV OPEN_TELEMETRY_EXPORTER_KEY_FILE="/certs/otel/cert.key"
ENV OPEN_TELEMETRY_EXPORTER_CA_POOL="/certs/otel/rootca.crt"

# coap-gateway
ENV COAP_GATEWAY_UNSECURE_PORT=5683
ENV COAP_GATEWAY_UNSECURE_ADDRESS="0.0.0.0:$COAP_GATEWAY_UNSECURE_PORT"
ENV COAP_GATEWAY_UNSECURE_ENABLED=true
ENV COAP_GATEWAY_PORT=5684
ENV COAP_GATEWAY_ADDRESS="0.0.0.0:$COAP_GATEWAY_PORT"
ENV COAP_GATEWAY_HUB_ID="00000000-0000-0000-0000-000000000001"
ENV COAP_GATEWAY_LOG_MESSAGES=true

# ports
ENV NGINX_PORT=443
ENV CERTIFICATE_AUTHORITY_PORT=9087
ENV MOCK_OAUTH_SERVER_PORT=9088
ENV RESOURCE_AGGREGATE_PORT=9083
ENV RESOURCE_DIRECTORY_PORT=9082
ENV IDENTITY_STORE_PORT=9081
ENV GRPC_GATEWAY_PORT=9084
ENV HTTP_GATEWAY_PORT=9086
ENV CLOUD2CLOUD_GATEWAY_PORT=9085
ENV CLOUD2CLOUD_CONNECTOR_PORT=9089
ENV MONGO_PORT=10000
ENV NATS_PORT=10001

# OAuth
ENV DEVICE_PROVIDER=plgd
ENV DEVICE_OAUTH_SCOPES="offline_access"
ENV OWNER_CLAIM="sub"
ENV MOCK_OAUTH_SERVER_ACCESS_TOKEN_LIFETIME="0s"

ENTRYPOINT ["/usr/local/bin/run.sh"]
