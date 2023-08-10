FROM golang:1.20.7-alpine AS build
RUN apk add --no-cache curl git build-base
WORKDIR $GOPATH/src/github.com/plgd-dev/hub
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN go mod vendor
RUN ( cd /usr/local/go && patch -p1 < $GOPATH/src/github.com/plgd-dev/hub/tools/docker/patches/shrink_tls_conn.patch )
RUN ( cd ./vendor/golang.org/x/oauth2 && patch -p1 < $GOPATH/src/github.com/plgd-dev/hub/tools/docker/patches/golang_org_x_oauth2_propagate_error.patch )
WORKDIR $GOPATH/src/github.com/plgd-dev/hub/tools/nats-server-config-reloader
RUN CGO_ENABLED=0 go build -mod=vendor -o /go/bin/nats-server-config-reloader .

FROM alpine:3.18 AS security-provider
RUN apk add -U --no-cache ca-certificates

FROM scratch AS service
COPY --from=security-provider /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=build /go/bin/nats-server-config-reloader /usr/local/bin/nats-server-config-reloader
ENTRYPOINT [ /usr/local/bin/nats-server-config-reloader ]