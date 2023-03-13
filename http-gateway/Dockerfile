FROM golang:1.19.6-alpine AS build
RUN apk add --no-cache curl git build-base
WORKDIR $GOPATH/src/github.com/plgd-dev/hub
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN go mod vendor
RUN ( cd /usr/local/go && patch -p1 < $GOPATH/src/github.com/plgd-dev/hub/tools/docker/patches/shrink_tls_conn.patch )
RUN ( cd ./vendor/golang.org/x/oauth2 && patch -p1 < $GOPATH/src/github.com/plgd-dev/hub/tools/docker/patches/golang_org_x_oauth2_propagate_error.patch )
WORKDIR $GOPATH/src/github.com/plgd-dev/hub/http-gateway
RUN CGO_ENABLED=0 go build -mod=vendor -o /go/bin/http-gateway ./cmd/service

FROM node:16 AS build-web
COPY http-gateway/web /web
RUN cd /web && \
    npm config set fetch-retries 3 && \
    npm config set fetch-retry-mintimeout 600000 &&  \
    npm config set fetch-retry-maxtimeout 1200000 && \
    npm config set fetch-timeout 1800000 && \
    npm install && \
    npm run build

FROM alpine:3.17 AS security-provider
RUN apk add -U --no-cache ca-certificates
RUN addgroup -S nonroot \
    && adduser -S nonroot -G nonroot

FROM scratch AS service
COPY --from=security-provider /etc/passwd /etc/passwd
COPY --from=security-provider /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=build-web /web/build /usr/local/var/www
COPY --from=build /go/bin/http-gateway /usr/local/bin/http-gateway
USER nonroot
ENTRYPOINT [ "/usr/local/bin/http-gateway" ]