FROM golang:1.14-alpine AS cloud-build
RUN apk add --no-cache curl git build-base
WORKDIR $GOPATH/src/github.com/go-ocf/cloud
COPY go.mod go.sum ./
RUN go mod download
COPY . .