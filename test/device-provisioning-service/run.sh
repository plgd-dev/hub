#!/usr/bin/env bash
set -e

umask 0000

# Configure services
export PATH="/usr/local/bin:$PATH"

COAP_GATEWAY_HUB_ID="adebc667-1f2b-41e3-bf5c-6d6eabc68cc6"

CERTIFICATES_PATH="/data/certs"
DEVICE_CERTIFICATES_PATH="/data/certs/device"
OAUTH_KEYS_PATH="/data/privKeys"
LOGS_PATH="/data/log"
MONGO_PATH="/data/db"

# CERTS
CA_POOL_DIR="$CERTIFICATES_PATH"
CA_POOL_CERT_NAME="root_ca.crt"
CA_POOL_CERT_KEY_NAME="root_ca.key"
CA_POOL_CERT_PATH="${CERTIFICATES_PATH}/${CA_POOL_CERT_NAME}"
CA_POOL_CERT_KEY_PATH="${CERTIFICATES_PATH}/${CA_POOL_CERT_KEY_NAME}"
CA_POOL_CERT_ALT_NAME="root_ca_alt.crt"
CA_POOL_CERT_KEY_ALT_NAME="root_ca_alt.key"
CA_POOL_CERT_ALT_PATH="${CERTIFICATES_PATH}/${CA_POOL_CERT_ALT_NAME}"
CA_POOL_CERT_KEY_ALT_PATH="${CERTIFICATES_PATH}/${CA_POOL_CERT_KEY_ALT_NAME}"
CA_POOL_CERT_VALID_FROM="2000-01-01T12:00:00Z"
CA_POOL_CERT_VALID_FOR="876000h"

COAP_GATEWAY_CERT_NAME="coap.crt"
COAP_GATEWAY_CERT_KEY_NAME="coap.key"
COAP_GATEWAY_CERT="${CERTIFICATES_PATH}/${COAP_GATEWAY_CERT_NAME}"
COAP_GATEWAY_CERT_KEY="${CERTIFICATES_PATH}/${COAP_GATEWAY_CERT_KEY_NAME}"
HTTP_CERT_NAME="http.crt"
HTTP_CERT_KEY_NAME="http.key"
HTTP_CERT="${CERTIFICATES_PATH}/${HTTP_CERT_NAME}"
HTTP_CERT_KEY="${CERTIFICATES_PATH}/${HTTP_CERT_KEY_NAME}"
MONGODB_CERT_KEY="${CERTIFICATES_PATH}/mongo.key"

DPS_CA_CERT="${DEVICE_CERTIFICATES_PATH}/dpsca.pem"
DPS_CA_CERT_KEY="${DEVICE_CERTIFICATES_PATH}/dpscakey.pem"
DPS_INTERMEDIATECA_CERT="${DEVICE_CERTIFICATES_PATH}/intermediatecacrt.pem"
DPS_INTERMEDIATECA_CERT_KEY="${DEVICE_CERTIFICATES_PATH}/intermediatecakey.pem"
DPS_MFG_CERT="${DEVICE_CERTIFICATES_PATH}/mfgcrt.pem"
DPS_MFG_CERT_KEY="${DEVICE_CERTIFICATES_PATH}/mfgkey.pem"

# LISTEN CERTS
export LISTEN_FILE_CA_POOL="${CA_POOL_CERT_PATH}"
export LISTEN_FILE_CERT_DIR_PATH="${CERTIFICATES_PATH}"
export LISTEN_FILE_CERT_NAME="${HTTP_CERT_NAME}"
export LISTEN_FILE_CERT_KEY_NAME="${HTTP_CERT_KEY_NAME}"
LISTEN_FILE_CERT="${LISTEN_FILE_CERT_DIR_PATH}/${LISTEN_FILE_CERT_NAME}"
LISTEN_FILE_CERT_KEY="${LISTEN_FILE_CERT_DIR_PATH}/${LISTEN_FILE_CERT_KEY_NAME}"

CERT_TOOL_SIGN_ALG=${CERT_TOOL_SIGN_ALG:-ECDSA-SHA256}
CERT_TOOL_ELLIPTIC_CURVE=${CERT_TOOL_ELLIPTIC_CURVE:-P256}

function startMongo {
	ID=$1
	PORT=$2
	REPLICA_SET=$3
	echo "starting mongod ${ID}"
	HOST=localhost:${PORT}
	DB_PATH="${MONGO_PATH}/${ID}"
	mkdir -p ${DB_PATH}
	mongod --setParameter maxNumActiveUserIndexBuilds=64 \
		--port "${PORT}" \
		--dbpath "${DB_PATH}" \
		--tlsMode requireTLS \
		--tlsCAFile "${CA_POOL_CERT_PATH}" \
		--replSet ${REPLICA_SET} \
		--bind_ip localhost \
		--tlsCertificateKeyFile "${MONGODB_CERT_KEY}" >"${LOGS_PATH}/mongod.$ID.log" 2>&1 &
	status=$?
	mongo_pid=$!
	if [ $status -ne 0 ]; then
		echo "Failed to start mongod: ${status}"
		sync
		cat "${LOGS_PATH}/mongod.$ID.log"
		exit ${status}
	fi

	# waiting for mongo DB. Without wait, sometimes auth service didn't connect.
	i=0
	while [ $i -le 20 ]; do
		i=$((i+1))
		if openssl s_client -connect "${HOST}" -cert "${LISTEN_FILE_CERT}" -key "${LISTEN_FILE_CERT_KEY}" <<< "Q" 2>/dev/null > /dev/null; then
			break
		fi
		if [ $i -eq 20 ]; then
			echo "Failed to connect to mongodb(${HOST})"
			exit 1
		fi
		echo "Try to reconnect to mongodb(${HOST}) $i"
		sleep 1
	done
}

if [ "${PREPARE_ENV}" = "true" ]; then
	mkdir -p "${CERTIFICATES_PATH}"
	echo "generating CA cert"
	cert-tool --cmd.generateRootCA --outCert="${CA_POOL_CERT_PATH}" --outKey="${CA_POOL_CERT_KEY_PATH}" \
		--cert.subject.cn="Root CA" --cert.signatureAlgorithm=${CERT_TOOL_SIGN_ALG} --cert.ellipticCurve=${CERT_TOOL_ELLIPTIC_CURVE} \
		--cert.validFrom="${CA_POOL_CERT_VALID_FROM}" --cert.validFor="${CA_POOL_CERT_VALID_FOR}"

	fqdnSAN="--cert.san.domain=$FQDN"
	if ip route get $FQDN 2>/dev/null >/dev/null; then
		fqdnSAN="--cert.san.ip=$FQDN"
	fi
	echo "generating HTTP cert"
	cert-tool --cmd.generateCertificate --outCert="${HTTP_CERT}" --outKey="${HTTP_CERT_KEY}" \
		--cert.subject.cn="localhost" --cert.san.domain="localhost" --cert.san.ip="0.0.0.0" --cert.san.ip="127.0.0.1" $fqdnSAN \
		--signerCert="${CA_POOL_CERT_PATH}" --signerKey="${CA_POOL_CERT_KEY_PATH}" \
		--cert.signatureAlgorithm=${CERT_TOOL_SIGN_ALG} --cert.ellipticCurve=${CERT_TOOL_ELLIPTIC_CURVE}

	echo "generating COAP-GW cert"
	COAP_GATEWAY_UNSECURE_FQDN=$FQDN
	COAP_GATEWAY_FQDN=$FQDN
	cert-tool --cmd.generateIdentityCertificate="${COAP_GATEWAY_HUB_ID}" --outCert="${COAP_GATEWAY_CERT}" \
		--outKey="${COAP_GATEWAY_CERT_KEY}" --cert.san.domain="${COAP_GATEWAY_FQDN}" --signerCert="${CA_POOL_CERT_PATH}" \
		--signerKey="${CA_POOL_CERT_KEY_PATH}" --cert.signatureAlgorithm=${CERT_TOOL_SIGN_ALG} --cert.ellipticCurve=${CERT_TOOL_ELLIPTIC_CURVE}

	echo "generating mongodb cert"
	cat "${HTTP_CERT}" > "${MONGODB_CERT_KEY}"
	cat "${HTTP_CERT_KEY}" >> "${MONGODB_CERT_KEY}"

	echo "generating DPS client device certs"
	mkdir -p "${DEVICE_CERTIFICATES_PATH}"
	cp "${CA_POOL_CERT_PATH}" "${DPS_CA_CERT}"
	cp "${CA_POOL_CERT_KEY_PATH}" "${DPS_CA_CERT_KEY}"
	cert-tool --signerCert="${DPS_CA_CERT}" --signerKey="${DPS_CA_CERT_KEY}" --outCert="${DPS_INTERMEDIATECA_CERT}" \
		--outKey="${DPS_INTERMEDIATECA_CERT_KEY}" --cert.basicConstraints.maxPathLen=0 --cert.subject.cn="intermediateCA" \
		--cmd.generateIntermediateCA --cert.signatureAlgorithm=${CERT_TOOL_SIGN_ALG} --cert.ellipticCurve=${CERT_TOOL_ELLIPTIC_CURVE}
	cert-tool --signerCert="${DPS_INTERMEDIATECA_CERT}" --signerKey="${DPS_INTERMEDIATECA_CERT_KEY}" \
		--outCert="${DPS_MFG_CERT}" --outKey="${DPS_MFG_CERT_KEY}" --cert.san.domain=localhost --cert.san.ip=127.0.0.1 \
		--cert.subject.cn="mfg" --cmd.generateCertificate --cert.signatureAlgorithm=${CERT_TOOL_SIGN_ALG} --cert.ellipticCurve=${CERT_TOOL_ELLIPTIC_CURVE}
	echo "generating alternative CA for testing"
	cert-tool --cmd.generateRootCA --outCert="${CA_POOL_CERT_ALT_PATH}" --outKey="${CA_POOL_CERT_KEY_ALT_PATH}" \
		--cert.subject.cn="Root CA" --cert.signatureAlgorithm=${CERT_TOOL_SIGN_ALG} --cert.ellipticCurve=${CERT_TOOL_ELLIPTIC_CURVE} \
		--cert.validFrom="${CA_POOL_CERT_VALID_FROM}" --cert.validFor="${CA_POOL_CERT_VALID_FOR}"
	chmod -R 0777 "${CERTIFICATES_PATH}"

	mkdir -p "${OAUTH_KEYS_PATH}"
	openssl genrsa -out ${OAUTH_KEYS_PATH}/idTokenKey.pem 4096
	openssl ecparam -name prime256v1 -genkey -noout -out ${OAUTH_KEYS_PATH}/accessTokenKey.pem
	openssl ecparam -name prime256v1 -genkey -noout -out ${OAUTH_KEYS_PATH}/m2mAccessTokenKey.pem

	# nats
	cat > /data/nats.config <<EOF
port: ${NATS_PORT}
max_pending: 128Mb
write_deadline: 10s
tls: {
	verify: true
	cert_file: "${LISTEN_FILE_CERT}"
	key_file: "${LISTEN_FILE_CERT_KEY}"
	ca_file: "${CA_POOL_CERT_PATH}"
}
EOF

	mkdir -p $LOGS_PATH
fi

if [ "${RUN}" = "true" ]; then
	echo "starting nats-server"
	nats-server -c /data/nats.config >$LOGS_PATH/nats-server.log 2>&1 &
	status=$?
	nats_server_pid=$!
	if [ $status -ne 0 ]; then
		echo "Failed to start nats-server: $status"
		sync
		cat "$LOGS_PATH/nats-server.log"
		exit $status
	fi

	NATS_HOST="localhost:${NATS_PORT}"
	NATS_URL="nats://${NATS_HOST}"

	# waiting for nats. Without wait, sometimes auth service didn't connect.
	i=0
	while [ $i -le 20 ]; do
		i=$((i+1))
		if nc -z localhost $NATS_PORT; then
			break
		fi
		if [ $i -eq 20 ]; then
			echo "Failed to connect to nats(${NATS_HOST})"
			exit 1
		fi
		echo "Try to reconnect to nats(${NATS_HOST}) $i"
		cat "$LOGS_PATH/nats-server.log"
		sleep 1
	done

	echo "starting mongo replica set"
	MONGO_REPLICA_SET=myReplicaSet
	startMongo 0 ${MONGO_PORT} ${MONGO_REPLICA_SET}
	startMongo 1 27018 ${MONGO_REPLICA_SET}
	startMongo 2 27019 ${MONGO_REPLICA_SET}
	mongosh --tls --tlsCAFile ${CA_POOL_CERT_PATH} --tlsCertificateKeyFile ${MONGODB_CERT_KEY} --eval "rs.initiate({
		_id: \"${MONGO_REPLICA_SET}\",
		members: [
			{_id: 0, host: \"localhost:${MONGO_PORT}\"},
			{_id: 1, host: \"localhost:27018\"},
			{_id: 2, host: \"localhost:27019\"}
		]
	})"

	# needed by dps-service.test, dps-mongodb.test, dps-clientcredentials.test
	export TEST_COAP_GW_CERT_FILE="${COAP_GATEWAY_CERT}"
	export TEST_COAP_GW_KEY_FILE="${COAP_GATEWAY_CERT_KEY}"
	export TEST_ROOT_CA_CERT="${CA_POOL_CERT_PATH}"
	export TEST_ROOT_CA_KEY="${CA_POOL_CERT_KEY_PATH}"
	export TEST_CLOUD_SID="${COAP_GATEWAY_HUB_ID}"
	export TEST_OAUTH_SERVER_ID_TOKEN_PRIVATE_KEY="${OAUTH_KEYS_PATH}/idTokenKey.pem"
	export TEST_OAUTH_SERVER_ACCESS_TOKEN_PRIVATE_KEY="${OAUTH_KEYS_PATH}/accessTokenKey.pem"
	export TEST_DPS_INTERMEDIATE_CA_CERT=${DPS_INTERMEDIATECA_CERT}
	export TEST_DPS_INTERMEDIATE_CA_KEY=${DPS_INTERMEDIATECA_CERT_KEY}
	# alternative certificate authority to validate security
	export TEST_DPS_ROOT_CA_CERT_ALT="${CA_POOL_CERT_ALT_PATH}"
	export TEST_DPS_ROOT_CA_KEY_ALT="${CA_POOL_CERT_KEY_ALT_PATH}"
	export M2M_OAUTH_SERVER_PRIVATE_KEY="${OAUTH_KEYS_PATH}/m2mAccessTokenKey.pem"
	export TEST_COAP_GATEWAY_UDP_ENABLED="${COAP_GATEWAY_UDP_ENABLED}"
	export TEST_DPS_UDP_ENABLED="${DPS_UDP_ENABLED}"

	echo "starting dps-service test"
	dps-service.test -test.v -test.timeout 1200s

	echo "starting dps-mongodb test"
	dps-mongodb.test -test.v -test.timeout 600s

	echo "starting dps-clientcredentials test"
	dps-clientcredentials.test -test.v -test.timeout 600s
fi
