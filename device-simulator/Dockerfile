FROM ubuntu:20.04 AS service

RUN apt-get update -y && \
 apt-get install -y bash curl gdb git-core g++ ca-certificates patch make --no-install-recommends
RUN git clone --recursive -b plgd.dev https://github.com/iotivity/iotivity-lite.git
COPY ./src/ /iotivity-lite/
RUN make -C /iotivity-lite/port/linux CLOUD=1 SECURE=1 OSCORE=0 DEBUG=1 MNT=1 ASAN=0 cloud_server
#RUN curl -sSfL https://github.com/mapbox/logbt/archive/v3.0.0.tar.gz | tar --gunzip --extract --strip-components=1 --exclude="*md" --exclude="test*" --directory=/usr/local
COPY logbt /usr/local/bin/logbt
RUN which logbt
RUN logbt --version
COPY run.sh /usr/local/bin/run.sh
ENV NUM_DEVICES=1
ENTRYPOINT ["/usr/local/bin/run.sh"]