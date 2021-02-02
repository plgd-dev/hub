FROM cloud-build AS build
WORKDIR $GOPATH/src/github.com/plgd-dev/cloud/resource-directory
RUN go build -o /go/bin/service ./cmd/service
RUN mkdir -p /data/
COPY resource-directory.yaml /data/resource-directory.yaml

FROM alpine:3.12 as service
RUN apk add --no-cache ca-certificates
RUN mkdir -p /data
COPY --from=build /go/bin/service /usr/local/bin/service
COPY --from=build /data/resource-directory.yaml /data/resource-directory.yaml

ENTRYPOINT ["/usr/local/bin/service", "--config=/data/resource-directory.yaml"]
