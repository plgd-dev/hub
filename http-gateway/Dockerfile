# syntax=docker/dockerfile:1
FROM golang:1.22.3-alpine AS build
ARG VERSION
ARG COMMIT_DATE
ARG SHORT_COMMIT
ARG DATE
ARG RELEASE_URL
RUN apk add --no-cache build-base curl git
WORKDIR $GOPATH/src/github.com/plgd-dev/hub
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN go mod vendor
WORKDIR /usr/local/go
RUN ( patch -p1 < "$GOPATH/src/github.com/plgd-dev/hub/tools/docker/patches/shrink_tls_conn.patch" )
WORKDIR $GOPATH/src/github.com/plgd-dev/hub/vendor/golang.org/x/oauth2
RUN ( patch -p1 < "$GOPATH/src/github.com/plgd-dev/hub/tools/docker/patches/golang_org_x_oauth2_propagate_error.patch" )
WORKDIR $GOPATH/src/github.com/plgd-dev/hub/http-gateway
RUN go build \
    -mod=vendor \
    -ldflags "-linkmode external -extldflags -static \
    -X github.com/plgd-dev/hub/v2/pkg/build.CommitDate=$COMMIT_DATE \
    -X github.com/plgd-dev/hub/v2/pkg/build.CommitHash=$SHORT_COMMIT \
    -X github.com/plgd-dev/hub/v2/pkg/build.BuildDate=$DATE \
    -X github.com/plgd-dev/hub/v2/pkg/build.Version=$VERSION \
    -X github.com/plgd-dev/hub/v2/pkg/build.ReleaseURL=$RELEASE_URL" \
    -o /go/bin/http-gateway \
    ./cmd/service

FROM alpine:3.20 AS security-provider
RUN apk add -U --no-cache ca-certificates \
    && addgroup -S nonroot \
    && adduser -S nonroot -G nonroot

FROM scratch AS service
COPY --from=security-provider /etc/passwd /etc/passwd
COPY --from=security-provider /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY http-gateway/web/build /usr/local/var/www
COPY --from=build /go/bin/http-gateway /usr/local/bin/http-gateway
USER nonroot
ENTRYPOINT [ "/usr/local/bin/http-gateway" ]
