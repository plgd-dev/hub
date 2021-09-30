FROM cloud-build AS build
WORKDIR $GOPATH/src/github.com/plgd-dev/cloud/identity
RUN go build -o /go/bin/service ./cmd/service

FROM alpine:3.14 as service
RUN apk add --no-cache ca-certificates
COPY --from=build /go/bin/service /usr/local/bin/service
ENTRYPOINT ["/usr/local/bin/service"]