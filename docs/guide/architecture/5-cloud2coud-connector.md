# 4. COAP gateway

## Description

Connects the cloud to a OCF Cloud, populate devices from the OCF cloud or update devices of the CLOUD cloud.

## API

Follow [OCF Cloud API For Cloud Services Specification](https://openconnectivity.org/specs/OCF_Cloud_API_For_Cloud_Services_Specification_v2.2.0.pdf)

### Commands

- POST /oic/sec/account - sign up the device with authorization code
- DELETE /oic/sec/account - sign off the device with access token
- POST /oic/sec/tokenrefresh - refresh access token with refresh token
- POST /oic/sec/session - sign in the device with access token and with login true
- POST /oic/sec/session - sign out the device with access token and with login false
- POST /oic/rd - publish resources from the signed device
- DELETE /oic/rd - unpublish resources from the signed device
- GET /oic/res - discover all cloud devices resources from the signed device
- GET /oic/route/{deviceID}/{href} - get/observe resource of the cloud device from signed device
- POST /oic/route/{deviceID}/{href} - update resource of the cloud device from signed device

## Docker Image

```bash
docker pull plgd/cloud2cloud-connector:vnext
```

## Configuration

| Option | ENV variable | Type | Description | Default |
| ------ | --------- | ----------- | ------- | ------- |
| `-` | `ADDRESS` | string | `listen address` | `"0.0.0.0:5684"` |
| `-` | `EXTERNAL_PORT` | string | `used to fill discovery hrefs` | `"0.0.0.0:5684"` |
| `-` | `FQDN` | string | `used to fill discovery` | `"coapgw.ocf.cloud"` |
| `-` | `AUTH_SERVER_ADDRESS` | string | `authoriztion server address` | `"127.0.0.1:9100"` |
| `-` | `RESOURCE_AGGREGATE_ADDRESS` | string | `resource aggregate address` | `"127.0.0.1:9100"` |
| `-` | `RESOURCE_DIRECTORY_ADDRESS` | string | `resource directory address` | `"127.0.0.1:9100"` |
| `-` | `REQUEST_TIMEOUT` | string | `wait for update/retrieve resource` | `10s` |
| `-` | `KEEPALIVE_ENABLE` | bool | `check devices connection` | true |
| `-` | `KEEPALIVE_TIMEOUT_CONNECTION` | string | `close inactive connection after limit` | `"20s"` |
| `-` | `DISABLE_BLOCKWISE_TRANSFER` | bool | `disable blockwise transfer` | `true` |
| `-` | `BLOCKWISE_TRANSFER_SZX` | int | `size of blockwise transfer block` | `1024` |
| `-` | `DISABLE_TCP_SIGNAL_MESSAGE_CSM` | bool | `disable send CSM when connection was established` | `false` |
| `-` | `DISABLE_PEER_TCP_SIGNAL_MESSAGE_CSMS` | bool | `disable process peer CSM` | `true` |
| `-` | `ERROR_IN_RESPONSE` | bool | `send text error message in response` |  `true` |
| `-` | `SERVICE_OAUTH_ENDPOINT_TOKEN_URL` | string | `url to get service access token via OAUTH client credential flow` | `""` |
| `-` | `SERVICE_OAUTH_CLIENT_ID` | string | `client id for authentication to get access token` | `""` |
| `-` | `SERVICE_OAUTH_CLIENT_SECRET` | string | `secrest for authentication to get access token` | `""` |
| `-` | `SERVICE_OAUTH_AUDIENCE` | string | `refer to the resource servers that should accept the token` | `""` |
| `-` | `HEARTBEAT` | string | `defines check of live service` | `"4s"` |
| `-` | `MAX_MESSAGE_SIZE` | int | `defines max message size which can be send/receive via coap` | `262144` |
| `-` | `DIAL_TYPE` | string | `defines how to obtain dial TLS certificates - options: acme|file` | `"acme"` |
| `-` | `DIAL_ACME_CA_POOL` | string | `path to pem file of CAs` | `""` |
| `-` | `DIAL_ACME_DIRECTORY_URL` | string |  `url of acme directory` | `""` |
| `-` | `DIAL_ACME_DOMAINS` | string | `list of domains for which will be in certificate provided from acme` | `""` |
| `-` | `DIAL_ACME_REGISTRATION_EMAIL` | string | `registration email for acme` | `""` |
| `-` | `DIAL_ACME_TICK_FREQUENCY` | string | `interval of validate certificate` | `""` |
| `-` | `DIAL_ACME_USE_SYSTEM_CERTIFICATION_POOL` | bool | `load CAs from system` | `false` |
| `-` | `DIAL_FILE_CA_POOL` | string | `path to pem file of CAs` |  `""` |
| `-` | `DIAL_FILE_CERT_KEY_NAME` | string | `name of pem certificate key file` | `""` |
| `-` | `DIAL_FILE_CERT_DIR_PATH` | string | `path to directory which contains DIAL_FILE_CERT_KEY_NAME and DIAL_FILE_CERT_NAME` | `""` |
| `-` | `DIAL_FILE_CERT_NAME` | string | `name of pem certificate file` | `""` |
| `-` | `DIAL_FILE_USE_SYSTEM_CERTIFICATION_POOL` | bool | `load CAs from system` | `false` |
| `-` | `LISTEN_TYPE` | string | `defines how to obtain listen TLS certificates - options: acme|file` | `"acme"` |
| `-` | `LISTEN_ACME_CA_POOL` | string | `path to pem file of CAs` | `""` |
| `-` | `LISTEN_ACME_DIRECTORY_URL` | string |  `url of acme directory` | `""` |
| `-` | `LISTEN_ACME_DOMAINS` | string | `list of domains for which will be in certificate provided from acme` | `""` |
| `-` | `LISTEN_ACME_REGISTRATION_EMAIL` | string | `registration email for acme` | `""` |
| `-` | `LISTEN_ACME_TICK_FREQUENCY` | string | `interval of validate certificate` | `""` |
| `-` | `LISTEN_ACME_USE_SYSTEM_CERTIFICATION_POOL` | bool | `load CAs from system` | `false` |
| `-` | `LISTEN_FILE_CA_POOL` | string | `path to pem file of CAs` | `""` |
| `-` | `LISTEN_FILE_CERT_KEY_NAME` | string | `name of pem certificate key file` | `""` |
| `-` | `LISTEN_FILE_CERT_DIR_PATH` | string | `path to directory which contains LISTEN_FILE_CERT_KEY_NAME and LISTEN_FILE_CERT_NAME` | `""` |
| `-` | `LISTEN_FILE_CERT_NAME` | string | `name of pem certificate file` | `""` |
| `-` | `LISTEN_FILE_USE_SYSTEM_CERTIFICATION_POOL` | bool | `load CAs from system` | `false` |
| `-` | `LISTEN_WITHOUT_TLS` | bool | `listen without TLS` | `false` |
| `-` | `LOG_ENABLE_DEBUG` | bool | `debug logging` | `false` |
