FROM golang:1.20.4-alpine AS build
ARG DIRECTORY
ARG NAME
RUN apk add --no-cache curl git build-base
WORKDIR $GOPATH/src/github.com/plgd-dev/hub
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN ( cd /usr/local/go && patch -p1 < $GOPATH/src/github.com/plgd-dev/hub/tools/docker/patches/shrink_tls_conn.patch )
WORKDIR $GOPATH/src/github.com/plgd-dev/hub/tools/cert-tool
RUN CGO_ENABLED=0 go build -o /go/bin/cert-tool ./cmd

FROM alpine:3.17 AS security-provider
RUN addgroup -S nonroot \
    && adduser -S nonroot -G nonroot

FROM scratch AS service
COPY --from=security-provider /etc/passwd /etc/passwd
COPY --from=build /go/bin/cert-tool /usr/local/bin/cert-tool
USER nonroot
ENTRYPOINT [ "/usr/local/bin/cert-tool" ]