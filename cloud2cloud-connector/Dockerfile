FROM cloud-build AS build
WORKDIR $GOPATH/src/github.com/plgd-dev/cloud/cloud2cloud-connector
RUN go build -o /go/bin/service ./cmd/service
RUN mkdir -p /data/
COPY c2c-connector.yaml /data/c2c-connector.yaml

FROM alpine:3.12 as service
RUN apk add --no-cache ca-certificates
RUN mkdir -p /data
COPY --from=build /go/bin/service /usr/local/bin/service
COPY --from=build /data/c2c-connector.yaml /data/c2c-connector.yaml

ENTRYPOINT ["/usr/local/bin/service", "--config=/data/c2c-connector.yaml"]
