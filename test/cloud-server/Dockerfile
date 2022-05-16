FROM golang:1.18.1-alpine AS build
RUN apk add --no-cache curl git build-base
WORKDIR $GOPATH/src/github.com/plgd-dev/hub
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN ( cd /usr/local/go && patch -p1 < $GOPATH/src/github.com/plgd-dev/hub/tools/docker/patches/shrink_tls_conn.patch )
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

FROM ubuntu:20.04 as service
RUN apt update
RUN apt install -y wget gnupg iproute2 systemctl openssl nginx ca-certificates netcat
RUN wget -qO - https://www.mongodb.org/static/pgp/server-4.4.asc | apt-key add -
RUN ARCH="$(dpkg --print-architecture)" ; \
    wget https://github.com/mikefarah/yq/releases/download/v4.6.3/yq_linux_${ARCH} -O /usr/bin/yq && chmod +x /usr/bin/yq ; \
    echo "deb [ arch=${ARCH} ] https://repo.mongodb.org/apt/ubuntu focal/mongodb-org/4.4 multiverse" | tee /etc/apt/sources.list.d/mongodb-org-4.4.list
RUN apt update
RUN apt-get install -y mongodb-org-server mongodb-org
COPY --from=build /go/bin/certificate-generator /usr/local/bin/certificate-generator
COPY --from=build /go/bin/grpc-gateway.test /usr/local/bin/grpc-gateway.test
COPY --from=build /go/bin/nats-server /usr/local/bin/nats-server
COPY --from=build /go/bin/nats /usr/local/bin/nats
COPY test/cloud-server/run.sh /usr/local/bin/run.sh

ENV FQDN="localhost"

# ports
ENV MONGO_PORT=27017
ENV NATS_PORT=4222


ENTRYPOINT ["/usr/local/bin/run.sh"]
