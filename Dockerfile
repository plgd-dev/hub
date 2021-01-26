FROM golang:1.15-alpine AS cloud-build
RUN apk add --no-cache curl git build-base
WORKDIR $GOPATH/src/github.com/plgd-dev/cloud
COPY go.mod go.sum ./
RUN go mod download
COPY . .