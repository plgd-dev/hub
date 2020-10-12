#!/usr/bin/env bash

if [ -z ${1+x} ]; then
    openssl ecparam -name secp256r1 -genkey -noout -out /tmp/mfg.key.pem
    cat /tmp/mfg.key.pem
    csr=`openssl req -new -sha256 -key /tmp/mfg.key.pem -subj "/CN=mfgCert" -addext extendedKeyUsage=serverAuth,clientAuth -addext keyUsage=digitalSignature,keyAgreement`
else
    csr=$1
fi

if [ -z ${2+x} ]; then
    token=`curl -s -k https://localhost:9085/api/authz/token | jq -r .access_token`
else
    token=$2
fi

if [ -z ${3+x} ]; then
    host="-k https://localhost:9086"
else
    host=$3
fi

curl -s -d "{\"CSR\":\"${csr}\"}" -H 'Content-Type: application/json' -H "Authorization: Bearer ${token}" ${host}/api/v1/certificate-authority/sign | jq -r .CertificateChain
