FROM golang:1.13.5-alpine3.10 AS build
RUN apk add --no-cache curl git build-base
WORKDIR $GOPATH/src/github.com/go-ocf/cloud
COPY . .
RUN go mod download

ARG root_directory=$GOPATH/src/github.com/go-ocf/cloud

#coap-gateway
ARG service=coap-gateway
WORKDIR $root_directory/$service
RUN go build -ldflags "-linkmode external -extldflags -static" -o /go/bin/$service ./cmd/service

#grpc-gateway
ARG service=grpc-gateway
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

#authorization
ARG service=authorization
WORKDIR $root_directory/$service
RUN go build -ldflags "-linkmode external -extldflags -static" -o /go/bin/$service ./cmd/service

#certificate-generator
ARG service=kit
WORKDIR /
RUN cd $GOPATH/pkg/mod/github.com/go-ocf/kit* && go build -ldflags "-linkmode external -extldflags -static" -o /go/bin/certificate-generator ./cmd/certificate-generator

#nats
WORKDIR $root_directory
RUN curl -L https://github.com/nats-io/nats-server/releases/download/v2.1.4/nats-server-v2.1.4-linux-amd64.zip -o ./nats-server.zip
RUN mkdir -p ./nats-server
RUN unzip ./nats-server.zip -d ./nats-server
RUN cp ./nats-server/*/nats-server /go/bin/nats-server

FROM ubuntu:18.04 as service
RUN apt update
RUN apt install -y mongodb-server
COPY --from=build /go/bin/coap-gateway /usr/local/bin/coap-gateway
COPY --from=build /go/bin/grpc-gateway /usr/local/bin/grpc-gateway
COPY --from=build /go/bin/resource-directory /usr/local/bin/resource-directory
COPY --from=build /go/bin/resource-aggregate /usr/local/bin/resource-aggregate
COPY --from=build /go/bin/authorization /usr/local/bin/authorization
COPY --from=build /go/bin/certificate-generator /usr/local/bin/certificate-generator
COPY --from=build /go/bin/nats-server /usr/local/bin/nats-server
COPY run.sh /usr/local/bin/run.sh

# coap-gateway
ENV COAP_GATEWAY_UNSECURE_PORT=5683
ENV COAP_GATEWAY_UNSECURE_ADDRESS="0.0.0.0:$COAP_GATEWAY_UNSECURE_PORT"
ENV COAP_GATEWAY_UNSECURE_FQDN="localhost"
ENV COAP_GATEWAY_PORT=5684
ENV COAP_GATEWAY_ADDRESS="0.0.0.0:$COAP_GATEWAY_PORT"
ENV COAP_GATEWAY_FQDN="localhost"
ENV COAP_GATEWAY_CLOUD_ID="00000000-0000-0000-0000-000000000001"
ENV COAP_GATEWAY_DISABLE_VERIFY_CLIENTS=true

# grpc-gateway
ENV GRPC_GATEWAY_ADDRESS="0.0.0.0:9084"
ENV GRPC_GATEWAY_DISABLE_VERIFY_CLIENTS=true

# authorization
ENV AUTHORIZATION_ADDRESS="localhost:9081"
ENV AUTHORIZATION_HTTP_ADDRESS="localhost:9085"

ENV JWKS_URL="https://$AUTHORIZATION_HTTP_ADDRESS/.well-known/jwks.json"
ENV RESOURCE_AGGREGATE_ADDRESS="localhost:9083"
ENV RESOURCE_DIRECTORY_ADDRESS="localhost:9082"
ENV INITIALIZE_CERITIFICATES=true
ENV MONGO_PATH="/data/db"
ENV MONGO_PORT=10000
ENV NATS_PORT=10001
ENV CERITIFICATES_PATH="/data/certs"
ENV LOGS_PATH="/data/log"

ENTRYPOINT ["/usr/local/bin/run.sh"]