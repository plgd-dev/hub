FROM cloud-test AS build
WORKDIR $GOPATH/src/github.com/plgd-dev/cloud/coap-gateway
RUN ls
RUN go build -o /go/bin/service ./cmd/service
RUN mkdir -p /data/
COPY ./coap-gw.yaml /data/coap-gw.yaml

WORKDIR $GOPATH/src/github.com/plgd-dev/cloud
RUN mkdir -p /data/certs/
RUN cp -a .tmp/certs/ /data/
#COPY .tmp/certs/ /data/
#RUN ls /data/certs/

FROM alpine:3.12 as service
RUN apk add --no-cache ca-certificates
COPY --from=build /go/bin/service /usr/local/bin/service
RUN mkdir -p /data
COPY --from=build /data/coap-gw.yaml /data/coap-gw.yaml
COPY --from=build /data/certs/ /data/certs/
#RUN ls /data/certs/

ENTRYPOINT ["/usr/local/bin/service", "--config=/data/coap-gw.yaml"]
