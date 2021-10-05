FROM cloud-build AS build
ARG root_directory=$GOPATH/src/github.com/plgd-dev/cloud

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


#certificate-generator
ARG service=kit
WORKDIR /
RUN cd $GOPATH/pkg/mod/github.com/plgd-dev/kit* && go build -ldflags "-linkmode external -extldflags -static" -o /go/bin/certificate-generator ./cmd/certificate-generator

#nats
WORKDIR $root_directory
RUN curl -L https://github.com/nats-io/nats-server/releases/download/v2.3.1/nats-server-v2.3.1-linux-amd64.zip -o ./nats-server.zip
RUN mkdir -p ./nats-server
RUN unzip ./nats-server.zip -d ./nats-server
RUN cp ./nats-server/*/nats-server /go/bin/nats-server

RUN curl -L https://github.com/nats-io/natscli/releases/download/0.0.24/nats-0.0.24-linux-amd64.zip -o ./nats.zip
RUN mkdir -p ./nats
RUN unzip ./nats.zip -d ./nats
RUN cp ./nats/*/nats /go/bin/nats

FROM node:12 AS build-web
COPY --from=cloud-build /go/src/github.com/plgd-dev/cloud/http-gateway/web /web
RUN cd /web && npm install && npm run build

FROM ubuntu:20.04 as service
RUN apt update
RUN apt install -y wget gnupg iproute2 systemctl openssl nginx ca-certificates netcat
RUN wget -qO - https://www.mongodb.org/static/pgp/server-4.4.asc | apt-key add -
RUN wget https://github.com/mikefarah/yq/releases/download/v4.6.3/yq_linux_amd64 -O /usr/bin/yq && chmod +x /usr/bin/yq
RUN echo "deb [ arch=amd64,arm64 ] https://repo.mongodb.org/apt/ubuntu focal/mongodb-org/4.4 multiverse" | tee /etc/apt/sources.list.d/mongodb-org-4.4.list
RUN apt update
RUN apt-get install -y mongodb-org-server mongodb-org
COPY --from=build /go/bin/coap-gateway /usr/local/bin/coap-gateway
COPY --from=cloud-build /go/src/github.com/plgd-dev/cloud/coap-gateway/config.yaml /configs/coap-gateway.yaml
COPY --from=build /go/bin/grpc-gateway /usr/local/bin/grpc-gateway
COPY --from=cloud-build /go/src/github.com/plgd-dev/cloud/grpc-gateway/config.yaml /configs/grpc-gateway.yaml
COPY --from=build /go/bin/http-gateway /usr/local/bin/http-gateway
COPY --from=cloud-build /go/src/github.com/plgd-dev/cloud/http-gateway/config.yaml /configs/http-gateway.yaml
COPY --from=build /go/bin/resource-directory /usr/local/bin/resource-directory
COPY --from=cloud-build /go/src/github.com/plgd-dev/cloud/resource-directory/config.yaml /configs/resource-directory.yaml
COPY --from=build /go/bin/resource-aggregate /usr/local/bin/resource-aggregate
COPY --from=cloud-build /go/src/github.com/plgd-dev/cloud/resource-aggregate/config.yaml /configs/resource-aggregate.yaml
COPY --from=build /go/bin/identity-store /usr/local/bin/identity-store
COPY --from=cloud-build /go/src/github.com/plgd-dev/cloud/identity-store/config.yaml /configs/identity-store.yaml
COPY --from=build /go/bin/certificate-authority /usr/local/bin/certificate-authority
COPY --from=cloud-build /go/src/github.com/plgd-dev/cloud/certificate-authority/config.yaml /configs/certificate-authority.yaml
COPY --from=build /go/bin/certificate-generator /usr/local/bin/certificate-generator
COPY --from=build /go/bin/nats-server /usr/local/bin/nats-server
COPY --from=build /go/bin/nats /usr/local/bin/nats
COPY --from=cloud-build /go/src/github.com/plgd-dev/cloud/bundle/jetstream.json /configs/jetstream.json
COPY --from=build /go/bin/oauth-server /usr/local/bin/oauth-server
COPY --from=cloud-build /go/src/github.com/plgd-dev/cloud/test/oauth-server/config.yaml /configs/oauth-server.yaml
COPY --from=build-web /web/build /usr/local/var/www
COPY --from=build /go/bin/cloud2cloud-gateway /usr/local/bin/cloud2cloud-gateway
COPY --from=cloud-build /go/src/github.com/plgd-dev/cloud/cloud2cloud-gateway/config.yaml /configs/cloud2cloud-gateway.yaml
COPY run.sh /usr/local/bin/run.sh
COPY nginx /nginx

# global
ENV FQDN="localhost"
ENV LOG_DEBUG=false
ENV JETSTREAM=false

# coap-gateway
ENV COAP_GATEWAY_UNSECURE_PORT=5683
ENV COAP_GATEWAY_UNSECURE_ADDRESS="0.0.0.0:$COAP_GATEWAY_UNSECURE_PORT"
ENV COAP_GATEWAY_PORT=5684
ENV COAP_GATEWAY_ADDRESS="0.0.0.0:$COAP_GATEWAY_PORT"
ENV COAP_GATEWAY_CLOUD_ID="00000000-0000-0000-0000-000000000001"
ENV COAP_GATEWAY_LOG_MESSAGES=true

# ports
ENV NGINX_PORT=443
ENV CERTIFICATE_AUTHORITY_PORT=9087
ENV MOCKED_OAUTH_SERVER_PORT=9088
ENV RESOURCE_AGGREGATE_PORT=9083
ENV RESOURCE_DIRECTORY_PORT=9082
ENV IDENTITY_STORE_PORT=9081
ENV GRPC_GATEWAY_PORT=9084
ENV HTTP_GATEWAY_PORT=9086
ENV CLOUD2CLOUD_GATEWAY_PORT=9085
ENV MONGO_PORT=10000
ENV NATS_PORT=10001

# OAuth
ENV DEVICE_PROVIDER=plgd
ENV DEVICE_OAUTH_SCOPES=profile,openid,offline_access
ENV OWNER_CLAIM="sub"

ENTRYPOINT ["/usr/local/bin/run.sh"]
