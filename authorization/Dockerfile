FROM golang:1.13.5-alpine3.10 AS build
RUN apk add --no-cache curl git build-base
WORKDIR $GOPATH/src/github.com/go-ocf/cloud/authorization
COPY . .
RUN go mod download
RUN go build -o /go/bin/service ./cmd/service

FROM alpine:3.11 as service
RUN apk add --no-cache ca-certificates
COPY --from=build /go/bin/service /usr/local/bin/service
ENTRYPOINT ["/usr/local/bin/service"]