FROM hub-build AS build
ARG root_directory=$GOPATH/src/github.com/plgd-dev/hub

#grpc-gateway
ARG service=grpc-gateway
WORKDIR $root_directory/$service/service
RUN go test -c -ldflags "-linkmode external -extldflags -static" -o /go/bin/grpc-gateway.test

#certificate-generator
ARG service=kit
WORKDIR /
RUN cd $GOPATH/pkg/mod/github.com/plgd-dev/kit/v2* && go build -ldflags "-linkmode external -extldflags -static" -o /go/bin/certificate-generator ./cmd/certificate-generator

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

FROM ubuntu:20.04 as service
RUN apt update
RUN apt install -y wget gnupg iproute2 systemctl openssl nginx ca-certificates netcat
RUN wget -qO - https://www.mongodb.org/static/pgp/server-4.4.asc | apt-key add -
RUN wget https://github.com/mikefarah/yq/releases/download/v4.6.3/yq_linux_amd64 -O /usr/bin/yq && chmod +x /usr/bin/yq
RUN echo "deb [ arch=amd64,arm64 ] https://repo.mongodb.org/apt/ubuntu focal/mongodb-org/4.4 multiverse" | tee /etc/apt/sources.list.d/mongodb-org-4.4.list
RUN apt update
RUN apt-get install -y mongodb-org-server mongodb-org
COPY --from=build /go/bin/certificate-generator /usr/local/bin/certificate-generator
COPY --from=build /go/bin/grpc-gateway.test /usr/local/bin/grpc-gateway.test
COPY --from=build /go/bin/nats-server /usr/local/bin/nats-server
COPY --from=build /go/bin/nats /usr/local/bin/nats
COPY run.sh /usr/local/bin/run.sh

ENV FQDN="localhost"

# ports
ENV MONGO_PORT=27017
ENV NATS_PORT=4222


ENTRYPOINT ["/usr/local/bin/run.sh"]
