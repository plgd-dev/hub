FROM hub-build AS build
WORKDIR $GOPATH/src/github.com/plgd-dev/hub/http-gateway
RUN go build -o /go/bin/service ./cmd/service

FROM node:12 AS build-web
COPY web /web
RUN cd /web && npm install && npm run build

FROM alpine:3.14 as service
RUN apk add --no-cache ca-certificates
COPY --from=build /go/bin/service /usr/local/bin/service
COPY --from=build-web /web/build /usr/local/var/www
ENTRYPOINT ["/usr/local/bin/service"]