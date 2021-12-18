#!/usr/bin/env bash
set -e

# Configure services
export PATH="/usr/local/bin:$PATH"

export CERTIFICATES_PATH="/data/certs"
export CA_POOL_CUSTOM_PATH="$CERTIFICATES_PATH/ca_pool_custom.crt"
export CA_POOL="$CERTIFICATES_PATH/ca.crt"
export OAUTH_KEYS_PATH="/data/oauth/keys"
export OAUTH_SECRETS_PATH="/data/oauth/secrets"
export LOGS_PATH="/data/log"
export MONGO_PATH="/data/db"
export NGINX_PATH="/data/nginx"
export JETSTREAM_PATH="/data/jetstream"

export CERTIFICATE_AUTHORITY_ADDRESS="localhost:${CERTIFICATE_AUTHORITY_PORT}"
export MOCK_OAUTH_SERVER_ADDRESS="localhost:${MOCK_OAUTH_SERVER_PORT}"
export RESOURCE_AGGREGATE_ADDRESS="localhost:${RESOURCE_AGGREGATE_PORT}"
export RESOURCE_DIRECTORY_ADDRESS="localhost:${RESOURCE_DIRECTORY_PORT}"
export IDENTITY_STORE_ADDRESS="localhost:${IDENTITY_STORE_PORT}"
export GRPC_GATEWAY_ADDRESS="localhost:${GRPC_GATEWAY_PORT}"
export HTTP_GATEWAY_ADDRESS="localhost:${HTTP_GATEWAY_PORT}"
export CLOUD2CLOUD_GATEWAY_ADDRESS="localhost:${CLOUD2CLOUD_GATEWAY_PORT}"
export CLOUD2CLOUD_CONNECTOR_ADDRESS="localhost:${CLOUD2CLOUD_CONNECTOR_PORT}"

export INTERNAL_CERT_DIR_PATH="$CERTIFICATES_PATH/internal"
export GRPC_INTERNAL_CERT_NAME="endpoint.crt"
export GRPC_INTERNAL_CERT_KEY_NAME="endpoint.key"

export EXTERNAL_CERT_DIR_PATH="$CERTIFICATES_PATH/external"
export COAP_GATEWAY_FILE_CERT_NAME="coap-gateway.crt"
export COAP_GATEWAY_FILE_CERT_KEY_NAME="coap-gateway.key"

# ROOT CERTS
export ROOT_CERT_PATH="$CERTIFICATES_PATH/root_ca.crt"
export ROOT_KEY_PATH="$CERTIFICATES_PATH/root_ca.key"

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

export FQDN_NGINX_HTTPS=${FQDN}:${NGINX_PORT}
export DOMAIN=${FQDN_NGINX_HTTPS}
if [ "$NGINX_PORT" = "443" ]; then
  export DOMAIN=${FQDN}
fi

#OAUTH SERVER
if [ -z "${OAUTH_AUDIENCE}" ]
then
  export OAUTH_AUDIENCE=test
fi
export SERVICE_OAUTH_AUDIENCE=${OAUTH_AUDIENCE}
export DEVICE_OAUTH_AUDIENCE=${OAUTH_AUDIENCE}

if [ -z "${DEVICE_OAUTH_REDIRECT_URL}" ]
then
  export DEVICE_OAUTH_REDIRECT_URL="cloud.plgd.mobile://login-callback"
fi

if [ -z "${OAUTH_ENDPOINT}" ]
then
  export OAUTH_ENDPOINT=${DOMAIN}
fi

if [ -z "${OAUTH_CLIENT_ID}" ]
then
  export DEVICE_OAUTH_CLIENT_ID=test
  export OAUTH_CLIENT_ID=test
else
  export DEVICE_OAUTH_CLIENT_ID=${OAUTH_CLIENT_ID}
fi

mkdir -p ${OAUTH_SECRETS_PATH}
if [ -z "${OAUTH_CLIENT_SECRET}" ]
then
  export OAUTH_CLIENT_SECRET="secret"
fi
echo -n ${OAUTH_CLIENT_SECRET} > ${OAUTH_DEVICE_SECRET_PATH}

export COAP_GATEWAY_UNSECURE_FQDN=$FQDN
export COAP_GATEWAY_FQDN=$FQDN

mkdir -p $CERTIFICATES_PATH
mkdir -p $INTERNAL_CERT_DIR_PATH
mkdir -p $EXTERNAL_CERT_DIR_PATH
mkdir -p ${SECRETS_DIRECTORY}
ln -s ${SECRETS_DIRECTORY} /secrets

export CERT_FILE=$INTERNAL_CERT_DIR_PATH/$GRPC_INTERNAL_CERT_NAME
export KEY_FILE=$INTERNAL_CERT_DIR_PATH/$GRPC_INTERNAL_CERT_KEY_NAME

fqdnSAN="--cert.san.domain=$FQDN"
if ip route get $FQDN 2>/dev/null >/dev/null; then
  fqdnSAN="--cert.san.ip=$FQDN"
fi
echo "generating CA cert"
certificate-generator --cmd.generateRootCA --outCert=$ROOT_CERT_PATH --outKey=$ROOT_KEY_PATH --cert.subject.cn="Root CA"
echo "generating GRPC internal cert"
certificate-generator --cmd.generateCertificate --outCert=$INTERNAL_CERT_DIR_PATH/$GRPC_INTERNAL_CERT_NAME --outKey=$INTERNAL_CERT_DIR_PATH/$GRPC_INTERNAL_CERT_KEY_NAME --cert.subject.cn="localhost" --cert.san.domain="localhost" --cert.san.ip="0.0.0.0" --cert.san.ip="127.0.0.1" $fqdnSAN --signerCert=$ROOT_CERT_PATH --signerKey=$ROOT_KEY_PATH
echo "generating COAP-GW cert"
certificate-generator --cmd.generateIdentityCertificate=$COAP_GATEWAY_HUB_ID --outCert=$EXTERNAL_CERT_DIR_PATH/$COAP_GATEWAY_FILE_CERT_NAME --outKey=$EXTERNAL_CERT_DIR_PATH/$COAP_GATEWAY_FILE_CERT_KEY_NAME --cert.san.domain=$COAP_GATEWAY_FQDN --signerCert=$ROOT_CERT_PATH --signerKey=$ROOT_KEY_PATH
echo "generating NGINX cert"
certificate-generator --cmd.generateCertificate --outCert=$EXTERNAL_CERT_DIR_PATH/$GRPC_INTERNAL_CERT_NAME --outKey=$EXTERNAL_CERT_DIR_PATH/$GRPC_INTERNAL_CERT_KEY_NAME --cert.subject.cn="localhost" --cert.san.domain="localhost" --cert.san.ip="0.0.0.0" --cert.san.ip="127.0.0.1" $fqdnSAN --signerCert=$ROOT_CERT_PATH --signerKey=$ROOT_KEY_PATH

echo "ROOT_CERT_PATH=${ROOT_CERT_PATH} CA_POOL=${CA_POOL}"
cat ${ROOT_CERT_PATH} > ${CA_POOL}
if [ -f ${CA_POOL_CUSTOM_PATH} ]; then
echo "CA_POOL_CUSTOM_PATH=${CA_POOL_CUSTOM_PATH} CA_POOL=${CA_POOL}"
  cat ${CA_POOL_CUSTOM_PATH} >> ${CA_POOL}
fi

# copy ceritficates to paths
## oauth-server
while read -r line; do
  file=`echo $line | yq e '.[0]' - `
  mkdir -p `dirname ${file}`
  cp $CA_POOL ${file}
done < <(yq e '[.. | select(has("caPool")) | .caPool]' /configs/oauth-server.yaml | sort | uniq)
### setup certificates
while read -r line; do
  file=`echo $line | yq e '.[0]' - `
  mkdir -p `dirname ${file}`
  cp $CERT_FILE ${file}
done < <(yq e '[.. | select(has("certFile")) | .certFile]' /configs/oauth-server.yaml | sort | uniq)
### setup private keys
while read -r line; do
  file=`echo $line | yq e '.[0]' - `
  mkdir -p `dirname ${file}`
  cp $KEY_FILE ${file}
done < <(yq e '[.. | select(has("keyFile")) | .keyFile]' /configs/oauth-server.yaml | sort | uniq)

## identity-store
while read -r line; do
  file=`echo $line | yq e '.[0]' - `
  mkdir -p `dirname ${file}`
  cp $CA_POOL ${file}
done < <(yq e '[.. | select(has("caPool")) | .caPool]' /configs/identity-store.yaml | sort | uniq)
### setup certificates
while read -r line; do
  file=`echo $line | yq e '.[0]' - `
  mkdir -p `dirname ${file}`
  cp $CERT_FILE ${file}
done < <(yq e '[.. | select(has("certFile")) | .certFile]' /configs/identity-store.yaml | sort | uniq)
### setup private keys
while read -r line; do
  file=`echo $line | yq e '.[0]' - `
  mkdir -p `dirname ${file}`
  cp $KEY_FILE ${file}
done < <(yq e '[.. | select(has("keyFile")) | .keyFile]' /configs/identity-store.yaml | sort | uniq)

## resource-aggregate
### setup root-cas
while read -r line; do
  file=`echo $line | yq e '.[0]' - `
  mkdir -p `dirname ${file}`
  cp $CA_POOL ${file}
done < <(yq e '[.. | select(has("caPool")) | .caPool]' /configs/resource-aggregate.yaml | sort | uniq)
### setup certificates
while read -r line; do
  file=`echo $line | yq e '.[0]' - `
  mkdir -p `dirname ${file}`
  cp $CERT_FILE ${file}
done < <(yq e '[.. | select(has("certFile")) | .certFile]' /configs/resource-aggregate.yaml | sort | uniq)
### setup private keys
while read -r line; do
  file=`echo $line | yq e '.[0]' - `
  mkdir -p `dirname ${file}`
  cp $KEY_FILE ${file}
done < <(yq e '[.. | select(has("keyFile")) | .keyFile]' /configs/resource-aggregate.yaml | sort | uniq)

## resource-directory
### setup root-cas
while read -r line; do
  file=`echo $line | yq e '.[0]' - `
  mkdir -p `dirname ${file}`
  cp $CA_POOL ${file}
done < <(yq e '[.. | select(has("caPool")) | .caPool]' /configs/resource-directory.yaml | sort | uniq)
### setup certificates
while read -r line; do
  file=`echo $line | yq e '.[0]' - `
  mkdir -p `dirname ${file}`
  cp $CERT_FILE ${file}
done < <(yq e '[.. | select(has("certFile")) | .certFile]' /configs/resource-directory.yaml | sort | uniq)
### setup private keys
while read -r line; do
  file=`echo $line | yq e '.[0]' - `
  mkdir -p `dirname ${file}`
  cp $KEY_FILE ${file}
done < <(yq e '[.. | select(has("keyFile")) | .keyFile]' /configs/resource-directory.yaml | sort | uniq)

## coap-gateway
### setup root-cas
while read -r line; do
  file=`echo $line | yq e '.[0]' - `
  mkdir -p `dirname ${file}`
  cp $CA_POOL ${file}
done < <(yq e '[.. | select(has("caPool")) | .caPool]' /configs/coap-gateway.yaml | sort | uniq)
### setup certificates
while read -r line; do
  file=`echo $line | yq e '.[0]' - `
  mkdir -p `dirname ${file}`
  cp $CERT_FILE ${file}
done < <(yq e '[.. | select(has("certFile")) | .certFile]' /configs/coap-gateway.yaml | sort | uniq)
### setup private keys
while read -r line; do
  file=`echo $line | yq e '.[0]' - `
  mkdir -p `dirname ${file}`
  cp $KEY_FILE ${file}
done < <(yq e '[.. | select(has("keyFile")) | .keyFile]' /configs/coap-gateway.yaml | sort | uniq)

## grpc-gateway
### setup root-cas
while read -r line; do
  file=`echo $line | yq e '.[0]' - `
  mkdir -p `dirname ${file}`
  cp $CA_POOL ${file}
done < <(yq e '[.. | select(has("caPool")) | .caPool]' /configs/grpc-gateway.yaml | sort | uniq)
### setup certificates
while read -r line; do
  file=`echo $line | yq e '.[0]' - `
  mkdir -p `dirname ${file}`
  cp $CERT_FILE ${file}
done < <(yq e '[.. | select(has("certFile")) | .certFile]' /configs/grpc-gateway.yaml | sort | uniq)
### setup private keys
while read -r line; do
  file=`echo $line | yq e '.[0]' - `
  mkdir -p `dirname ${file}`
  cp $KEY_FILE ${file}
done < <(yq e '[.. | select(has("keyFile")) | .keyFile]' /configs/grpc-gateway.yaml | sort | uniq)

## http-gateway
### setup root-cas
while read -r line; do
  file=`echo $line | yq e '.[0]' - `
  mkdir -p `dirname ${file}`
  cp $CA_POOL ${file}
done < <(yq e '[.. | select(has("caPool")) | .caPool]' /configs/http-gateway.yaml | sort | uniq)
### setup certificates
while read -r line; do
  file=`echo $line | yq e '.[0]' - `
  mkdir -p `dirname ${file}`
  cp $CERT_FILE ${file}
done < <(yq e '[.. | select(has("certFile")) | .certFile]' /configs/http-gateway.yaml | sort | uniq)
### setup private keys
while read -r line; do
  file=`echo $line | yq e '.[0]' - `
  mkdir -p `dirname ${file}`
  cp $KEY_FILE ${file}
done < <(yq e '[.. | select(has("keyFile")) | .keyFile]' /configs/http-gateway.yaml | sort | uniq)

## certificate-authority
### setup root-cas
while read -r line; do
  file=`echo $line | yq e '.[0]' - `
  mkdir -p `dirname ${file}`
  cp $CA_POOL ${file}
done < <(yq e '[.. | select(has("caPool")) | .caPool]' /configs/certificate-authority.yaml | sort | uniq)
### setup certificates
while read -r line; do
  file=`echo $line | yq e '.[0]' - `
  mkdir -p `dirname ${file}`
  cp $CERT_FILE ${file}
done < <(yq e '[.. | select(has("certFile")) | .certFile]' /configs/certificate-authority.yaml | sort | uniq)
### setup private keys
while read -r line; do
  file=`echo $line | yq e '.[0]' - `
  mkdir -p `dirname ${file}`
  cp $KEY_FILE ${file}
done < <(yq e '[.. | select(has("keyFile")) | .keyFile]' /configs/certificate-authority.yaml | sort | uniq)

## cloud2cloud-gateway
### setup root-cas
while read -r line; do
  file=`echo $line | yq e '.[0]' - `
  mkdir -p `dirname ${file}`
  cp $CA_POOL ${file}
done < <(yq e '[.. | select(has("caPool")) | .caPool]' /configs/cloud2cloud-gateway.yaml | sort | uniq)
### setup certificates
while read -r line; do
  file=`echo $line | yq e '.[0]' - `
  mkdir -p `dirname ${file}`
  cp $CERT_FILE ${file}
done < <(yq e '[.. | select(has("certFile")) | .certFile]' /configs/cloud2cloud-gateway.yaml | sort | uniq)
### setup private keys
while read -r line; do
  file=`echo $line | yq e '.[0]' - `
  mkdir -p `dirname ${file}`
  cp $KEY_FILE ${file}
done < <(yq e '[.. | select(has("keyFile")) | .keyFile]' /configs/cloud2cloud-gateway.yaml | sort | uniq)

## cloud2cloud-connector
### setup root-cas
while read -r line; do
  file=`echo $line | yq e '.[0]' - `
  mkdir -p `dirname ${file}`
  cp $CA_POOL ${file}
done < <(yq e '[.. | select(has("caPool")) | .caPool]' /configs/cloud2cloud-connector.yaml | sort | uniq)
### setup certificates
while read -r line; do
  file=`echo $line | yq e '.[0]' - `
  mkdir -p `dirname ${file}`
  cp $CERT_FILE ${file}
done < <(yq e '[.. | select(has("certFile")) | .certFile]' /configs/cloud2cloud-connector.yaml | sort | uniq)
### setup private keys
while read -r line; do
  file=`echo $line | yq e '.[0]' - `
  mkdir -p `dirname ${file}`
  cp $KEY_FILE ${file}
done < <(yq e '[.. | select(has("keyFile")) | .keyFile]' /configs/cloud2cloud-connector.yaml | sort | uniq)



mkdir -p ${OAUTH_KEYS_PATH}
openssl genrsa -out ${OAUTH_ID_TOKEN_KEY_PATH} 4096
openssl ecparam -name prime256v1 -genkey -noout -out ${OAUTH_ACCESS_TOKEN_KEY_PATH}

mkdir -p $MONGO_PATH
mkdir -p $CERTIFICATES_PATH
mkdir -p $LOGS_PATH
mkdir -p ${NGINX_PATH}
cp /nginx/nginx.conf.template ${NGINX_PATH}/nginx.conf
sed -i "s/REPLACE_NGINX_PORT/$NGINX_PORT/g" ${NGINX_PATH}/nginx.conf
sed -i "s/REPLACE_HTTP_GATEWAY_PORT/$HTTP_GATEWAY_PORT/g" ${NGINX_PATH}/nginx.conf
sed -i "s/REPLACE_GRPC_GATEWAY_PORT/$GRPC_GATEWAY_PORT/g" ${NGINX_PATH}/nginx.conf
sed -i "s/REPLACE_MOCK_OAUTH_SERVER_PORT/$MOCK_OAUTH_SERVER_PORT/g" ${NGINX_PATH}/nginx.conf
sed -i "s/REPLACE_CERTIFICATE_AUTHORITY_PORT/$CERTIFICATE_AUTHORITY_PORT/g" ${NGINX_PATH}/nginx.conf
sed -i "s/REPLACE_CLOUD2CLOUD_GATEWAY_PORT/$CLOUD2CLOUD_GATEWAY_PORT/g" ${NGINX_PATH}/nginx.conf
sed -i "s/REPLACE_CLOUD2CLOUD_CONNECTOR_PORT/$CLOUD2CLOUD_CONNECTOR_PORT/g" ${NGINX_PATH}/nginx.conf

# nats
echo "starting nats-server"
cat > /data/nats.config <<EOF
port: $NATS_PORT
max_pending: 128Mb
write_deadline: 10s
tls: {
  verify: true
  cert_file: "$INTERNAL_CERT_DIR_PATH/$GRPC_INTERNAL_CERT_NAME"
  key_file: "$INTERNAL_CERT_DIR_PATH/$GRPC_INTERNAL_CERT_KEY_NAME"
  ca_file: "$CA_POOL"
}
EOF
# Enable jetstream
if [ "${JETSTREAM}" = "true" ]; then
  cat >> /data/nats.config <<EOF
jetstream: {
  store_dir: "$JETSTREAM_PATH"
  // 1GB
    max_memory_store: 1073741824

  // 10GB
  max_file_store: 10737418240
}
EOF
fi
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

if [ "${JETSTREAM}" = "true" ]; then
  echo "Setup streaming at nats"
  nats --tlscert="$INTERNAL_CERT_DIR_PATH/$GRPC_INTERNAL_CERT_NAME" --tlskey="$INTERNAL_CERT_DIR_PATH/$GRPC_INTERNAL_CERT_KEY_NAME" --tlsca="$CA_POOL" str add EVENTS --config /configs/jetstream.json
fi

# mongo
echo "starting mongod"
cat $INTERNAL_CERT_DIR_PATH/$GRPC_INTERNAL_CERT_NAME > $INTERNAL_CERT_DIR_PATH/mongo.key
cat $INTERNAL_CERT_DIR_PATH/$GRPC_INTERNAL_CERT_KEY_NAME >> $INTERNAL_CERT_DIR_PATH/mongo.key
mongod --setParameter maxNumActiveUserIndexBuilds=64 --port $MONGO_PORT --dbpath $MONGO_PATH --sslMode requireSSL --sslCAFile $CA_POOL --sslPEMKeyFile $INTERNAL_CERT_DIR_PATH/mongo.key >$LOGS_PATH/mongod.log 2>&1 &
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
  if openssl s_client -connect ${MONGODB_HOST} -cert ${INTERNAL_CERT_DIR_PATH}/${GRPC_INTERNAL_CERT_NAME} -key ${INTERNAL_CERT_DIR_PATH}/${GRPC_INTERNAL_CERT_KEY_NAME} <<< "Q" 2>/dev/null > /dev/null; then
    break
  fi
  echo "Try to reconnect to mongodb(${MONGODB_HOST}) $i"
  sleep 1
done

# starting nginx
echo "starting nginx"
nginx -c $NGINX_PATH/nginx.conf >$LOGS_PATH/nginx.log 2>&1
status=$?
nginx_pid=$!
if [ $status -ne 0 ]; then
  echo "Failed to start nginx: $status"
  sync
  cat $LOGS_PATH/nginx.log
  exit $status
fi

i=0
while true; do
  i=$((i+1))
  if openssl s_client -connect ${FQDN_NGINX_HTTPS} <<< "Q" 2>/dev/null > /dev/null; then
    break
  fi
  echo "Try to reconnect to nginx(${FQDN_NGINX_HTTPS}) $i"
  sleep 1
done

# oauth-server
echo "starting oauth-server"

## setup cfg
cat /configs/oauth-server.yaml | yq e "\
  .apis.http.address = \"${MOCK_OAUTH_SERVER_ADDRESS}\" |
  .oauthSigner.idTokenKeyFile = \"${OAUTH_ID_TOKEN_KEY_PATH}\" |
  .oauthSigner.accessTokenKeyFile = \"${OAUTH_ACCESS_TOKEN_KEY_PATH}\" |
  .oauthSigner.domain = \"${DOMAIN}\" |
  .oauthSigner.clients[0].id = \"${OAUTH_CLIENT_ID}\" |
  .oauthSigner.clients[0].accessTokenLifetime= \"${MOCK_OAUTH_SERVER_ACCESS_TOKEN_LIFETIME}\"
" - > /data/oauth-server.yaml

oauth-server --config /data/oauth-server.yaml >$LOGS_PATH/oauth-server.log 2>&1 &
status=$?
oauth_server_pid=$!
if [ $status -ne 0 ]; then
  echo "Failed to start oauth-server: $status"
  sync
  cat $LOGS_PATH/oauth-server.log
  exit $status
fi

i=0
while true; do
  i=$((i+1))
  if openssl s_client -connect ${MOCK_OAUTH_SERVER_ADDRESS} -cert ${INTERNAL_CERT_DIR_PATH}/${GRPC_INTERNAL_CERT_NAME} -key ${INTERNAL_CERT_DIR_PATH}/${GRPC_INTERNAL_CERT_KEY_NAME} <<< "Q" 2>/dev/null > /dev/null; then
    break
  fi
  echo "Try to reconnect to oauth-server(${MOCK_OAUTH_SERVER_ADDRESS}) $i"
  sleep 1
done

# identity-store
## configuration
cat /configs/identity-store.yaml | yq e "\
  .apis.grpc.address = \"${IDENTITY_STORE_ADDRESS}\" |
  .apis.grpc.authorization.http.tls.useSystemCAPool = true |
  .apis.grpc.authorization.ownerClaim = \"${OWNER_CLAIM}\" |
  .apis.grpc.authorization.audience = \"${SERVICE_OAUTH_AUDIENCE}\" |
  .apis.grpc.authorization.authority = \"https://${OAUTH_ENDPOINT}\" |
  .clients.storage.mongoDB.uri = \"${MONGODB_URI}\" |
  .clients.eventBus.nats.url = \"${NATS_URL}\" |
  .clients.eventBus.nats.jetstream = ${JETSTREAM}
" - > /data/identity-store.yaml

echo "starting identity-store"
identity-store --config /data/identity-store.yaml >$LOGS_PATH/identity-store.log 2>&1 &
status=$?
identity_store_pid=$!
if [ $status -ne 0 ]; then
  echo "Failed to start identity-store: $status"
  sync
  cat $LOGS_PATH/identity-store.log
  exit $status
fi

i=0
while true; do
  i=$((i+1))
  if openssl s_client -connect ${IDENTITY_STORE_ADDRESS} \
    -cert ${INTERNAL_CERT_DIR_PATH}/${GRPC_INTERNAL_CERT_NAME} \
    -key ${INTERNAL_CERT_DIR_PATH}/${GRPC_INTERNAL_CERT_KEY_NAME} <<< "Q" 2>/dev/null > /dev/null; then
    break
  fi
  echo "Try to reconnect to identity-store(${IDENTITY_STORE_ADDRESS}) $i"
  sleep 1
done

# resource-aggregate
## configuration
cat /configs/resource-aggregate.yaml | yq e "\
  .apis.grpc.address = \"${RESOURCE_AGGREGATE_ADDRESS}\" |
  .apis.grpc.authorization.http.tls.useSystemCAPool = true |
  .apis.grpc.authorization.ownerClaim = \"${OWNER_CLAIM}\" |
  .apis.grpc.authorization.audience = \"${SERVICE_OAUTH_AUDIENCE}\" |
  .apis.grpc.authorization.authority = \"https://${OAUTH_ENDPOINT}\" |
  .clients.eventStore.mongoDB.uri = \"${MONGODB_URI}\" |
  .clients.eventBus.nats.url = \"${NATS_URL}\" |
  .clients.eventBus.nats.jetstream = ${JETSTREAM} |
  .clients.identityStore.grpc.address = \"${IDENTITY_STORE_ADDRESS}\"
" - > /data/resource-aggregate.yaml

echo "starting resource-aggregate"
resource-aggregate --config /data/resource-aggregate.yaml >$LOGS_PATH/resource-aggregate.log 2>&1 &
status=$?
resource_aggregate_pid=$!
if [ $status -ne 0 ]; then
  echo "Failed to start resource-aggregate: $status"
  sync
  cat $LOGS_PATH/resource-aggregate.log
  exit $status
fi

# waiting for resource-aggregate. Without wait, sometimes auth service didn't connect.
i=0
while true; do
  i=$((i+1))
  if openssl s_client -connect ${RESOURCE_AGGREGATE_ADDRESS} -cert ${INTERNAL_CERT_DIR_PATH}/${GRPC_INTERNAL_CERT_NAME} -key ${INTERNAL_CERT_DIR_PATH}/${GRPC_INTERNAL_CERT_KEY_NAME} <<< "Q" 2>/dev/null > /dev/null; then
    break
  fi
  echo "Try to reconnect to resource-aggregate(${RESOURCE_AGGREGATE_ADDRESS}) $i"
  sleep 1
done

# resource-directory
## configuration
cat /configs/resource-directory.yaml | yq e "\
  .apis.grpc.address = \"${RESOURCE_DIRECTORY_ADDRESS}\" |
  .apis.grpc.authorization.ownerClaim = \"${OWNER_CLAIM}\" |
  .apis.grpc.authorization.audience = \"${SERVICE_OAUTH_AUDIENCE}\" |
  .apis.grpc.authorization.http.tls.useSystemCAPool = true |
  .apis.grpc.authorization.authority = \"https://${OAUTH_ENDPOINT}\" |
  .clients.eventStore.mongoDB.uri = \"${MONGODB_URI}\" |
  .clients.eventBus.nats.url = \"${NATS_URL}\" |
  .clients.identityStore.grpc.address = \"${IDENTITY_STORE_ADDRESS}\" |
  .publicConfiguration.authorizationServer = \"https://${OAUTH_ENDPOINT}\" |
  .publicConfiguration.hubID = \"${COAP_GATEWAY_HUB_ID}\" |
  .publicConfiguration.coapGateway = \"coaps+tcp://${COAP_GATEWAY_FQDN}:${COAP_GATEWAY_PORT}\" |
  .publicConfiguration.ownerClaim = \"${OWNER_CLAIM}\"
" - > /data/resource-directory.yaml

echo "starting resource-directory"
resource-directory --config /data/resource-directory.yaml >$LOGS_PATH/resource-directory.log 2>&1 &
status=$?
resource_directory_pid=$!
if [ $status -ne 0 ]; then
  echo "Failed to start resource-directory: $status"
  sync
  cat $LOGS_PATH/resource-directory.log
  exit $status
fi

# waiting for resource-directory. Without wait, sometimes auth service didn't connect.
i=0
while true; do
  i=$((i+1))
  if openssl s_client -connect ${RESOURCE_DIRECTORY_ADDRESS} -cert ${INTERNAL_CERT_DIR_PATH}/${GRPC_INTERNAL_CERT_NAME} -key ${INTERNAL_CERT_DIR_PATH}/${GRPC_INTERNAL_CERT_KEY_NAME} <<< "Q" 2>/dev/null > /dev/null; then
    break
  fi
  echo "Try to reconnect to resource-directory(${RESOURCE_DIRECTORY_ADDRESS}) $i"
  sleep 1
done

# coap-gateway-unsecure
echo "starting coap-gateway-unsecure"
## configuration
cat /configs/coap-gateway.yaml | yq e "\
  .log.debug = ${LOG_DEBUG} |
  .log.dumpCoapMessages = ${COAP_GATEWAY_LOG_MESSAGES} |
  .apis.coap.address = \"${COAP_GATEWAY_UNSECURE_ADDRESS}\" |
  .apis.coap.externalAddress = \"${FQDN}:${COAP_GATEWAY_UNSECURE_PORT}\" |
  .apis.coap.tls.enabled = false |
  .apis.coap.authorization.ownerClaim = \"${OWNER_CLAIM}\" |
  .apis.coap.authorization.providers[0].name = \"${DEVICE_PROVIDER}\" |
  .apis.coap.authorization.providers[0].authority = \"https://${OAUTH_ENDPOINT}\" |
  .apis.coap.authorization.providers[0].clientID = \"${DEVICE_OAUTH_CLIENT_ID}\" |
  .apis.coap.authorization.providers[0].clientSecretFile = \"${OAUTH_DEVICE_SECRET_PATH}\" |
  .apis.coap.authorization.providers[0].redirectURL = \"${DEVICE_OAUTH_REDIRECT_URL}\" |
  .apis.coap.authorization.providers[0].scopes = [ \"${DEVICE_OAUTH_SCOPES}\" ] |
  .apis.coap.authorization.providers[0].audience = \"${DEVICE_OAUTH_AUDIENCE}\" |
  .apis.coap.authorization.providers[0].http.tls.useSystemCAPool = true |
  .apis.coap.authorization.providers[1].name = \"plgd.mobile\" |
  .apis.coap.authorization.providers[1].authority = \"https://${OAUTH_ENDPOINT}\" |
  .apis.coap.authorization.providers[1].clientID = \"${DEVICE_OAUTH_CLIENT_ID}\" |
  .apis.coap.authorization.providers[1].clientSecretFile = \"${OAUTH_DEVICE_SECRET_PATH}\" |
  .apis.coap.authorization.providers[1].redirectURL = \"cloud.plgd.mobile://login-callback\" |
  .apis.coap.authorization.providers[1].http.tls.useSystemCAPool = true |
  .clients.eventBus.nats.url = \"${NATS_URL}\" |
  .clients.identityStore.grpc.address = \"${IDENTITY_STORE_ADDRESS}\" |
  .clients.resourceAggregate.grpc.address = \"${RESOURCE_AGGREGATE_ADDRESS}\" |
  .clients.resourceDirectory.grpc.address = \"${RESOURCE_DIRECTORY_ADDRESS}\"
" - > /data/coap-gateway-unsecure.yaml

if [ "${COAP_GATEWAY_UNSECURE_ENABLED}" = "true" ]; then
  coap-gateway --config /data/coap-gateway-unsecure.yaml >$LOGS_PATH/coap-gateway-unsecure.log 2>&1 &
  status=$?
  coap_gw_unsecure_pid=$!
  if [ $status -ne 0 ]; then
    echo "Failed to start coap-gateway-unsecure: $status"
    sync
    cat $LOGS_PATH/coap-gateway-unsecure.log
    exit $status
  fi
fi

# coap-gateway-secure
echo "starting coap-gateway-secure"
### setup cfgs from env
cat /configs/coap-gateway.yaml | yq e "\
  .log.debug = ${LOG_DEBUG} |
  .log.dumpCoapMessages =  ${COAP_GATEWAY_LOG_MESSAGES} |
  .apis.coap.address = \"${COAP_GATEWAY_ADDRESS}\" |
  .apis.coap.externalAddress = \"${FQDN}:${COAP_GATEWAY_PORT}\" |
  .apis.coap.tls.enabled = true |
  .apis.coap.tls.keyFile = \"${EXTERNAL_CERT_DIR_PATH}/${COAP_GATEWAY_FILE_CERT_KEY_NAME}\" |
  .apis.coap.tls.certFile = \"${EXTERNAL_CERT_DIR_PATH}/${COAP_GATEWAY_FILE_CERT_NAME}\" |
  .apis.coap.tls.clientCertificateRequired = false |
  .apis.coap.authorization.ownerClaim = \"${OWNER_CLAIM}\" |
  .apis.coap.authorization.providers[0].name = \"${DEVICE_PROVIDER}\" |
  .apis.coap.authorization.providers[0].authority = \"https://${OAUTH_ENDPOINT}\" |
  .apis.coap.authorization.providers[0].clientID = \"${DEVICE_OAUTH_CLIENT_ID}\" |
  .apis.coap.authorization.providers[0].clientSecretFile = \"${OAUTH_DEVICE_SECRET_PATH}\" |
  .apis.coap.authorization.providers[0].redirectURL = \"${DEVICE_OAUTH_REDIRECT_URL}\" |
  .apis.coap.authorization.providers[0].scopes = [ \"${DEVICE_OAUTH_SCOPES}\" ] |
  .apis.coap.authorization.providers[0].audience = \"${DEVICE_OAUTH_AUDIENCE}\" |
  .apis.coap.authorization.providers[0].http.tls.useSystemCAPool = true |
  .apis.coap.authorization.providers[1].name = \"plgd.mobile\" |
  .apis.coap.authorization.providers[1].authority = \"https://${OAUTH_ENDPOINT}\" |
  .apis.coap.authorization.providers[1].clientID = \"${DEVICE_OAUTH_CLIENT_ID}\" |
  .apis.coap.authorization.providers[1].clientSecretFile = \"${OAUTH_DEVICE_SECRET_PATH}\" |
  .apis.coap.authorization.providers[1].redirectURL = \"cloud.plgd.mobile://login-callback\" |
  .apis.coap.authorization.providers[1].http.tls.useSystemCAPool = true |
  .clients.eventBus.nats.url = \"${NATS_URL}\" |
  .clients.identityStore.grpc.address = \"${IDENTITY_STORE_ADDRESS}\" |
  .clients.resourceAggregate.grpc.address = \"${RESOURCE_AGGREGATE_ADDRESS}\" |
  .clients.resourceDirectory.grpc.address = \"${RESOURCE_DIRECTORY_ADDRESS}\"
" - > /data/coap-gateway-secure.yaml

coap-gateway --config /data/coap-gateway-secure.yaml >$LOGS_PATH/coap-gateway.log 2>&1 &
status=$?
coap_gw_pid=$!
if [ $status -ne 0 ]; then
  echo "Failed to start coap-gateway: $status"
  sync
  cat $LOGS_PATH/coap-gateway.log
  exit $status
fi

# grpc-gateway
echo "starting grpc-gateway"
## configuration
cat /configs/grpc-gateway.yaml | yq e "\
  .apis.grpc.address = \"${GRPC_GATEWAY_ADDRESS}\" |
  .apis.grpc.authorization.audience = \"${SERVICE_OAUTH_AUDIENCE}\" |
  .apis.grpc.authorization.http.tls.useSystemCAPool = true |
  .apis.grpc.authorization.authority = \"https://${OAUTH_ENDPOINT}\" |
  .clients.identityStore.grpc.address = \"${IDENTITY_STORE_ADDRESS}\" |
  .clients.eventBus.nats.url = \"${NATS_URL}\" |
  .clients.resourceAggregate.grpc.address = \"${RESOURCE_AGGREGATE_ADDRESS}\" |
  .clients.resourceDirectory.grpc.address = \"${RESOURCE_DIRECTORY_ADDRESS}\"
" - > /data/grpc-gateway.yaml
grpc-gateway --config=/data/grpc-gateway.yaml >$LOGS_PATH/grpc-gateway.log 2>&1 &
status=$?
grpc_gw_pid=$!
if [ $status -ne 0 ]; then
  echo "Failed to start grpc-gateway: $status"
  sync
  cat $LOGS_PATH/grpc-gateway.log
  exit $status
fi

# waiting for grpc-gateway. Without wait, sometimes auth service didn't connect.
i=0
while true; do
  i=$((i+1))
  if openssl s_client -connect ${GRPC_GATEWAY_ADDRESS} -cert ${INTERNAL_CERT_DIR_PATH}/${GRPC_INTERNAL_CERT_NAME} -key ${INTERNAL_CERT_DIR_PATH}/${GRPC_INTERNAL_CERT_KEY_NAME} <<< "Q" 2>/dev/null > /dev/null; then
    break
  fi
  echo "Try to reconnect to grpc-gateway(${GRPC_GATEWAY_ADDRESS}) $i"
  sleep 1
done


# http-gateway
## configuration
cat /configs/http-gateway.yaml | yq e "\
  .apis.http.address = \"${HTTP_GATEWAY_ADDRESS}\" |
  .apis.http.authorization.audience = \"${SERVICE_OAUTH_AUDIENCE}\" |
  .apis.http.authorization.http.tls.useSystemCAPool = true |
  .apis.http.authorization.authority = \"https://${OAUTH_ENDPOINT}\" |
  .clients.grpcGateway.grpc.address = \"${GRPC_GATEWAY_ADDRESS}\" |
  .ui.enabled = true |
  .ui.webConfiguration.authority = \"https://${OAUTH_ENDPOINT}\" |
  .ui.webConfiguration.httpGatewayAddress =\"https://${FQDN_NGINX_HTTPS}\" |
  .ui.webConfiguration.webOAuthClient.clientID = \"${OAUTH_CLIENT_ID}\" |
  .ui.webConfiguration.webOAuthClient.audience = \"${OAUTH_AUDIENCE}\" |
  .ui.webConfiguration.webOAuthClient.scopes = [ \"openid\", \"offline_access\" ] |
  .ui.webConfiguration.deviceOAuthClient.clientID = \"${DEVICE_OAUTH_CLIENT_ID}\" |
  .ui.webConfiguration.deviceOAuthClient.scopes = [ \"${DEVICE_OAUTH_SCOPES}\" ] |
  .ui.webConfiguration.deviceOAuthClient.audience = \"${DEVICE_OAUTH_AUDIENCE}\" |
  .ui.webConfiguration.deviceOAuthClient.providerName = \"${DEVICE_PROVIDER}\"
" - > /data/http-gateway.yaml

echo "starting http-gateway"
http-gateway --config=/data/http-gateway.yaml >$LOGS_PATH/http-gateway.log 2>&1 &
status=$?
http_gw_pid=$!
if [ $status -ne 0 ]; then
  echo "Failed to start http-gateway: $status"
  sync
  cat $LOGS_PATH/http-gateway.log
  exit $status
fi

# waiting for http-gateway. Without wait, sometimes auth service didn't connect.
i=0
while true; do
  i=$((i+1))
  if openssl s_client -connect ${HTTP_GATEWAY_ADDRESS} -cert ${INTERNAL_CERT_DIR_PATH}/${GRPC_INTERNAL_CERT_NAME} -key ${INTERNAL_CERT_DIR_PATH}/${GRPC_INTERNAL_CERT_KEY_NAME} <<< "Q" 2>/dev/null > /dev/null; then
    break
  fi
  echo "Try to reconnect to http-gateway(${HTTP_GATEWAY_ADDRESS}) $i"
  sleep 1
done

# certificate-authority
echo "starting certificate-authority"
## configuration
cat /configs/certificate-authority.yaml | yq e "\
  .apis.grpc.address = \"${CERTIFICATE_AUTHORITY_ADDRESS}\" |
  .apis.grpc.authorization.audience = \"${SERVICE_OAUTH_AUDIENCE}\" |
  .apis.grpc.authorization.http.tls.useSystemCAPool = true |
  .apis.grpc.authorization.authority = \"https://${OAUTH_ENDPOINT}\" |
  .apis.grpc.authorization.ownerClaim = \"${OWNER_CLAIM}\" |
  .signer.hubID = \"${COAP_GATEWAY_HUB_ID}\" |
  .signer.keyFile = \"${ROOT_KEY_PATH}\" |
  .signer.certFile = \"${ROOT_CERT_PATH}\"
" - > /data/certificate-authority.yaml
certificate-authority --config /data/certificate-authority.yaml >$LOGS_PATH/certificate-authority.log 2>&1 &
status=$?
certificate_authority_pid=$!
if [ $status -ne 0 ]; then
  echo "Failed to start certificate-authority: $status"
  sync
  cat $LOGS_PATH/certificate-authority.log
  exit $status
fi

# waiting for grpc-gateway. Without wait, sometimes auth service didn't connect.
i=0
while true; do
  i=$((i+1))
  if openssl s_client -connect ${CERTIFICATE_AUTHORITY_ADDRESS} -cert ${INTERNAL_CERT_DIR_PATH}/${GRPC_INTERNAL_CERT_NAME} -key ${INTERNAL_CERT_DIR_PATH}/${GRPC_INTERNAL_CERT_KEY_NAME} <<< "Q" 2>/dev/null > /dev/null; then
    break
  fi
  echo "Try to reconnect to certificate-authority(${CERTIFICATE_AUTHORITY_ADDRESS}) $i"
  sleep 1
done

# cloud2cloud-gateway
## configuration
cat /configs/cloud2cloud-gateway.yaml | yq e "\
  .log.debug = ${LOG_DEBUG} |
  .apis.http.address = \"${CLOUD2CLOUD_GATEWAY_ADDRESS}\" |
  .apis.http.authorization.audience = \"${SERVICE_OAUTH_AUDIENCE}\" |
  .apis.http.authorization.http.tls.useSystemCAPool = true |
  .apis.http.authorization.authority = \"https://${OAUTH_ENDPOINT}\" |
  .clients.eventBus.nats.url = \"${NATS_URL}\" |
  .clients.grpcGateway.grpc.address = \"${GRPC_GATEWAY_ADDRESS}\" |
  .clients.resourceAggregate.grpc.address = \"${RESOURCE_AGGREGATE_ADDRESS}\" |
  .clients.storage.mongoDB.uri = \"${MONGODB_URI}\" |
  .clients.subscription.http.tls.useSystemCAPool = true
" - > /data/cloud2cloud-gateway.yaml

echo "starting cloud2cloud-gateway"
cloud2cloud-gateway --config=/data/cloud2cloud-gateway.yaml >$LOGS_PATH/cloud2cloud-gateway.log 2>&1 &
status=$?
cloud2cloud_gw_pid=$!
if [ $status -ne 0 ]; then
  echo "Failed to start cloud2cloud-gateway: $status"
  sync
  cat $LOGS_PATH/cloud2cloud-gateway.log
  exit $status
fi

# waiting for cloud2cloud-gateway. Without wait, sometimes auth service didn't connect.
i=0
while true; do
  i=$((i+1))
  if openssl s_client -connect ${CLOUD2CLOUD_GATEWAY_ADDRESS} -cert ${INTERNAL_CERT_DIR_PATH}/${GRPC_INTERNAL_CERT_NAME} -key ${INTERNAL_CERT_DIR_PATH}/${GRPC_INTERNAL_CERT_KEY_NAME} <<< "Q" 2>/dev/null > /dev/null; then
    break
  fi
  echo "Try to reconnect to cloud2cloud-gateway(${CLOUD2CLOUD_GATEWAY_ADDRESS}) $i"
  sleep 1
done

# cloud2cloud-connector
## configuration
cat /configs/cloud2cloud-connector.yaml | yq e "\
  .log.debug = ${LOG_DEBUG} |
  .apis.http.address = \"${CLOUD2CLOUD_CONNECTOR_ADDRESS}\" |
  .apis.http.authorization.audience = \"${SERVICE_OAUTH_AUDIENCE}\" |
  .apis.http.authorization.http.tls.useSystemCAPool = true |
  .apis.http.authorization.authority = \"https://${OAUTH_ENDPOINT}\" |
  .apis.http.eventsURL = \"https://${DOMAIN}/c2c/connector/api/v1/events\" |
  .apis.http.authorization.clientID = \"${DEVICE_OAUTH_CLIENT_ID}\" |
  .apis.http.authorization.clientSecretFile = \"${OAUTH_DEVICE_SECRET_PATH}\" |
  .apis.http.authorization.scopes = [ \"${DEVICE_OAUTH_SCOPES}\" ] |
  .apis.http.authorization.audience = \"${DEVICE_OAUTH_AUDIENCE}\" |
  .apis.http.authorization.redirectURL = \"https://${DOMAIN}/c2c/connector/api/v1/oauthCallback\" |
  .apis.http.authorization.http.tls.useSystemCAPool = true |
  .clients.identityStore.grpc.address = \"${IDENTITY_STORE_ADDRESS}\" |
  .clients.eventBus.nats.url = \"${NATS_URL}\" |
  .clients.grpcGateway.grpc.address = \"${GRPC_GATEWAY_ADDRESS}\" |
  .clients.resourceAggregate.grpc.address = \"${RESOURCE_AGGREGATE_ADDRESS}\" |
  .clients.storage.mongoDB.uri = \"${MONGODB_URI}\"
" - > /data/cloud2cloud-connector.yaml

echo "starting cloud2cloud-connector"
cloud2cloud-connector --config=/data/cloud2cloud-connector.yaml >$LOGS_PATH/cloud2cloud-connector.log 2>&1 &
status=$?
cloud2cloud_connector_pid=$!
if [ $status -ne 0 ]; then
  echo "Failed to start cloud2cloud-connector: $status"
  sync
  cat $LOGS_PATH/cloud2cloud-connector.log
  exit $status
fi

# waiting for cloud2cloud-connector. Without wait, sometimes auth service didn't connect.
i=0
while true; do
  i=$((i+1))
  if openssl s_client -connect ${CLOUD2CLOUD_CONNECTOR_ADDRESS} -cert ${INTERNAL_CERT_DIR_PATH}/${GRPC_INTERNAL_CERT_NAME} -key ${INTERNAL_CERT_DIR_PATH}/${GRPC_INTERNAL_CERT_KEY_NAME} <<< "Q" 2>/dev/null > /dev/null; then
    break
  fi
  echo "Try to reconnect to cloud2cloud-connector(${CLOUD2CLOUD_CONNECTOR_ADDRESS}) $i"
  sleep 1
done



echo "Open browser at https://${DOMAIN}"

# Naive check runs checks once a minute to see if either of the processes exited.
# This illustrates part of the heavy lifting you need to do if you want to run
# more than one service in a container. The container exits with an error
# if it detects that either of the processes has exited.
# Otherwise it loops forever, waking up every 60 seconds
while sleep 10; do
  ps aux |grep $nats_server_pid |grep -q -v grep
  if [ $? -ne 0 ]; then
    echo "nats-server has already exited."
    sync
    cat $LOGS_PATH/nats-server.log
    exit 1
  fi
  ps aux |grep $mongo_pid |grep -q -v grep
  if [ $? -ne 0 ]; then
    echo "mongod has already exited."
    sync
    cat $LOGS_PATH/mongod.log
    exit 1
  fi
  ps aux |grep $identity_store_pid |grep -q -v grep
  if [ $? -ne 0 ]; then
    echo "identity-store has already exited."
    sync
    cat $LOGS_PATH/identity-store.log
    exit 1
  fi
  ps aux |grep $resource_aggregate_pid |grep -q -v grep
  if [ $? -ne 0 ]; then
    echo "resource-aggregate has already exited."
    sync
    cat $LOGS_PATH/resource-aggregate.log
    exit 1
  fi
  ps aux |grep $resource_directory_pid |grep -q -v grep
  if [ $? -ne 0 ]; then
    echo "resource-directory has already exited."
    sync
    cat $LOGS_PATH/resource-directory.log
    exit 1
  fi
  if  [ ! -z "${coap_gw_unsecure_pid}" ]; then
    ps aux |grep $coap_gw_unsecure_pid |grep -q -v grep
    if [ $? -ne 0 ]; then
      echo "coap-gateway-unsecure has already exited."
      sync
      cat $LOGS_PATH/coap-gateway-unsecure.log
      exit 1
    fi
  fi
  ps aux |grep $coap_gw_pid |grep -q -v grep
  if [ $? -ne 0 ]; then
    echo "coap-gateway has already exited."
    sync
    cat $LOGS_PATH/coap-gateway.log
    exit 1
  fi
  ps aux |grep $grpc_gw_pid |grep -q -v grep
  if [ $? -ne 0 ]; then
    echo "grpc-gateway has already exited."
    sync
    cat $LOGS_PATH/grpc-gateway.log
   exit 1
  fi
  ps aux |grep $http_gw_pid |grep -q -v grep
  if [ $? -ne 0 ]; then
    echo "http-gateway has already exited."
    sync
    cat $LOGS_PATH/http-gateway.log
   exit 1
  fi
  ps aux |grep $certificate_authority_pid |grep -q -v grep
  if [ $? -ne 0 ]; then
    echo "certificate-authority has already exited."
    sync
    cat $LOGS_PATH/certificate-authority.log
   exit 1
  fi
  ps aux |grep $oauth_server_pid |grep -q -v grep
  if [ $? -ne 0 ]; then
    echo "oauth-server has already exited."
    sync
    cat $LOGS_PATH/oauth-server.log
   exit 1
  fi
  ps aux |grep $nginx_pid |grep -q -v grep
  if [ $? -ne 0 ]; then
    echo "nginx has already exited."
    sync
    cat $LOGS_PATH/nginx.log
    exit 1
  fi
  ps aux |grep $cloud2cloud_gw_pid |grep -q -v grep
  if [ $? -ne 0 ]; then
    echo "cloud2cloud-gateway has already exited."
    sync
    cat $LOGS_PATH/cloud2cloud-gateway.log
   exit 1
  fi
  ps aux |grep $cloud2cloud_connector_pid |grep -q -v grep
  if [ $? -ne 0 ]; then
    echo "cloud2cloud_connector has already exited."
    sync
    cat $LOGS_PATH/cloud2cloud_connector.log
   exit 1
  fi
done
