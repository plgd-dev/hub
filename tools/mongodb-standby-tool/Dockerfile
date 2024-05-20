FROM alpine:3.19 AS security-provider
RUN apk add -U --no-cache ca-certificates
RUN addgroup -S nonroot \
    && adduser -S nonroot -G nonroot

FROM ubuntu:22.04

RUN apt-get update && apt-get install -y \
    wget \
    curl \
    jq \
    bash \
    && rm -rf /var/lib/apt/lists/*

# Install mongosh
RUN \
    ARCH=$(uname -m) && \
    if [ "$ARCH" = "x86_64" ]; then \
        ARCH="x64"; \
    fi && \
    wget https://downloads.mongodb.com/compass/mongosh-2.2.6-linux-$ARCH.tgz -O /tmp/mongosh.tgz && \
    tar -xzf /tmp/mongosh.tgz -C /tmp && \
    mv /tmp/mongosh-2.2.6-linux-x64/bin/* /usr/bin/

# Install yq
RUN \
    ARCH=$(uname -m) && \
    if [ "$ARCH" = "x86_64" ]; then \
        ARCH="amd64"; \
    fi && \
    wget https://github.com/mikefarah/yq/releases/download/v4.44.1/yq_linux_$ARCH -O /usr/bin/yq && chmod +x /usr/bin/yq

COPY --from=security-provider /etc/passwd /etc/passwd
COPY --from=security-provider /etc/group /etc/group
COPY --from=security-provider /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
RUN mkdir /home/nonroot && chown nonroot:nonroot /home/nonroot
USER nonroot
COPY --chown=nonroot:nonroot ./tools/mongodb-standby-tool/common.sh /common.sh
COPY --chown=nonroot:nonroot ./tools/mongodb-standby-tool/set_active.sh /set_active.sh
COPY --chown=nonroot:nonroot ./tools/mongodb-standby-tool/set_standby.sh /set_standby.sh
COPY --chown=nonroot:nonroot ./tools/mongodb-standby-tool/config.yaml /config.yaml
ENTRYPOINT [ "/bin/bash" ]
