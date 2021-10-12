#!/usr/bin/env bash
set -e

umask 0000

# Configure services
export PATH="/usr/local/bin:$PATH"

export COAP_GATEWAY_CLOUD_ID="adebc667-1f2b-41e3-bf5c-6d6eabc68cc6"

export CERTIFICATES_PATH="/data/certs"
export OAUTH_KEYS_PATH="/data/oauth/keys"
export OAUTH_SECRETS_PATH="/data/oauth/secrets"
export LOGS_PATH="/data/log"
export MONGO_PATH="/data/db"

export INTERNAL_CERT_DIR_PATH="$CERTIFICATES_PATH/internal"
export GRPC_INTERNAL_CERT_NAME="endpoint.crt"
export GRPC_INTERNAL_CERT_KEY_NAME="endpoint.key"

export EXTERNAL_CERT_DIR_PATH="$CERTIFICATES_PATH/external"
export COAP_GATEWAY_FILE_CERT_NAME="coap-gateway.crt"
export COAP_GATEWAY_FILE_CERT_KEY_NAME="coap-gateway.key"

# ROOT CERTS
export CA_POOL_DIR="$CERTIFICATES_PATH"
export CA_POOL_NAME_PREFIX="root_ca"
export CA_POOL_CERT_PATH="$CA_POOL_DIR/$CA_POOL_NAME_PREFIX.crt"
export CA_POOL_CERT_KEY_PATH="$CA_POOL_DIR/$CA_POOL_NAME_PREFIX.key"

# DIAL CERTS
export DIAL_FILE_CA_POOL="$CA_POOL_CERT_PATH"
export DIAL_FILE_CERT_DIR_PATH="$INTERNAL_CERT_DIR_PATH"
export DIAL_FILE_CERT_NAME="$GRPC_INTERNAL_CERT_NAME"
export DIAL_FILE_CERT_KEY_NAME="$GRPC_INTERNAL_CERT_KEY_NAME"

#LISTEN CERTS
export LISTEN_FILE_CA_POOL="$CA_POOL_CERT_PATH"
export LISTEN_FILE_CERT_DIR_PATH="$INTERNAL_CERT_DIR_PATH"
export LISTEN_FILE_CERT_NAME="$GRPC_INTERNAL_CERT_NAME"
export LISTEN_FILE_CERT_KEY_NAME="$GRPC_INTERNAL_CERT_KEY_NAME"

#SECRETS
export SECRETS_DIRECTORY=/data/secrets

#OAUTH-SEVER KEYS
export OAUTH_ID_TOKEN_KEY_PATH=${OAUTH_KEYS_PATH}/id-token.pem
export OAUTH_ACCESS_TOKEN_KEY_PATH=${OAUTH_KEYS_PATH}/access-token.pem

export OAUTH_DEVICE_SECRET_PATH=${OAUTH_SECRETS_PATH}/device.secret

#ENDPOINTS
export MONGODB_HOST="localhost:$MONGO_PORT"
export MONGODB_URI="mongodb://$MONGODB_HOST"
export NATS_HOST="localhost:$NATS_PORT"
export NATS_URL="nats://${NATS_HOST}"

# needed by grpc-gateway.test
export TEST_COAP_GW_CERT_FILE="${EXTERNAL_CERT_DIR_PATH}/${COAP_GATEWAY_FILE_CERT_NAME}"
export TEST_COAP_GW_KEY_FILE="${EXTERNAL_CERT_DIR_PATH}/${COAP_GATEWAY_FILE_CERT_KEY_NAME}"
export TEST_ROOT_CA_CERT="${CA_POOL_CERT_PATH}"
export TEST_ROOT_CA_KEY="${CA_POOL_CERT_KEY_PATH}"
export TEST_CLOUD_SID="${COAP_GATEWAY_CLOUD_ID}"
export TEST_OAUTH_SERVER_ID_TOKEN_PRIVATE_KEY="${OAUTH_ID_TOKEN_KEY_PATH}"
export TEST_OAUTH_SERVER_ACCESS_TOKEN_PRIVATE_KEY="${OAUTH_ACCESS_TOKEN_KEY_PATH}"

mkdir -p ${OAUTH_SECRETS_PATH}
if [ -z "${OAUTH_CLIENT_SECRET}" ]
then
  export OAUTH_CLIENT_SECRET="secret"
fi
echo -n ${OAUTH_CLIENT_SECRET} > ${OAUTH_DEVICE_SECRET_PATH}

mkdir -p $CA_POOL_DIR
mkdir -p $INTERNAL_CERT_DIR_PATH
mkdir -p $EXTERNAL_CERT_DIR_PATH
mkdir -p ${SECRETS_DIRECTORY}
ln -s ${SECRETS_DIRECTORY} /secrets

export CA_POOL=$CA_POOL_CERT_PATH
export CERT_FILE=$DIAL_FILE_CERT_DIR_PATH/$DIAL_FILE_CERT_NAME
export KEY_FILE=$DIAL_FILE_CERT_DIR_PATH/$DIAL_FILE_CERT_KEY_NAME

fqdnSAN="--cert.san.domain=$FQDN"
if ip route get $FQDN 2>/dev/null >/dev/null; then
  fqdnSAN="--cert.san.ip=$FQDN"
fi
echo "generating CA cert"
certificate-generator --cmd.generateRootCA --outCert=$CA_POOL_CERT_PATH --outKey=$CA_POOL_CERT_KEY_PATH --cert.subject.cn="Root CA"
echo "generating GRPC internal cert"
certificate-generator --cmd.generateCertificate --outCert=$DIAL_FILE_CERT_DIR_PATH/$DIAL_FILE_CERT_NAME --outKey=$DIAL_FILE_CERT_DIR_PATH/$DIAL_FILE_CERT_KEY_NAME --cert.subject.cn="localhost" --cert.san.domain="localhost" --cert.san.ip="0.0.0.0" --cert.san.ip="127.0.0.1" $fqdnSAN --signerCert=$CA_POOL_CERT_PATH --signerKey=$CA_POOL_CERT_KEY_PATH
echo "generating COAP-GW cert"
certificate-generator --cmd.generateIdentityCertificate=$COAP_GATEWAY_CLOUD_ID --outCert=$EXTERNAL_CERT_DIR_PATH/$COAP_GATEWAY_FILE_CERT_NAME --outKey=$EXTERNAL_CERT_DIR_PATH/$COAP_GATEWAY_FILE_CERT_KEY_NAME --cert.san.domain=$COAP_GATEWAY_FQDN --signerCert=$CA_POOL_CERT_PATH --signerKey=$CA_POOL_CERT_KEY_PATH
echo "generating NGINX cert"
certificate-generator --cmd.generateCertificate --outCert=$EXTERNAL_CERT_DIR_PATH/$DIAL_FILE_CERT_NAME --outKey=$EXTERNAL_CERT_DIR_PATH/$DIAL_FILE_CERT_KEY_NAME --cert.subject.cn="localhost" --cert.san.domain="localhost" --cert.san.ip="0.0.0.0" --cert.san.ip="127.0.0.1" $fqdnSAN --signerCert=$CA_POOL_CERT_PATH --signerKey=$CA_POOL_CERT_KEY_PATH

mkdir -p ${OAUTH_KEYS_PATH}
openssl genrsa -out ${OAUTH_ID_TOKEN_KEY_PATH} 4096
openssl ecparam -name prime256v1 -genkey -noout -out ${OAUTH_ACCESS_TOKEN_KEY_PATH}

mkdir -p $MONGO_PATH
mkdir -p $CERTIFICATES_PATH
mkdir -p $LOGS_PATH

# nats
echo "starting nats-server"
cat > /data/nats.config <<EOF
port: $NATS_PORT
max_pending: 128Mb
write_deadline: 10s
tls: {
  verify: true
  cert_file: "$DIAL_FILE_CERT_DIR_PATH/$DIAL_FILE_CERT_NAME"
  key_file: "$DIAL_FILE_CERT_DIR_PATH/$DIAL_FILE_CERT_KEY_NAME"
  ca_file: "$CA_POOL_CERT_PATH"
}
EOF

nats-server -c /data/nats.config >$LOGS_PATH/nats-server.log 2>&1 &
status=$?
nats_server_pid=$!
if [ $status -ne 0 ]; then
  echo "Failed to start nats-server: $status"
  sync
  cat $LOGS_PATH/nats-server.log
  exit $status
fi

# waiting for nats. Without wait, sometimes auth service didn't connect.
i=0
while true; do
  i=$((i+1))
  if nc -z localhost $NATS_PORT; then
    break
  fi
  echo "Try to reconnect to nats(${NATS_HOST}) $i"
  sleep 1
done

# mongo
echo "starting mongod"
cat $DIAL_FILE_CERT_DIR_PATH/$DIAL_FILE_CERT_NAME > $DIAL_FILE_CERT_DIR_PATH/mongo.key
cat $DIAL_FILE_CERT_DIR_PATH/$DIAL_FILE_CERT_KEY_NAME >> $DIAL_FILE_CERT_DIR_PATH/mongo.key
mongod --setParameter maxNumActiveUserIndexBuilds=64 --port $MONGO_PORT --dbpath $MONGO_PATH --sslMode requireSSL --sslCAFile $CA_POOL_CERT_PATH --sslPEMKeyFile $DIAL_FILE_CERT_DIR_PATH/mongo.key >$LOGS_PATH/mongod.log 2>&1 &
status=$?
mongo_pid=$!
if [ $status -ne 0 ]; then
  echo "Failed to start mongod: $status"
  sync
  cat $LOGS_PATH/mongod.log
  exit $status
fi

# waiting for mongo DB. Without wait, sometimes auth service didn't connect.
i=0
while true; do
  i=$((i+1))
  if openssl s_client -connect ${MONGODB_HOST} -cert ${INTERNAL_CERT_DIR_PATH}/${DIAL_FILE_CERT_NAME} -key ${INTERNAL_CERT_DIR_PATH}/${DIAL_FILE_CERT_KEY_NAME} <<< "Q" 2>/dev/null > /dev/null; then
    break
  fi
  echo "Try to reconnect to mongodb(${MONGODB_HOST}) $i"
  sleep 1
done

echo "starting test"
grpc-gateway.test -test.v -test.timeout 600s
