FROM golang:1.17.3-alpine AS build
ARG DIRECTORY
ARG NAME
RUN apk add --no-cache curl git build-base
WORKDIR $GOPATH/src/github.com/plgd-dev/hub
COPY go.mod go.sum ./
RUN go mod download
COPY . .
WORKDIR $GOPATH/src/github.com/plgd-dev/hub/tools/cert-tool
RUN go build -o /go/bin/cert-tool ./cmd

FROM alpine:3.14 as service
COPY --from=build /go/bin/cert-tool /usr/local/bin/cert-tool
ENTRYPOINT [ "/usr/local/bin/cert-tool" ]