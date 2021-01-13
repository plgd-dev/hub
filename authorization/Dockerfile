FROM cloud-build AS build
WORKDIR $GOPATH/src/github.com/plgd-dev/cloud/authorization
RUN go build -o /go/bin/service ./cmd/service
RUN mkdir -p /data
COPY authorization.yaml /data/authorization.yaml

FROM alpine:3.12 as service
RUN apk add --no-cache ca-certificates
COPY --from=build /go/bin/service /usr/local/bin/service
RUN mkdir -p /data
COPY --from=build /data/authorization.yaml /data/authorization.yaml
ls /data
#COPY --from=build ../.tmp/certs /data/certs
ENTRYPOINT ["/usr/local/bin/service", "--config=/data/authorization.yaml"]

