FROM cloud-build AS build
WORKDIR $GOPATH/src/github.com/plgd-dev/cloud/http-gateway
RUN go build -o /go/bin/service ./cmd/service
RUN mkdir -p /data/
COPY http-gateway.yaml /data/http-gateway.yaml

FROM node:12 AS build-web
COPY web /web
RUN cd /web && npm install && npm run build

FROM alpine:3.12 as service
RUN apk add --no-cache ca-certificates
RUN mkdir -p /data
COPY --from=build /go/bin/service /usr/local/bin/service
COPY --from=build-web /web/build /usr/local/var/www
COPY --from=build /data/http-gateway.yaml /data/http-gateway.yaml

ENTRYPOINT ["/usr/local/bin/service", "--config=/data/http-gateway.yaml"]
