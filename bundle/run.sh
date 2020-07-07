#!/usr/bin/env bash
set -e

# Configure services
export PATH="/usr/local/bin:$PATH"

export INTERNAL_CERT_DIR_PATH="$CERITIFICATES_PATH/internal"
export GRPC_INTERNAL_CERT_NAME="endpoint.crt"
export GRPC_INTERNAL_CERT_KEY_NAME="endpoint.key"

export EXTERNAL_CERT_DIR_PATH="$CERITIFICATES_PATH/external"
export COAP_GATEWAY_FILE_CERT_NAME="coap-gateway.crt"
export COAP_GATEWAY_FILE_CERT_KEY_NAME="coap-gateway.key"

# ROOT CERTS
export CA_POOL_DIR="$CERITIFICATES_PATH"
export CA_POOL_NAME_PREFIX="root_ca"
export CA_POOL_CERT_PATH="$CA_POOL_DIR/$CA_POOL_NAME_PREFIX.crt"
export CA_POOL_CERT_KEY_PATH="$CA_POOL_DIR/$CA_POOL_NAME_PREFIX.key"

# DIAL CERTS
export DIAL_TYPE="file"
export DIAL_FILE_CA_POOL="$CA_POOL_CERT_PATH"
export DIAL_FILE_CERT_DIR_PATH="$INTERNAL_CERT_DIR_PATH"
export DIAL_FILE_CERT_NAME="$GRPC_INTERNAL_CERT_NAME"
export DIAL_FILE_CERT_KEY_NAME="$GRPC_INTERNAL_CERT_KEY_NAME"

#LISTEN CERTS
export LISTEN_TYPE="file"
export LISTEN_FILE_CA_POOL="$CA_POOL_CERT_PATH"
export LISTEN_FILE_CERT_DIR_PATH="$INTERNAL_CERT_DIR_PATH"
export LISTEN_FILE_CERT_NAME="$GRPC_INTERNAL_CERT_NAME"
export LISTEN_FILE_CERT_KEY_NAME="$GRPC_INTERNAL_CERT_KEY_NAME"

export MONGODB_URL="mongodb://localhost:$MONGO_PORT"
export MONGODB_URI="mongodb://localhost:$MONGO_PORT"
export MONGO_URI="mongodb://localhost:$MONGO_PORT"

export NATS_URL="nats://localhost:$NATS_PORT"

export AUTH_SERVER_ADDRESS=${AUTHORIZATION_ADDRESS}
export OAUTH_ENDPOINT_TOKEN_URL=https://${AUTHORIZATION_HTTP_ADDRESS}/api/authz/token
export SERVICE_OAUTH_ENDPOINT_TOKEN_URL=${OAUTH_ENDPOINT_TOKEN_URL}

if [ "$INITIALIZE_CERITIFICATES" = "true" ]; then
  mkdir -p $CA_POOL_DIR
  mkdir -p $INTERNAL_CERT_DIR_PATH
  mkdir -p $EXTERNAL_CERT_DIR_PATH
  echo "generating CA cert"
  certificate-generator --cmd.generateRootCA --outCert=$CA_POOL_CERT_PATH --outKey=$CA_POOL_CERT_KEY_PATH --cert.subject.cn="Root CA"
  echo "generating GRPC internal cert"
  certificate-generator --cmd.generateCertificate --outCert=$DIAL_FILE_CERT_DIR_PATH/$DIAL_FILE_CERT_NAME --outKey=$DIAL_FILE_CERT_DIR_PATH/$DIAL_FILE_CERT_KEY_NAME --cert.subject.cn="localhost" --cert.san.domain="localhost" --signerCert=$CA_POOL_CERT_PATH --signerKey=$CA_POOL_CERT_KEY_PATH
  echo "generating COAP-GW cert"
  certificate-generator --cmd.generateIdentityCertificate=$COAP_GATEWAY_CLOUD_ID --outCert=$EXTERNAL_CERT_DIR_PATH/$COAP_GATEWAY_FILE_CERT_NAME --outKey=$EXTERNAL_CERT_DIR_PATH/$COAP_GATEWAY_FILE_CERT_KEY_NAME --cert.san.domain=$COAP_GATEWAY_FQDN --signerCert=$CA_POOL_CERT_PATH --signerKey=$CA_POOL_CERT_KEY_PATH
fi

mkdir -p $MONGO_PATH
mkdir -p $CERITIFICATES_PATH
mkdir -p $LOGS_PATH

# nats
nats-server --port $NATS_PORT --tls --tlsverify --tlscert=$DIAL_FILE_CERT_DIR_PATH/$DIAL_FILE_CERT_NAME --tlskey=$DIAL_FILE_CERT_DIR_PATH/$DIAL_FILE_CERT_KEY_NAME --tlscacert=$CA_POOL_CERT_PATH >$LOGS_PATH/nats-server.log 2>&1 &
status=$?
if [ $status -ne 0 ]; then
  echo "Failed to start nats-server: $status"
  sync
  cat $LOGS_PATH/nats-server.log
  exit $status
fi

# mongo
cat $DIAL_FILE_CERT_DIR_PATH/$DIAL_FILE_CERT_NAME > $DIAL_FILE_CERT_DIR_PATH/mongo.key
cat $DIAL_FILE_CERT_DIR_PATH/$DIAL_FILE_CERT_KEY_NAME >> $DIAL_FILE_CERT_DIR_PATH/mongo.key
mongod --port $MONGO_PORT --dbpath $MONGO_PATH --sslMode requireSSL --sslCAFile $CA_POOL_CERT_PATH --sslPEMKeyFile $DIAL_FILE_CERT_DIR_PATH/mongo.key >$LOGS_PATH/mongod.log 2>&1 &
status=$?
if [ $status -ne 0 ]; then
  echo "Failed to start mongod: $status"
  sync
  cat $LOGS_PATH/mongod.log
  exit $status
fi
    
# authorization
echo "starting authorization"
ADDRESS=${AUTHORIZATION_ADDRESS} HTTP_ADDRESS=${AUTHORIZATION_HTTP_ADDRESS} DEVICE_PROVIDER="test" authorization >$LOGS_PATH/authorization.log 2>&1 &
status=$?
if [ $status -ne 0 ]; then
  echo "Failed to start authorization: $status"
  sync
  cat $LOGS_PATH/authorization.log
  exit $status
fi

for ((i=0;i<10;i++)); do
  if curl -s -k ${OAUTH_ENDPOINT_TOKEN_URL} > /dev/null; then
    break
  fi
  echo "Retry connect to ${OAUTH_ENDPOINT_TOKEN_URL} $((i+1))/10"
  sleep 1
done

# resource-aggregate
echo "starting resource-aggregate"
ADDRESS=${RESOURCE_AGGREGATE_ADDRESS} resource-aggregate >$LOGS_PATH/resource-aggregate.log 2>&1 &
status=$?
if [ $status -ne 0 ]; then
  echo "Failed to start resource-aggregate: $status"
  sync
  cat $LOGS_PATH/resource-aggregate.log
  exit $status
fi

# resource-directory
echo "starting resource-directory"
ADDRESS=${RESOURCE_DIRECTORY_ADDRESS} resource-directory >$LOGS_PATH/resource-directory.log 2>&1 &
status=$?
if [ $status -ne 0 ]; then
  echo "Failed to start resource-directory: $status"
  sync
  cat $LOGS_PATH/resource-directory.log
  exit $status
fi

# coap-gateway-unsecure
echo "starting coap-gateway-unsecure"
ADDRESS=${COAP_GATEWAY_UNSECURE_ADDRESS} \
LOG_ENABLE_DEBUG=true \
EXTERNAL_PORT=${COAP_GATEWAY_UNSECURE_PORT} \
FQDN=${COAP_GATEWAY_UNSECURE_FQDN} \
DISABLE_BLOCKWISE_TRANSFER=${COAP_GATEWAY_DISABLE_BLOCKWISE_TRANSFER} \
BLOCKWISE_TRANSFER_SZX=${COAP_GATEWAY_BLOCKWISE_TRANSFER_SZX} \
DISABLE_PEER_TCP_SIGNAL_MESSAGE_CSMS=${COAP_GATEWAY_DISABLE_PEER_TCP_SIGNAL_MESSAGE_CSMS} \
LISTEN_WITHOUT_TLS="true" \
coap-gateway >$LOGS_PATH/coap-gateway-unsecure.log 2>&1 &
status=$?
if [ $status -ne 0 ]; then
  echo "Failed to start coap-gateway-unsecure: $status"
  sync
  cat $LOGS_PATH/coap-gateway-unsecure.log
  exit $status
fi
coap_gw_unsecure_pid=`ps -ef | grep coap-gateway | grep -v grep | awk '{print $2}'`
if [ "$coap_gw_unsecure_pid" = "" ]; then
  echo "Failed to get pid coap-gateway-unsecured"
  sync
  cat $LOGS_PATH/coap-gateway-unsecure.log
  exit 1
fi

# coap-gateway-secure
echo "starting coap-gateway-secure"
ADDRESS=${COAP_GATEWAY_ADDRESS} \
LOG_ENABLE_DEBUG=true \
EXTERNAL_PORT=${COAP_GATEWAY_PORT} \
FQDN=${COAP_GATEWAY_FQDN} \
LISTEN_FILE_CERT_DIR_PATH=${EXTERNAL_CERT_DIR_PATH} \
LISTEN_FILE_CERT_NAME=${COAP_GATEWAY_FILE_CERT_NAME} \
LISTEN_FILE_CERT_KEY_NAME=${COAP_GATEWAY_FILE_CERT_KEY_NAME} \
LISTEN_FILE_DISABLE_VERIFY_CLIENT_CERTIFICATE=${COAP_GATEWAY_DISABLE_VERIFY_CLIENTS} \
DISABLE_BLOCKWISE_TRANSFER=${COAP_GATEWAY_DISABLE_BLOCKWISE_TRANSFER} \
BLOCKWISE_TRANSFER_SZX=${COAP_GATEWAY_BLOCKWISE_TRANSFER_SZX} \
DISABLE_PEER_TCP_SIGNAL_MESSAGE_CSMS=${COAP_GATEWAY_DISABLE_PEER_TCP_SIGNAL_MESSAGE_CSMS} \
coap-gateway >$LOGS_PATH/coap-gateway.log 2>&1 &
status=$?
if [ $status -ne 0 ]; then
  echo "Failed to start coap-gateway: $status"
  sync
  cat $LOGS_PATH/coap-gateway.log
  exit $status
fi
coap_gw_pid=`ps -ef | grep coap-gateway | grep -v grep | grep -v $coap_gw_unsecure_pid | awk '{print $2}'`
if [ "$coap_gw_pid" = "" ]; then
  echo "Failed to get pid coap-gateway"
  sync
  cat $LOGS_PATH/coap-gateway.log
  exit 1
fi

# grpc-gateway
echo "starting grpc-gateway"
ADDRESS=${GRPC_GATEWAY_ADDRESS} \
LOG_ENABLE_DEBUG=true \
LISTEN_FILE_DISABLE_VERIFY_CLIENT_CERTIFICATE=${GRPC_GATEWAY_DISABLE_VERIFY_CLIENTS} \
grpc-gateway >$LOGS_PATH/grpc-gateway.log 2>&1 &
status=$?
if [ $status -ne 0 ]; then
  echo "Failed to start grpc-gateway: $status"
  sync
  cat $LOGS_PATH/grpc-gateway.log
  exit $status
fi


# Naive check runs checks once a minute to see if either of the processes exited.
# This illustrates part of the heavy lifting you need to do if you want to run
# more than one service in a container. The container exits with an error
# if it detects that either of the processes has exited.
# Otherwise it loops forever, waking up every 60 seconds
while sleep 10; do
  ps aux |grep nats-server |grep -q -v grep
  if [ $? -ne 0 ]; then 
    echo "nats-server has already exited."
    sync
    cat $LOGS_PATH/nats-server.log
    exit 1
  fi
  ps aux |grep mongod |grep -q -v grep
  if [ $? -ne 0 ]; then 
    echo "mongod has already exited."
    sync
    cat $LOGS_PATH/mongod.log
    exit 1
  fi
  ps aux |grep authorization |grep -q -v grep
  if [ $? -ne 0 ]; then 
    echo "authorization has already exited."
    sync
    cat $LOGS_PATH/authorization.log
    exit 1
  fi
  ps aux |grep resource-aggregate |grep -q -v grep
  if [ $? -ne 0 ]; then 
    echo "resource-aggregate has already exited."
    sync
    cat $LOGS_PATH/resource-aggregate.log
    exit 1
  fi
  ps aux |grep resource-directory |grep -q -v grep
  if [ $? -ne 0 ]; then 
    echo "resource-directory has already exited."
    sync
    cat $LOGS_PATH/resource-directory.log
    exit 1
  fi
  ps aux |grep $coap_gw_unsecure_pid |grep -q -v grep
  if [ $? -ne 0 ]; then 
    echo "coap-gateway-unsecure has already exited."
    sync
    cat $LOGS_PATH/coap-gateway-unsecure.log
    exit 1
  fi
  ps aux |grep $coap_gw_pid |grep -q -v grep
  if [ $? -ne 0 ]; then 
    echo "coap-gateway has already exited."
    sync
    cat $LOGS_PATH/coap-gateway.log
    exit 1
  fi
  ps aux |grep grpc-gateway |grep -q -v grep
  if [ $? -ne 0 ]; then 
    echo "grpc-gateway has already exited."
    sync
    cat $LOGS_PATH/grpc-gateway.log
   exit 1
  fi
done