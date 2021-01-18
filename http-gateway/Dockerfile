FROM cloud-build AS build
WORKDIR $GOPATH/src/github.com/plgd-dev/cloud/http-gateway
RUN go build -o /go/bin/service ./cmd/service

FROM ubuntu:20.04 AS build-web
RUN apt-get update && apt-get install -y gcc g++ make curl sudo
RUN curl -sL https://deb.nodesource.com/setup_12.x | sudo -E bash -
RUN apt-get update
RUN apt-get install -y nodejs
COPY web /web
RUN cd /web && npm install && npm run build

FROM alpine:3.12 as service
RUN apk add --no-cache ca-certificates
COPY --from=build /go/bin/service /usr/local/bin/service
COPY --from=build-web /web/build /usr/local/var/www
ENTRYPOINT ["/usr/local/bin/service"]