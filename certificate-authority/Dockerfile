FROM cloud-build AS build
WORKDIR $GOPATH/src/github.com/plgd-dev/cloud/certificate-authority
RUN go build -o /go/bin/service ./cmd/service
RUN mkdir -p /data/
COPY certificate-authority.yaml /data/certificate-authority.yaml

FROM alpine:3.12 as service
RUN apk add --no-cache ca-certificates
RUN mkdir -p /data
COPY --from=build /go/bin/service /usr/local/bin/service
COPY --from=build /data/certificate-authority.yaml /data/certificate-authority.yaml

ENTRYPOINT ["/usr/local/bin/service", "--config=/data/certificate-authority.yaml"]
