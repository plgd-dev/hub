FROM golang:1.20.0-alpine AS build
RUN apk add --no-cache curl git build-base
WORKDIR $GOPATH/src/github.com/plgd-dev/hub
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN ( cd /usr/local/go && patch -p1 < $GOPATH/src/github.com/plgd-dev/hub/tools/docker/patches/shrink_tls_conn.patch )
WORKDIR $GOPATH/src/github.com/plgd-dev/hub/http-gateway
RUN go build -o /go/bin/http-gateway ./cmd/service

FROM node:16 AS build-web
COPY http-gateway/web /web
RUN cd /web && \
    npm config set fetch-retries 3 && \
    npm config set fetch-retry-mintimeout 600000 &&  \
    npm config set fetch-retry-maxtimeout 1200000 && \
    npm config set fetch-timeout 1800000 && \
    npm install && \
    npm run build

FROM alpine:3.17 as service
RUN apk add --no-cache ca-certificates
COPY --from=build-web /web/build /usr/local/var/www
COPY --from=build /go/bin/http-gateway /usr/local/bin/http-gateway
ENTRYPOINT [ "/usr/local/bin/http-gateway" ]