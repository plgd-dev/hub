#!/usr/bin/env bash
set -e

# Configure services
export PATH="/usr/local/bin:$PATH"

# INTERNAL CERTS
export INTERNAL_CERT_DIR_PATH="$CERITIFICATES_PATH"
export INTERNAL_CERT_NAME="$INTERNAL_CERT_DIR_PATH/http.crt"
export INTERNAL_KEY_NAME="$INTERNAL_CERT_DIR_PATH/http.key"

# EXTERNAL CERTS
export COAP_GATEWAY_CERT_DIR_PATH="$CERITIFICATES_PATH"
export COAP_GATEWAY_CERT_NAME="$COAP_GATEWAY_CERT_DIR_PATH/coap.crt"
export COAP_GATEWAY_KEY_NAME="$COAP_GATEWAY_CERT_DIR_PATH/coap.key"

# MONGO CERTS
export MONGO_CERT_DIR_PATH="$CERITIFICATES_PATH"
export MONGO_CERT_NAME="$MONGO_CERT_DIR_PATH/mongo.crt"
export MONGO_KEY_NAME="$MONGO_CERT_DIR_PATH/mongo.key"
export MONGO_CERT_KEY_NAME="$MONGO_CERT_DIR_PATH/mongo.certkey"

# ROOT CERTS
export CA_POOL_DIR="$CERITIFICATES_PATH"
export CA_POOL_NAME_PREFIX="root_ca"
export CA_POOL_CERT_PATH="$CA_POOL_DIR/$CA_POOL_NAME_PREFIX.crt"
export CA_POOL_KEY_PATH="$CA_POOL_DIR/$CA_POOL_NAME_PREFIX.key"

mkdir -p $MONGO_PATH
mkdir -p $CERITIFICATES_PATH
mkdir -p $LOGS_PATH

if [ "$INITIALIZE_CERITIFICATES" = "true" ]; then
  fqdnSAN="--cert.san.domain=$FQDN"
  if ip route get $FQDN 2>/dev/null >/dev/null; then
    fqdnSAN="--cert.san.ip=$FQDN"
  fi
  echo "generating CA cert"
  certificate-generator --cmd.generateRootCA --outCert=$CA_POOL_CERT_PATH --outKey=$CA_POOL_KEY_PATH --cert.subject.cn="Root CA"
  echo "generating GRPC internal cert"
  certificate-generator --cmd.generateCertificate --outCert=$INTERNAL_CERT_NAME --outKey=$INTERNAL_KEY_NAME --cert.subject.cn="localhost" --cert.san.domain="localhost" --cert.san.ip="0.0.0.0" --cert.san.ip="127.0.0.1" $fqdnSAN --signerCert=$CA_POOL_CERT_PATH --signerKey=$CA_POOL_KEY_PATH
  echo "generating COAP-GW cert"
  certificate-generator --cmd.generateIdentityCertificate=$COAP_GATEWAY_CLOUD_ID --outCert=$COAP_GATEWAY_CERT_NAME --outKey=$COAP_GATEWAY_KEY_NAME --cert.san.domain="localhost" --cert.san.ip="127.0.0.1" --cert.san.domain=$COAP_GATEWAY_FQDN --signerCert=$CA_POOL_CERT_PATH --signerKey=$CA_POOL_KEY_PATH
fi

# nats
echo "starting nats-server"
nats-server --tls --tlsverify --tlscert=$INTERNAL_CERT_NAME --tlskey=$INTERNAL_KEY_NAME --tlscacert=$CA_POOL_CERT_PATH >$LOGS_PATH/nats-server.log 2>&1 &
status=$?
nats_server_pid=$!
if [ $status -ne 0 ]; then
  echo "Failed to start nats-server: $status"
  sync
  cat $LOGS_PATH/nats-server.log
  exit $status
fi

# mongo
echo "starting mongod"
cp $INTERNAL_CERT_NAME $MONGO_CERT_NAME
cp $INTERNAL_KEY_NAME $MONGO_KEY_NAME
cat $MONGO_CERT_NAME > $MONGO_CERT_KEY_NAME
cat $MONGO_KEY_NAME >> $MONGO_CERT_KEY_NAME
mongod --dbpath $MONGO_PATH --sslMode requireSSL --sslCAFile $CA_POOL_CERT_PATH --sslPEMKeyFile $MONGO_CERT_KEY_NAME >$LOGS_PATH/mongod.log 2>&1 &
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
  if openssl s_client -connect ${MONGODB_HOST} <<< "Q" 2>/dev/null > /dev/null; then
    break
  fi
  echo "Try to reconnect to mongodb(${MONGODB_HOST}) $i"
  sleep 1
done
    
# authorization
echo "starting authorization"
authorization --config=/data/yaml/authorization.yaml >$LOGS_PATH/authorization.log 2>&1 &
status=$?
authorization_pid=$!
if [ $status -ne 0 ]; then
  echo "Failed to start authorization: $status"
  sync
  cat $LOGS_PATH/authorization.log
  exit $status
fi

i=0
while true; do
  i=$((i+1))
  if openssl s_client -connect ${AUTHORIZATION_HTTP_ADDRESS} <<< "Q" 2>/dev/null > /dev/null; then
    break
  fi
  echo "Try to reconnect to authorization service(${AUTHORIZATION_HTTP_ADDRESS}) $i"
  sleep 1
done

# resource-aggregate
echo "starting resource-aggregate"
resource-aggregate --config=/data/yaml/resource-aggregate.yaml >$LOGS_PATH/resource-aggregate.log 2>&1 &
status=$?
resource_aggregate_pid=$!
if [ $status -ne 0 ]; then
  echo "Failed to start resource-aggregate: $status"
  sync
  cat $LOGS_PATH/resource-aggregate.log
  exit $status
fi

# resource-directory
echo "starting resource-directory"
resource-directory --config=/data/yaml/resource-directory.yaml >$LOGS_PATH/resource-directory.log 2>&1 &
status=$?
resource_directory_pid=$!
if [ $status -ne 0 ]; then
  echo "Failed to start resource-directory: $status"
  sync
  cat $LOGS_PATH/resource-directory.log
  exit $status
fi

# coap-gateway-unsecure
echo "starting coap-gateway-unsecure"
coap-gateway --config=/data/yaml/coap-gateway-unsecure.yaml >$LOGS_PATH/coap-gateway-unsecure.log 2>&1 &
status=$?
coap_gw_unsecure_pid=$!
if [ $status -ne 0 ]; then
  echo "Failed to start coap-gateway-unsecure: $status"
  sync
  cat $LOGS_PATH/coap-gateway-unsecure.log
  exit $status
fi

# coap-gateway-secure
echo "starting coap-gateway-secure"
coap-gateway --config=/data/yaml/coap-gateway.yaml >$LOGS_PATH/coap-gateway.log 2>&1 &
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
grpc-gateway --config=/data/yaml/grpc-gateway.yaml >$LOGS_PATH/grpc-gateway.log 2>&1 &
status=$?
grpc_gw_pid=$!
if [ $status -ne 0 ]; then
  echo "Failed to start grpc-gateway: $status"
  sync
  cat $LOGS_PATH/grpc-gateway.log
  exit $status
fi

# http-gateway
echo "starting http-gateway"
http-gateway --config=/data/yaml/http-gateway.yaml >$LOGS_PATH/http-gateway.log 2>&1 &
status=$?
http_gw_pid=$!
if [ $status -ne 0 ]; then
  echo "Failed to start http-gateway: $status"
  sync
  cat $LOGS_PATH/http-gateway.log
  exit $status
fi

# certificate-authority
echo "starting certificate-authority"
certificate-authority --config=/data/yaml/certificate-authority.yaml >$LOGS_PATH/certificate-authority.log 2>&1 &
status=$?
certificate_authority_pid=$!
if [ $status -ne 0 ]; then
  echo "Failed to start certificate-authority: $status"
  sync
  cat $LOGS_PATH/certificate-authority.log
  exit $status
fi

# cloud2cloud-connector
echo "starting cloud2cloud-connector"
cloud2cloud-connector --config=/data/yaml/c2c-connector.yaml >$LOGS_PATH/cloud2cloud-connector.log 2>&1 &
status=$?
cloud2cloud_connector_pid=$!
if [ $status -ne 0 ]; then
  echo "Failed to start cloud2cloud-connector: $status"
  sync
  cat $LOGS_PATH/cloud2cloud-connector.log
  exit $status
fi

# cloud2cloud-gateway
echo "starting cloud2cloud-gateway"
cloud2cloud-connector --config=/data/yaml/c2c-gateway.yaml >$LOGS_PATH/cloud2cloud-gateway.log 2>&1 &
status=$?
cloud2cloud_gateway_pid=$!
if [ $status -ne 0 ]; then
  echo "Failed to start cloud2cloud-connector: $status"
  sync
  cat $LOGS_PATH/cloud2cloud-gateway.log
  exit $status
fi

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
  ps aux |grep $authorization_pid |grep -q -v grep
  if [ $? -ne 0 ]; then 
    echo "authorization has already exited."
    sync
    cat $LOGS_PATH/authorization.log
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
  ps aux |grep $cloud2cloud_connector_pid |grep -q -v grep
  if [ $? -ne 0 ]; then
    echo "cloud2cloud-connector has already exited."
    sync
    cat $LOGS_PATH/cloud2cloud-connector.log
   exit 1
  fi
  ps aux |grep $cloud2cloud_gateway_pid |grep -q -v grep
  if [ $? -ne 0 ]; then
    echo "cloud2cloud-gateway has already exited."
    sync
    cat $LOGS_PATH/cloud2cloud-gateway.log
   exit 1
  fi
done