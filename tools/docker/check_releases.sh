#!/bin/bash

RELEASES=(
    v2.0.0
    #start index=1
    v2.1.0 2.1.0 2.1.0 2.1.2
    #start index=5
    2.2.0 2.2.1 2.2.2 2.2.3 2.2.4
    #start index=10
    2.3.0 2.3.1 2.3.2 2.3.3 2.3.4 2.3.5 2.3.6 2.3.7 2.3.8 2.3.9 2.3.10
    #start index=21
    2.4.0 2.4.1 2.4.2 2.4.3 2.4.4 2.4.5 2.4.6 2.4.7 2.4.8
    #start index=30
    2.5.0 2.5.1
    #start index=32
    2.6.0 2.6.1 2.6.2
    #start index=35
    2.7.0 2.7.1 2.7.2 2.7.3 2.7.4 2.7.5 2.7.6 2.7.7 2.7.8 2.7.9 2.7.10 2.7.11 2.7.12 2.7.13 2.7.14 2.7.15 2.7.16
)

PACKAGES=(
    "test-cloud-server"
    "cert-tool"
    "grpc-gateway"
    "coap-gateway"
    "resource-directory"
    "resource-aggregate"
    "identity-store"
    "http-gateway"
    "certificate-authority"
    "mock-oauth-server"
    "coap-gateway-go1-18"
    "bundle"
    "cloud2cloud-gateway"
    "cloud2cloud-connector"
)

PACKAGE_RELEASES=("${RELEASES[@]}")

if [[ -z "${PACKAGE}" ]]; then
    echo "ERROR: package not set" >&2
    exit 1
fi

if [[ "${PACKAGE}" == "test-cloud-server" ]]; then
    # all
    :
fi

if [[ "${PACKAGE}" == "cert-tool" ]]; then
    # available from version 2.2.0
    PACKAGE_RELEASES=("${RELEASES[@]:5}")
fi

if [[ "${PACKAGE}" == "grpc-gateway" ]]; then
    # available from version v2.1.0
    PACKAGE_RELEASES=("${RELEASES[@]:1}")
fi

if [[ "${PACKAGE}" == "coap-gateway" ]]; then
    # available from version v2.1.0
    PACKAGE_RELEASES=("${RELEASES[@]:1}")
fi

if [[ "${PACKAGE}" == "resource-directory" ]]; then
    # available from version v2.1.0
    PACKAGE_RELEASES=("${RELEASES[@]:1}")
fi

if [[ "${PACKAGE}" == "resource-aggregate" ]]; then
    # available from version v2.1.0
    PACKAGE_RELEASES=("${RELEASES[@]:1}")
fi

if [[ "${PACKAGE}" == "identity-store" ]]; then
    # available from version v2.1.0
    PACKAGE_RELEASES=("${RELEASES[@]:1}")
fi

if [[ "${PACKAGE}" == "http-gateway" ]]; then
    # available from version v2.1.0
    PACKAGE_RELEASES=("${RELEASES[@]:1}")
fi

if [[ "${PACKAGE}" == "certificate-authority" ]]; then
    # available from version v2.1.0
    PACKAGE_RELEASES=("${RELEASES[@]:1}")
fi

if [[ "${PACKAGE}" == "mock-oauth-server" ]]; then
    # available from version v2.1.0
    PACKAGE_RELEASES=("${RELEASES[@]:1}")
fi

if [[ "${PACKAGE}" == "coap-gateway-go1-18" ]]; then
    # available from version 2.7.9
    PACKAGE_RELEASES=("${RELEASES[@]:44}")
fi

if [[ "${PACKAGE}" == "bundle" ]]; then
    # available from version v2.1.0
    PACKAGE_RELEASES=("${RELEASES[@]:1}")
fi

if [[ "${PACKAGE}" == "cloud2cloud-gateway" ]]; then
    # available from version v2.1.0
    PACKAGE_RELEASES=("${RELEASES[@]:1}")
fi

if [[ "${PACKAGE}" == "cloud2cloud-connector" ]]; then
    # available from version v2.1.0
    PACKAGE_RELEASES=("${RELEASES[@]:1}")
fi

MISSING_PACKAGES=()
for i in "${PACKAGE_RELEASES[@]}"; do
    echo "Checking ${PACKAGE}:${i}"
    if docker pull ghcr.io/plgd-dev/hub/${PACKAGE}:${i}; then
        docker rmi ghcr.io/plgd-dev/hub/${PACKAGE}:${i} > /dev/null
    else
        echo "ERROR: ${PACKAGE}:${i} not found"
        MISSING_RELEASES+=(${i})
    fi
done

echo ""
echo "Missing releases:"
printf '%s ' "${MISSING_RELEASES[@]}"
echo ""
