FROM golang:1.18.1-alpine AS build
ARG DIRECTORY
ARG NAME
RUN apk add --no-cache curl git build-base
WORKDIR $GOPATH/src/github.com/plgd-dev/hub
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN ( cd /usr/local/go && patch -p1 < $GOPATH/src/github.com/plgd-dev/hub/tools/docker/patches/shrink_tls_conn.patch )
WORKDIR $GOPATH/src/github.com/plgd-dev/hub/tools/cert-tool
RUN go build -o /go/bin/cert-tool ./cmd

FROM alpine:3.15 as service
COPY --from=build /go/bin/cert-tool /usr/local/bin/cert-tool
ENTRYPOINT [ "/usr/local/bin/cert-tool" ]