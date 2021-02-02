FROM cloud-build AS build
WORKDIR $GOPATH/src/github.com/plgd-dev/cloud/resource-aggregate
RUN go build -o /go/bin/service ./cmd/service
RUN mkdir -p /data/
COPY resource-aggregate.yaml /data/resource-aggregate.yaml

FROM alpine:3.12 as service
RUN apk add --no-cache ca-certificates
RUN mkdir -p /data
COPY --from=build /go/bin/service /usr/local/bin/service
COPY --from=build /data/resource-aggregate.yaml /data/resource-aggregate.yaml

ENTRYPOINT ["/usr/local/bin/service", "--config=/data/resource-aggregate.yaml"]
