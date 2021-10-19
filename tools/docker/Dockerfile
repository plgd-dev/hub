FROM golang:1.16.4-alpine AS build
ARG DIRECTORY
ARG NAME
RUN apk add --no-cache curl git build-base
WORKDIR $GOPATH/src/github.com/plgd-dev/hub
COPY go.mod go.sum ./
RUN go mod download
COPY . .
WORKDIR $GOPATH/src/github.com/plgd-dev/hub/$DIRECTORY
RUN go build -o /go/bin/$NAME ./cmd/service

FROM alpine:3.14 as service
ARG NAME
RUN apk add --no-cache ca-certificates
COPY --from=build /go/bin/$NAME /usr/local/bin/$NAME
ENV BINARY=$NAME
ENTRYPOINT /usr/local/bin/$BINARY