# Scalable OCF Cloud Hosting / Testing

Being [plugged.in](https://pluggedin.cloud) provides you a complete set of tools and services to manage your devices at scale. Allowing for the processing of real-time device data and interconnection of your devices and applications based on an interoperable standard. Interconnect, monitor and manage your devices in a cloud native way.

## OCF Cloud Bundle

Provides a simple docker cloud image for **testing purpose**.

### Features

- [OCF Native Cloud](https://openconnectivity.org/specs/OCF_Device_To_Cloud_Services_Specification_v2.1.0.pdf)
- OAUTH Athorization code is not verified
- [GRPC](https://github.com/plgd-dev/cloud/blob/master/grpc-gateway/pb/service.proto)

### Supported clients

- [iotivity-lite v2+](https://github.com/iotivity/iotivity-lite)

### Pull the image

```bash
docker pull plgd/bundle:vnext
```

### Configuration

Image can be configured via environment variables as argument `-e ENV=VALUE` of command `docker`:
| ENV variable | Type | Description | Default |
| --------- | ----------- | ------- | ------- |
| `FQDN` | string | public FQDN for bundle | `"localhost"` |
| `NGINX_PORT` | uint16 | nginx https port for localhost | `"443"` |
| `OWNER_CLAIM` | string | which claim will be used from JWT to determine ownership | `"sub"` |
| `COAP_GATEWAY_UNSECURE_PORT` | uint16 | exposed public port for coap-tcp  | `"5683"` |
| `COAP_GATEWAY_UNSECURE_ADDRESS` | string | coap-tcp listen address | `"0.0.0.0:5683"` |
| `COAP_GATEWAY_PORT` | uint16 | exposed public port for coaps-tcp  | `"5684"` |
| `COAP_GATEWAY_ADDRESS` | string | coaps-tcp listen address | `"0.0.0.0:5684"` |
| `COAP_GATEWAY_CLOUD_ID` | string | cloud id | `"00000000-0000-0000-0000-000000000001"` |
| `COAP_GATEWAY_LOG_MESSAGES` | bool | log received/send messages | false |
| `GRPC_GATEWAY_PORT`| uint16 | secure grpc-tcp listen port for localhost | `"9084"` |
| `HTTP_GATEWAY_PORT`| uint16 | secure grpc-tcp listen port for localhost | `"9086"` |
| `CERTIFICATE_AUTHORITY_PORT` | uint16 | secure grpc-tcp listen port for localhost | `"9087"` |
| `OAUTH_SERVER_PORT` | uint16 | secure grpc-tcp listen port for localhost | `"9088"` |
| `RESOURCE_AGGREGATE_PORT` | uint16 | secure grpc-tcp listen port for localhost | `"9083"` |
| `RESOURCE_DIRECTORY_PORT` | uint16 | secure grpc-tcp listen port for localhost | `"9082"` |
| `IDENTITY_STORE_PORT` | uint16 | secure grpc-tcp listen port for localhost | `"9081"` |
| `MONGO_PORT` | uint16 | mongo listen port for localhost | `"10000"` |
| `NATS_PORT` | uint16 | nats listen port for localhost | `"10001"` |

### Run

All datas, confgurations and logs are stored under /data directory at the container.

```bash
mkdir -p `pwd`/data
docker run -d --network=host -v `pwd`/data:/data --name=cloud -t plgd/bundle:vnext
```

### Access via HTTPS/GRPC

All http-gateway, oauth-server, grpc-gateway, certificate-authority endpoints are accessible through nginx.

- HTTP - UI: `https://{FQDN}:{NGINX_PORT}` eg: `https://localhost:8443`
- HTTP - API: `https://{FQDN}:{NGINX_PORT}/api/v1/...` eg: `https://localhost:8443/api/v1/devices`
- GRPC: `{FQDN}:{NGINX_PORT}` eg: `localhost:8443`

### Device Onboarding

The onboarding values which should be set to the [coapcloudconf](https://github.com/openconnectivityfoundation/cloud-services/blob/c2c/swagger2.0/oic.r.coapcloudconf.swagger.json) device resource are:

#### Unsecured device

| Attribute | Value |
| --------- | ------|
| `apn` | `plgd` |
| `cis` | `coap+tcp://127.0.0.1:5683` |
| `sid` | `same as is set in COAP_GATEWAY_CLOUD_ID` |
| `at` | `test` |

##### Unsecured iotivity-lite sample device example

```bash
# Start the cloud container with "unsecured" parameters
docker run -d --network=host --name=cloud -t plgd/bundle:vnext \
-e COAP_GATEWAY_CLOUD_ID="00000000-0000-0000-0000-000000000001" \
-e COAP_GATEWAY_UNSECURE_PORT="5683" \
-e COAP_GATEWAY_UNSECURE_ADDRESS="0.0.0.0:5683" \
-e "coap+tcp://127.0.0.1:5683"
```

```bash
# Retrieve iotivity-lite project
git clone --recursive https://github.com/iotivity/iotivity-lite.git
cd ./iotivity-lite/port/linux

# Build and run unsecured applications
make CLOUD=1 SECURE=0 cloud_server cloud_client

# Start unsecured device sample
./cloud_server cloud_server test coap+tcp://127.0.0.1:5683 00000000-0000-0000-0000-000000000001 plgd

# Start unsecured client
./cloud_client cloud_client test coap+tcp://127.0.0.1:5683 00000000-0000-0000-0000-000000000001 plgd
```

#### Secured device

| Attribute | Value |
| --------- | ------|
| `apn` | `plgd`|
| `cis` | `coaps+tcp://127.0.0.1:5684` |
| `sid` | `same as is set in COAP_GATEWAY_CLOUD_ID` |
| `at` | `test` |

##### Conditions

- Device must be owned.
- Cloud CA must be set as TRUST CA with subject COAP_GATEWAY_CLOUD_ID in device.
- Cloud CA in PEM:
  ```bash
  docker exec -it cloud cat CERTIFICATES_PATH/root_ca.crt
  ```
- ACL for Cloud (Subject: COAP_GATEWAY_CLOUD_ID) must be set with full access to all published resources in device.

##### Secured iotivity-lite sample device example

```bash
# Start the cloud container with "secured" parameters
docker run -d --network=host --name=cloud -t plgd/bundle:vnext \
-e COAP_GATEWAY_CLOUD_ID="00000000-0000-0000-0000-000000000001" \
-e COAP_GATEWAY_PORT="5684" \
-e COAP_GATEWAY_ADDRESS="0.0.0.0:5684" \
-e "coaps+tcp://127.0.0.1:5684"
```

```bash
# Retrieve iotivity-lite project
git clone --recursive https://github.com/iotivity/iotivity-lite.git
cd ./iotivity-lite/port/linux

# Then build secured applications and onboarding_tool
make CLOUD=1 SECURE=1 PKI=1 onboarding_tool cloud_server cloud_client

# Retrieve cloud CA and set it as "pki_certs/cloudca.pem", to make the sample import this as TRUST CA
docker exec -it cloud cat /data/certs/root_ca.crt > pki_certs/cloudca.pem

# Start secured device sample
./cloud_server cloud_server test coaps+tcp://127.0.0.1:5684 00000000-0000-0000-0000-000000000001 plgd
# Start secured client
./cloud_client cloud_client test coaps+tcp://127.0.0.1:5684 00000000-0000-0000-0000-000000000001 plgd

# Execute onboarding_tool and:
# - Onboard and set ACLs to the device sample as described in https://github.com/iotivity/iotivity-lite#step-1-onboard-and-provision-the-server
# - Onboard the client as described in https://github.com/iotivity/iotivity-lite#step-2-onboard-the-client
# - Pair client and server as described in https://github.com/iotivity/iotivity-lite#step-3-pair-server-and-client
./onboarding_tool

# Then restart device sample and client
```

### Build a COAP client application

To build the client you need to have **golang v1.16+**.

```bash
cd ./client/coap
go build
./coap --help
# gets a resource links of the registered devices from cloud
./coap --signUp test --href /oic/res
```

### Build a GRPC client application

To build the client you need to have **golang v1.16+**.

```bash
cd ./client/grpc
go build
./grpc --help
# gets all resources with contents from cloud
./grpc
# gets resources of device with contents from cloud
./grpc --deviceid {deviceID}
# gets devices from cloud
./grpc --getdevices
```

### HTTP access

[REST API](https://petstore.swagger.io/?url=https://raw.githubusercontent.com/plgd-dev/cloud/master/http-gateway/swagger.yaml)
