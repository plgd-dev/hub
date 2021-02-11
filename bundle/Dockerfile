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

#authorization
ARG service=authorization
WORKDIR $root_directory/$service
RUN go build -ldflags "-linkmode external -extldflags -static" -o /go/bin/$service ./cmd/service

#certificate-authority
ARG service=certificate-authority
WORKDIR $root_directory/$service
RUN go build -ldflags "-linkmode external -extldflags -static" -o /go/bin/$service ./cmd/service

#certificate-generator
ARG service=kit
WORKDIR /
RUN cd $GOPATH/pkg/mod/github.com/plgd-dev/kit* && go build -ldflags "-linkmode external -extldflags -static" -o /go/bin/certificate-generator ./cmd/certificate-generator

#nats
WORKDIR $root_directory
RUN curl -L https://github.com/nats-io/nats-server/releases/download/v2.1.9/nats-server-v2.1.9-linux-amd64.zip -o ./nats-server.zip
RUN mkdir -p ./nats-server
RUN unzip ./nats-server.zip -d ./nats-server
RUN cp ./nats-server/*/nats-server /go/bin/nats-server

FROM ubuntu:20.04 as service
RUN apt update
RUN apt install -y wget gnupg iproute2 systemctl openssl
RUN wget -qO - https://www.mongodb.org/static/pgp/server-4.4.asc | apt-key add -
RUN echo "deb [ arch=amd64,arm64 ] https://repo.mongodb.org/apt/ubuntu focal/mongodb-org/4.4 multiverse" | tee /etc/apt/sources.list.d/mongodb-org-4.4.list
RUN apt update
RUN apt-get install -y mongodb-org-server mongodb-org
COPY --from=build /go/bin/coap-gateway /usr/local/bin/coap-gateway
COPY --from=build /go/bin/grpc-gateway /usr/local/bin/grpc-gateway
COPY --from=build /go/bin/http-gateway /usr/local/bin/http-gateway
COPY --from=build /go/bin/resource-directory /usr/local/bin/resource-directory
COPY --from=build /go/bin/resource-aggregate /usr/local/bin/resource-aggregate
COPY --from=build /go/bin/authorization /usr/local/bin/authorization
COPY --from=build /go/bin/certificate-authority /usr/local/bin/certificate-authority
COPY --from=build /go/bin/certificate-generator /usr/local/bin/certificate-generator
COPY --from=build /go/bin/nats-server /usr/local/bin/nats-server
COPY run.sh /usr/local/bin/run.sh

# global
ENV FQDN="localhost"

# coap-gateway
ENV COAP_GATEWAY_UNSECURE_PORT=5683
ENV COAP_GATEWAY_UNSECURE_ADDRESS="0.0.0.0:$COAP_GATEWAY_UNSECURE_PORT"
ENV COAP_GATEWAY_PORT=5684
ENV COAP_GATEWAY_ADDRESS="0.0.0.0:$COAP_GATEWAY_PORT"
ENV COAP_GATEWAY_CLOUD_ID="00000000-0000-0000-0000-000000000001"
ENV COAP_GATEWAY_DISABLE_VERIFY_CLIENTS=true
ENV COAP_GATEWAY_DISABLE_BLOCKWISE_TRANSFER=false
ENV COAP_GATEWAY_BLOCKWISE_TRANSFER_SZX="1024"
ENV COAP_GATEWAY_DISABLE_PEER_TCP_SIGNAL_MESSAGE_CSMS=false
ENV COAP_GATEWAY_LOG_MESSAGES=true

# grpc-gateway
ENV GRPC_GATEWAY_ADDRESS="0.0.0.0:9084"
ENV GRPC_GATEWAY_DISABLE_VERIFY_CLIENTS=true

# http-gateway
ENV HTTP_GATEWAY_ADDRESS="0.0.0.0:9086"
ENV HTTP_GATEWAY_DISABLE_VERIFY_CLIENTS=true

# authorization
ENV AUTHORIZATION_ADDRESS="localhost:9081"
ENV AUTHORIZATION_HTTP_ADDRESS="0.0.0.0:9085"

# certificate-authority
ENV CERTIFICATE_AUTHORITY_ADDRESS="0.0.0.0:9087"
ENV CERTIFICATE_AUTHORITY_DISABLE_VERIFY_CLIENTS=true

ENV JWKS_URL="https://$AUTHORIZATION_HTTP_ADDRESS/.well-known/jwks.json"
ENV RESOURCE_AGGREGATE_ADDRESS="localhost:9083"
ENV RESOURCE_AGGREGATE_MONGO_MAX_PARALLEL_QUERIES="8"
ENV RESOURCE_DIRECTORY_ADDRESS="localhost:9082"
ENV INITIALIZE_CERITIFICATES=true
ENV MONGO_PATH="/data/db"
ENV MONGO_PORT=10000
ENV NATS_PORT=10001
ENV CERITIFICATES_PATH="/data/certs"
ENV LOGS_PATH="/data/log"

ENTRYPOINT ["/usr/local/bin/run.sh"]