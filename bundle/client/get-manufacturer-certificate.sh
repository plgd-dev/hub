#!/usr/bin/env bash

usage()
{
    echo "usage: get-manufacturer-certificate.sh [[[-c csr] [-t token] [-a address]] | [-h]]"
}

tmp_dir=$(mktemp -d -t mfgcert-XXXXXXXXXX)

while [ "$1" != "" ]; do
    case $1 in
        -c | --csr )            shift
                                csr="$1"
                                ;;
        -t | --token )          shift
                                token="$1"
                                ;;
        -a | --addr )           shift
                                host="$1"
                                ;;
                                        
        -h | --help )           usage
                                exit
                                ;;
        * )                     usage
                                exit 1
    esac
    shift
done

if [ -z ${csr+x} ]; then
    openssl ecparam -name prime256v1 -genkey -noout -out ${tmp_dir}/mfgcert.key.pem
    
    csr=`openssl req -new -sha256 -key ${tmp_dir}/mfgcert.key.pem -subj "/CN=mfgCert" -addext extendedKeyUsage=serverAuth,clientAuth -addext keyUsage=digitalSignature,keyAgreement`
fi

if [ -z ${token+x} ]; then
    token=`curl -s -k https://localhost:9085/api/authz/token | jq -r .access_token`
fi

if [ -z ${host+x} ]; then
    host="-k https://localhost:9086"
fi

curl -s -d "{\"CSR\":\"${csr}\"}" -H 'Content-Type: application/json' -H "Authorization: Bearer ${token}" ${host}/api/v1/certificate-authority/sign | jq -r .CertificateChain > ${tmp_dir}/mfgcert.pem
cat ${tmp_dir}/mfgcert.pem | tac | awk '/END/,/BEGIN/{ if(/BEGIN/){print; exit}; print}' | tac > ${tmp_dir}/mfgtrustca.pem
echo "{"
if [ -f ${tmp_dir}/mfgcert.key.pem ]; then
    echo "  \"mfgcert-key\": \"${tmp_dir}/mfgcert.key.pem\","
fi
echo "  \"mfgcert\": \"${tmp_dir}/mfgcert.pem\","
echo "  \"mfgtrustca\": \"${tmp_dir}/mfgtrustca.pem\""
echo "}"