# COAP gateway

## Description

OCF Servers / Clients communicate over TCP / UDP using the CoAP application protocol. Communication within the OCF Native Cloud shouldn't be restricted to the CoAP protocol, implementation should allow the use of whatever protocol might be introduced in the future. That's why the gateway is the access point for CoAP over TCP, and further communication is OCF Native Cloud specific.

TCP connection to the OCF Native Cloud is by its nature stateful. The OCF CoAP Gateway is therefore also stateful, keeping open connections to the OCF Servers / Clients.  The goal of the Gateway is to translate between the OCF Servers / Clients (CoAP) and the protocol of the OCF Native Cloud and communicate in an asynchronous way.

### Validation

- OCF CoAP Gateway can accept requests from the OCF Client / Server only after a successful sign-in
- OCF CoAP Gateway can forward requests to the OCF Client / Server only after successful sign-in
- If sign-in was not issued within the configured amount of time or sign-in request failed, OCF Native Cloud will forcibly close the TCP connection
- OCF CoAP Gateway sends command to update device core resource with its status.
  - Online when the device was successfully signed-in and communication lock released
  - Offline when the device was disconnected or signed-out
- Access Token from a successful sign-in must be locally persisted in the OCF CoAP Gateway and linked with an opened TCP channel
- Access Token linked with the opened TCP channel has to be included in each command issued to other OCF Native Cloud components
- OCF CoAP Gateway processes only those commands, which are designated for a device which the Gateway has an opened TCP channel to
- OCF CoAP Gateway is observing each resource published to the resource directory and publishes an event for every change
- OCF CoAP Gateway retrieves each published resource and updates Resources
- OCF CoAP Gateway has to expose the coap ping-pong + retry count configuration, which can be configured during the deployment
- OCF CoAP Gateway has to ping the device in the configured time, if pong is not received after the configured number of retries, then the connection with the device is closed and device is set as offline
- OCF CoAP Gateway processes events from Resources, by issuing a proper CoAP request to the device and raising an event with the response
- OCF CoAP Gateway has to process a waiting request within the configured time, or set the device as offline

## Docker Image

```bash
docker pull plgd/coap-gateway:vnext
```

## API

Follow [OCF Device To Cloud Services Specification](https://openconnectivity.org/specs/OCF_Device_To_Cloud_Services_Specification_v2.2.0.pdf)

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
- DELETE /oic/route/{deviceID}/{href} - delete resource of the cloud device from signed device

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
| `-` | `DISABLE_BLOCKWISE_TRANSFER` | bool | `disable blockwise transfer` | `false` |
| `-` | `BLOCKWISE_TRANSFER_SZX` | int | `size of blockwise transfer block` | `1024` |
| `-` | `DISABLE_TCP_SIGNAL_MESSAGE_CSM` | bool | `disable send CSM when connection was established` | `false` |
| `-` | `DISABLE_PEER_TCP_SIGNAL_MESSAGE_CSMS` | bool | `disable process peer CSM` | `false` |
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
| `-` | `LISTEN_ACME_DEVICE_ID` | string | `deviceID for OCF Identity Certificate` | `""` |
| `-` | `LISTEN_FILE_CA_POOL` | string | `path to pem file of CAs` | `""` |
| `-` | `LISTEN_FILE_CERT_KEY_NAME` | string | `name of pem certificate key file` | `""` |
| `-` | `LISTEN_FILE_CERT_DIR_PATH` | string | `path to directory which contains LISTEN_FILE_CERT_KEY_NAME and LISTEN_FILE_CERT_NAME` | `""` |
| `-` | `LISTEN_FILE_CERT_NAME` | string | `name of pem certificate file` | `""` |
| `-` | `LISTEN_FILE_USE_SYSTEM_CERTIFICATION_POOL` | bool | `load CAs from system` | `false` |
| `-` | `LISTEN_WITHOUT_TLS` | bool | `listen without TLS` | `false` |
| `-` | `LOG_ENABLE_DEBUG` | bool | `debug logging` | `false` |
