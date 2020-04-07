FROM alpine:3.8 AS build

RUN apk add --no-cache curl git build-base gcc linux-headers
RUN git clone --recursive https://github.com/iotivity/iotivity-lite.git
COPY ./patches/devsim.diff /devsim.diff
RUN (cd /iotivity-lite && git checkout e2d38bd7718ed4914060287ab4dfe47f03b832a0 && patch -p1 < /devsim.diff)
RUN make -C /iotivity-lite/port/linux CLOUD=1 SECURE=0 DEBUG=1 cloud_server

FROM alpine:3.8 AS service
RUN apk add --no-cache ca-certificates
COPY --from=build /iotivity-lite/port/linux/cloud_server /usr/local/bin/service
ENTRYPOINT ["/usr/local/bin/service"]
