FROM cloud-build AS build
WORKDIR $GOPATH/src/github.com/plgd-dev/cloud/coap-gateway/bootstrapper
RUN go build -o /go/bin/bootstrapper .

FROM alpine:3.12 as service
RUN apk add --no-cache ca-certificates
COPY --from=build /go/bin/bootstrapper /usr/local/bin/bootstrapper
ENTRYPOINT ["/usr/local/bin/bootstrapper"]