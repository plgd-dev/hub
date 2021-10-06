FROM ubuntu:20.04 AS service

RUN apt-get update -y && \
  apt-get install -y bash curl gdb git-core g++ ca-certificates patch make --no-install-recommends
ARG IOTIVITY_COMMIT=5748c5a706cfb9d056a4c4a2f5c4ad91f7fa12a0
ARG GTEST_COMMIT=c99458533a9b4c743ed51537e25989ea55944908
ARG MBEDTLS_COMMIT=d81c11b8ab61fd5b2da8133aa73c5fe33a0633eb
ARG TINYCBOR_COMMIT=70aba6ba51881e5b8d108c105a17ed9cdee6bc30
RUN git clone --recursive https://github.com/iotivity/iotivity-lite.git && \
  cd /iotivity-lite && git checkout $IOTIVITY_COMMIT && \
  cd /iotivity-lite/deps/gtest && git checkout $GTEST_COMMIT && \
  cd /iotivity-lite/deps/mbedtls && git checkout $MBEDTLS_COMMIT && \
  cd /iotivity-lite/deps/tinycbor && git checkout $TINYCBOR_COMMIT
RUN make -C /iotivity-lite/port/linux CLOUD=1 SECURE=1 OSCORE=0 DEBUG=1 MNT=1 CREATE=1 ASAN=0 cloud_server
#RUN curl -sSfL https://github.com/mapbox/logbt/archive/v3.0.0.tar.gz | tar --gunzip --extract --strip-components=1 --exclude="*md" --exclude="test*" --directory=/usr/local
COPY logbt /usr/local/bin/logbt
RUN which logbt
RUN logbt --version
COPY run.sh /usr/local/bin/run.sh
ENV NUM_DEVICES=1
ENTRYPOINT ["/usr/local/bin/run.sh"]
