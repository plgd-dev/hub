FROM golang:1.13.5-alpine3.10 AS ocf-cloud-build
RUN apk add --no-cache curl git build-base
WORKDIR $GOPATH/src/github.com/go-ocf/ocf-cloud
COPY go.mod go.sum ./
RUN go mod download
COPY . .