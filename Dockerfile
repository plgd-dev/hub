FROM golang:1.13.5-alpine3.10 AS test-build
RUN apk add --no-cache curl git build-base
WORKDIR $GOPATH/src/github.com/go-ocf/cloud
COPY . .
RUN go mod download