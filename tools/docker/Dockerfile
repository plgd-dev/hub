FROM golang:1.18.1-alpine AS build
ARG DIRECTORY
ARG NAME
RUN apk add --no-cache curl git build-base
WORKDIR $GOPATH/src/github.com/plgd-dev/hub
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN ( cd /usr/local/go && patch -p1 < $GOPATH/src/github.com/plgd-dev/hub/tools/docker/patches/shrink_tls_conn.patch )
WORKDIR $GOPATH/src/github.com/plgd-dev/hub/$DIRECTORY
RUN go build -o /go/bin/$NAME ./cmd/service

FROM alpine:3.15 as service
ARG NAME
RUN apk add --no-cache ca-certificates
COPY --from=build /go/bin/$NAME /usr/local/bin/$NAME
COPY tools/docker/run.sh /usr/local/bin/run.sh
ENV BINARY=$NAME
ENTRYPOINT [ "/usr/local/bin/run.sh" ]