# syntax=docker/dockerfile:1
FROM golang:1.22.3-alpine AS build
RUN apk add --no-cache build-base curl git
WORKDIR $GOPATH/src/github.com/plgd-dev/hub
COPY go.mod go.sum ./
RUN go mod download
COPY . .
WORKDIR /usr/local/go
RUN ( patch -p1 < "$GOPATH/src/github.com/plgd-dev/hub/tools/docker/patches/shrink_tls_conn.patch" )
WORKDIR $GOPATH/src/github.com/plgd-dev/hub

ARG root_directory=$GOPATH/src/github.com/plgd-dev/hub

ARG COMMIT_DATE
ARG SHORT_COMMIT
ARG DATE
ARG VERSION
ARG RELEASE_URL

#coap-gateway
ARG service=coap-gateway
WORKDIR $root_directory/$service
RUN go build \
    -ldflags "-linkmode external -extldflags -static \
    -X github.com/plgd-dev/hub/v2/pkg/build.CommitDate=$COMMIT_DATE \
    -X github.com/plgd-dev/hub/v2/pkg/build.CommitHash=$SHORT_COMMIT \
    -X github.com/plgd-dev/hub/v2/pkg/build.BuildDate=$DATE \
    -X github.com/plgd-dev/hub/v2/pkg/build.Version=$VERSION -X \
    github.com/plgd-dev/hub/v2/pkg/build.ReleaseURL=$RELEASE_URL" \
    -o "/go/bin/$service" \
    ./cmd/service

#grpc-gateway
ARG service=grpc-gateway
WORKDIR $root_directory/$service
RUN go build \
    -ldflags "-linkmode external -extldflags -static \
    -X github.com/plgd-dev/hub/v2/pkg/build.CommitDate=$COMMIT_DATE \
    -X github.com/plgd-dev/hub/v2/pkg/build.CommitHash=$SHORT_COMMIT \
    -X github.com/plgd-dev/hub/v2/pkg/build.BuildDate=$DATE \
    -X github.com/plgd-dev/hub/v2/pkg/build.Version=$VERSION \
    -X github.com/plgd-dev/hub/v2/pkg/build.ReleaseURL=$RELEASE_URL" \
    -o "/go/bin/$service" \
    ./cmd/service

#http-gateway
ARG service=http-gateway
WORKDIR $root_directory/$service
RUN go build \
    -ldflags "-linkmode external -extldflags -static \
    -X github.com/plgd-dev/hub/v2/pkg/build.CommitDate=$COMMIT_DATE \
    -X github.com/plgd-dev/hub/v2/pkg/build.CommitHash=$SHORT_COMMIT \
    -X github.com/plgd-dev/hub/v2/pkg/build.BuildDate=$DATE \
    -X github.com/plgd-dev/hub/v2/pkg/build.Version=$VERSION \
    -X github.com/plgd-dev/hub/v2/pkg/build.ReleaseURL=$RELEASE_URL" \
    -o "/go/bin/$service" \
    ./cmd/service

#resource-directory
ARG service=resource-directory
WORKDIR $root_directory/$service
RUN go build \
    -ldflags "-linkmode external -extldflags -static \
    -X github.com/plgd-dev/hub/v2/pkg/build.CommitDate=$COMMIT_DATE \
    -X github.com/plgd-dev/hub/v2/pkg/build.CommitHash=$SHORT_COMMIT \
    -X github.com/plgd-dev/hub/v2/pkg/build.BuildDate=$DATE \
    -X github.com/plgd-dev/hub/v2/pkg/build.Version=$VERSION \
    -X github.com/plgd-dev/hub/v2/pkg/build.ReleaseURL=$RELEASE_URL" \
    -o "/go/bin/$service" \
    ./cmd/service

#resource-aggregate
ARG service=resource-aggregate
WORKDIR $root_directory/$service
RUN go build \
    -ldflags "-linkmode external -extldflags -static \
    -X github.com/plgd-dev/hub/v2/pkg/build.CommitDate=$COMMIT_DATE \
    -X github.com/plgd-dev/hub/v2/pkg/build.CommitHash=$SHORT_COMMIT \
    -X github.com/plgd-dev/hub/v2/pkg/build.BuildDate=$DATE \
    -X github.com/plgd-dev/hub/v2/pkg/build.Version=$VERSION \
    -X github.com/plgd-dev/hub/v2/pkg/build.ReleaseURL=$RELEASE_URL" \
    -o "/go/bin/$service" \
    ./cmd/service

#identity-store
ARG service=identity-store
WORKDIR $root_directory/$service
RUN go build \
    -ldflags "-linkmode external -extldflags -static \
    -X github.com/plgd-dev/hub/v2/pkg/build.CommitDate=$COMMIT_DATE \
    -X github.com/plgd-dev/hub/v2/pkg/build.CommitHash=$SHORT_COMMIT \
    -X github.com/plgd-dev/hub/v2/pkg/build.BuildDate=$DATE \
    -X github.com/plgd-dev/hub/v2/pkg/build.Version=$VERSION \
    -X github.com/plgd-dev/hub/v2/pkg/build.ReleaseURL=$RELEASE_URL" \
    -o "/go/bin/$service" \
    ./cmd/service

#certificate-authority
ARG service=certificate-authority
WORKDIR $root_directory/$service
RUN go build \
    -ldflags "-linkmode external -extldflags -static \
    -X github.com/plgd-dev/hub/v2/pkg/build.CommitDate=$COMMIT_DATE \
    -X github.com/plgd-dev/hub/v2/pkg/build.CommitHash=$SHORT_COMMIT \
    -X github.com/plgd-dev/hub/v2/pkg/build.BuildDate=$DATE \
    -X github.com/plgd-dev/hub/v2/pkg/build.Version=$VERSION \
    -X github.com/plgd-dev/hub/v2/pkg/build.ReleaseURL=$RELEASE_URL" \
    -o "/go/bin/$service" \
    ./cmd/service

#oauth-server
ARG service=oauth-server
WORKDIR $root_directory/test/$service
RUN go build \
    -ldflags "-linkmode external -extldflags -static \
    -X github.com/plgd-dev/hub/v2/pkg/build.CommitDate=$COMMIT_DATE \
    -X github.com/plgd-dev/hub/v2/pkg/build.CommitHash=$SHORT_COMMIT \
    -X github.com/plgd-dev/hub/v2/pkg/build.BuildDate=$DATE \
    -X github.com/plgd-dev/hub/v2/pkg/build.Version=$VERSION \
    -X github.com/plgd-dev/hub/v2/pkg/build.ReleaseURL=$RELEASE_URL" \
    -o "/go/bin/$service" \
    ./cmd/service

#oauth-server
ARG service=m2m-oauth-server
WORKDIR $root_directory/$service
RUN go build \
    -ldflags "-linkmode external -extldflags -static \
    -X github.com/plgd-dev/hub/v2/pkg/build.CommitDate=$COMMIT_DATE \
    -X github.com/plgd-dev/hub/v2/pkg/build.CommitHash=$SHORT_COMMIT \
    -X github.com/plgd-dev/hub/v2/pkg/build.BuildDate=$DATE \
    -X github.com/plgd-dev/hub/v2/pkg/build.Version=$VERSION \
    -X github.com/plgd-dev/hub/v2/pkg/build.ReleaseURL=$RELEASE_URL" \
    -o "/go/bin/$service" \
    ./cmd/service

#cloud2cloud-gateway
ARG service=cloud2cloud-gateway
WORKDIR $root_directory/$service
RUN go build \
    -ldflags "-linkmode external -extldflags -static \
    -X github.com/plgd-dev/hub/v2/pkg/build.CommitDate=$COMMIT_DATE \
    -X github.com/plgd-dev/hub/v2/pkg/build.CommitHash=$SHORT_COMMIT \
    -X github.com/plgd-dev/hub/v2/pkg/build.BuildDate=$DATE \
    -X github.com/plgd-dev/hub/v2/pkg/build.Version=$VERSION \
    -X github.com/plgd-dev/hub/v2/pkg/build.ReleaseURL=$RELEASE_URL" \
    -o "/go/bin/$service" \
    ./cmd/service

#cloud2cloud-connector
ARG service=cloud2cloud-connector
WORKDIR $root_directory/$service
RUN go build \
    -ldflags "-linkmode external -extldflags -static \
    -X github.com/plgd-dev/hub/v2/pkg/build.CommitDate=$COMMIT_DATE \
    -X github.com/plgd-dev/hub/v2/pkg/build.CommitHash=$SHORT_COMMIT \
    -X github.com/plgd-dev/hub/v2/pkg/build.BuildDate=$DATE \
    -X github.com/plgd-dev/hub/v2/pkg/build.Version=$VERSION \
    -X github.com/plgd-dev/hub/v2/pkg/build.ReleaseURL=$RELEASE_URL" \
    -o "/go/bin/$service" \
    ./cmd/service

#cert-tool
ARG tool=cert-tool
WORKDIR $root_directory/tools/$tool
RUN go build \
    -ldflags "-linkmode external -extldflags -static \
    -X github.com/plgd-dev/hub/v2/pkg/build.CommitDate=$COMMIT_DATE \
    -X github.com/plgd-dev/hub/v2/pkg/build.CommitHash=$SHORT_COMMIT \
    -X github.com/plgd-dev/hub/v2/pkg/build.BuildDate=$DATE \
    -X github.com/plgd-dev/hub/v2/pkg/build.Version=$VERSION \
    -X github.com/plgd-dev/hub/v2/pkg/build.ReleaseURL=$RELEASE_URL" \
    -o "/go/bin/$tool" \
    ./

#snippet-service
ARG service=snippet-service
WORKDIR $root_directory/$service
RUN go build \
    -ldflags "-linkmode external -extldflags -static \
    -X github.com/plgd-dev/hub/v2/pkg/build.CommitDate=$COMMIT_DATE \
    -X github.com/plgd-dev/hub/v2/pkg/build.CommitHash=$SHORT_COMMIT \
    -X github.com/plgd-dev/hub/v2/pkg/build.BuildDate=$DATE \
    -X github.com/plgd-dev/hub/v2/pkg/build.Version=$VERSION \
    -X github.com/plgd-dev/hub/v2/pkg/build.ReleaseURL=$RELEASE_URL" \
    -o "/go/bin/$service" \
    ./cmd/service

#grpc-reflection
ARG service=grpc-reflection
WORKDIR $root_directory/tools/$service
RUN go build \
    -ldflags "-linkmode external -extldflags -static \
    -X github.com/plgd-dev/hub/v2/pkg/build.CommitDate=$COMMIT_DATE \
    -X github.com/plgd-dev/hub/v2/pkg/build.CommitHash=$SHORT_COMMIT \
    -X github.com/plgd-dev/hub/v2/pkg/build.BuildDate=$DATE \
    -X github.com/plgd-dev/hub/v2/pkg/build.Version=$VERSION \
    -X github.com/plgd-dev/hub/v2/pkg/build.ReleaseURL=$RELEASE_URL" \
    -o "/go/bin/$service" \
    ./cmd/service

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
    curl -L https://github.com/nats-io/natscli/releases/download/0.0.24/nats-0.0.24-linux-${ARCH}.zip -o ./nats.zip \
    && mkdir -p ./nats-server \
    && unzip ./nats-server.zip -d ./nats-server \
    && cp ./nats-server/*/nats-server /go/bin/nats-server \
    && mkdir -p ./nats \
    && unzip ./nats.zip -d ./nats \
    && cp ./nats/*/nats /go/bin/nats

FROM ubuntu:22.04 AS service
# iproute2 -> ip utility in run.sh
# netcat -> nc utility in run.sh
# nginx -> nginx server in run.sh
# openssl -> openssl utility in run.sh
# yq utility in run.sh
RUN apt update \
    && apt-get install -y --no-install-recommends ca-certificates coreutils curl gnupg iproute2 netcat nginx openssl sudo wget \
    && wget https://github.com/mikefarah/yq/releases/download/v4.44.2/yq_linux_$(dpkg --print-architecture) -O /usr/bin/yq && chmod +x /usr/bin/yq \
    && wget -qO - https://pgp.mongodb.com/server-6.0.asc | gpg --dearmor -o /etc/apt/trusted.gpg.d/mongodb-6.0.gpg \
    && echo "deb [ arch=$(dpkg --print-architecture) ] https://repo.mongodb.org/apt/ubuntu jammy/mongodb-org/6.0 multiverse" | tee /etc/apt/sources.list.d/mongodb-org-6.0.list \
    && apt update \
    && DEBIAN_FRONTEND=noninteractive apt-get install -y --no-install-recommends mongodb-org mongodb-org-server \
    && apt-get clean

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
COPY --from=build /go/bin/cert-tool /usr/local/bin/cert-tool
COPY --from=build /go/bin/nats-server /usr/local/bin/nats-server
COPY --from=build /go/bin/nats /usr/local/bin/nats
COPY --from=build /go/src/github.com/plgd-dev/hub/bundle/jetstream.json /configs/jetstream.json
COPY --from=build /go/bin/oauth-server /usr/local/bin/oauth-server
COPY --from=build /go/src/github.com/plgd-dev/hub/test/oauth-server/config.yaml /configs/oauth-server.yaml
COPY --from=build /go/bin/m2m-oauth-server /usr/local/bin/m2m-oauth-server
COPY --from=build /go/src/github.com/plgd-dev/hub/m2m-oauth-server/config.yaml /configs/m2m-oauth-server.yaml
COPY http-gateway/web/build /usr/local/var/www
COPY --from=build /go/bin/cloud2cloud-gateway /usr/local/bin/cloud2cloud-gateway
COPY --from=build /go/src/github.com/plgd-dev/hub/cloud2cloud-gateway/config.yaml /configs/cloud2cloud-gateway.yaml
COPY --from=build /go/bin/cloud2cloud-connector /usr/local/bin/cloud2cloud-connector
COPY --from=build /go/src/github.com/plgd-dev/hub/cloud2cloud-connector/config.yaml /configs/cloud2cloud-connector.yaml
COPY --from=build /go/src/github.com/plgd-dev/hub/bundle/run.sh /usr/local/bin/run.sh
COPY --from=build /go/src/github.com/plgd-dev/hub/bundle/nginx /nginx
COPY --from=build /go/bin/snippet-service /usr/local/bin/snippet-service
COPY --from=build /go/src/github.com/plgd-dev/hub/snippet-service/config.yaml /configs/snippet-service.yaml
COPY --from=build /go/bin/grpc-reflection /usr/local/bin/grpc-reflection
COPY --from=build /go/src/github.com/plgd-dev/hub/tools/grpc-reflection/config.yaml /configs/grpc-reflection.yaml

# install scylla
RUN curl -sSf get.scylladb.com/server | sudo bash -s -- --scylla-version 5.4
COPY bundle/scylla /scylla

# global
ENV FQDN="localhost"
ENV LOG_DEBUG=false
# supported values: mongo, scylla
ENV DATABASE=mongo
ENV OVERRIDE_FILES=false
ENV HUB_ID="afd434f8-bf61-4729-a258-3c1a67fb0287"

# global - NATS
ENV LEAD_RESOURCE_TYPE_ENABLED=false
ENV LEAD_RESOURCE_TYPE_REGEX_FILTER=""
ENV LEAD_RESOURCE_TYPE_FILTER=""
ENV LEAD_RESOURCE_TYPE_USE_UUID=false
ENV JETSTREAM=false

# global - open telemetry collector client
ENV OPEN_TELEMETRY_EXPORTER_ENABLED=false
ENV OPEN_TELEMETRY_EXPORTER_ADDRESS="localhost:4317"
ENV OPEN_TELEMETRY_EXPORTER_CERT_FILE="/certs/otel/cert.crt"
ENV OPEN_TELEMETRY_EXPORTER_KEY_FILE="/certs/otel/cert.key"
ENV OPEN_TELEMETRY_EXPORTER_CA_POOL="/certs/otel/rootca.crt"

# global - certificates generated by cert-tool
ENV CERT_TOOL_SIGN_ALG=ECDSA-SHA256
ENV CERT_TOOL_ELLIPTIC_CURVE=P256

# coap-gateway
ENV COAP_GATEWAY_UNSECURE_PORT=5683
ENV COAP_GATEWAY_UNSECURE_ADDRESS="0.0.0.0:$COAP_GATEWAY_UNSECURE_PORT"
ENV COAP_GATEWAY_UNSECURE_ENABLED=true
ENV COAP_GATEWAY_PORT=5684
ENV COAP_GATEWAY_ADDRESS="0.0.0.0:$COAP_GATEWAY_PORT"
ENV COAP_GATEWAY_SCHEME="coaps+tcp"
ENV COAP_GATEWAY_LOG_MESSAGES=true
ENV COAP_GATEWAY_EXTERNAL_ADDRESS=""
#deprecated
ENV COAP_GATEWAY_HUB_ID=""

# ports
ENV NGINX_PORT=443
ENV CERTIFICATE_AUTHORITY_PORT=9087
ENV HTTP_CERTIFICATE_AUTHORITY_PORT=9090
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
ENV SCYLLA_SMP=1
ENV SCYLLA_DEVELOPER_MODE=true
ENV SCYLLA_PORT=29142
ENV SNIPPET_SERVICE_PORT=9091
ENV HTTP_SNIPPET_SERVICE_PORT=9092
ENV M2M_OAUTH_SERVER_PORT=9079
ENV HTTP_M2M_OAUTH_SERVER_PORT=9080
ENV GRPC_REFLECTION_PORT=9078

# OAuth
ENV DEVICE_PROVIDER=plgd
ENV DEVICE_OAUTH_SCOPES="offline_access"
ENV OWNER_CLAIM="sub"
ENV MOCK_OAUTH_SERVER_ACCESS_TOKEN_LIFETIME="0s"

ENTRYPOINT ["/usr/local/bin/run.sh"]
