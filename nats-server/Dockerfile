FROM golang:1.15-alpine AS build
RUN apk add --no-cache curl git build-base
COPY ./patches/server.filestore.go.patch /server.filestore.go.patch
RUN git clone https://github.com/nats-io/nats-server.git
WORKDIR ./nats-server
RUN git checkout d7741b9aa16a7687c5426697b17a6eb5e5dd44f3
RUN patch -p1 < /server.filestore.go.patch
RUN go mod tidy
RUN go mod vendor
RUN go build -o /go/bin/service

FROM build as build-nats-server-static
RUN go build  -ldflags "-linkmode external -extldflags -static" -o /go/bin/service

FROM alpine:3.12 as service
RUN apk add --no-cache ca-certificates
COPY --from=build /go/bin/service /usr/local/bin/service
RUN mkdir /data && chown 1000:1000 /data
ENTRYPOINT ["/usr/local/bin/service"]