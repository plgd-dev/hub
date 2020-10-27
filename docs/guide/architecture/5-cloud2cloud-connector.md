# Cloud2Cloud connector

## Description

Connects the cloud to a OCF Cloud, populate devices from the OCF cloud, update devices of the OCF cloud, maintenance of linked clouds and linked accounts.

## Docker Image

```bash
docker pull plgd/cloud2cloud-connector:vnext
```

## API

Follow [OCF Cloud API For Cloud Services Specification](https://openconnectivity.org/specs/OCF_Cloud_API_For_Cloud_Services_Specification_v2.2.0.pdf)

### Commands

- maintenance of linked clouds
- maintenance of linked accounts
- [swagger](https://petstore.swagger.io/?url=https://raw.githubusercontent.com/plgd-dev/cloud/master/cloud2cloud-connector/swagger.yaml)

## Configuration

| Option | ENV variable | Type | Description | Default |
| ------ | --------- | ----------- | ------- | ------- |
| `-` | `ADDRESS` | string | `listen address` | `"0.0.0.0:9100"` |
| `-` | `AUTH_SERVER_ADDRESS` | string | `authoriztion server address` | `"127.0.0.1:9100"` |
| `-` | `RESOURCE_AGGREGATE_ADDRESS` | string | `resource aggregate address` | `"127.0.0.1:9100"` |
| `-` | `RESOURCE_DIRECTORY_ADDRESS` | string | `resource directory address` | `"127.0.0.1:9100"` |
| `-` | `JWKS_URL` | string | `url to get JSON Web Key` | `""` |
| `-` | `SERVICE_OAUTH_CALLBACK` | string | `external redirect url callback for acquire authorization code` | `""` |
| `-` | `SERVICE_EVENTS_URL` | string | `external url where will be send events from another cloud` | `""` |
| `-` | `SERVICE_PULL_DEVICES_DISABLED` | bool | `disable get devices via pull for all cloud` | `false` |
| `-` | `SERVICE_PULL_DEVICES_INTERVAL` | string | `time interval between pulls`  | `5s` |
| `-` | `SERVICE_TASK_PROCESSOR_CACHE_SIZE` | int | `size of processor task queue` | `2048` |
| `-` | `SERVICE_TASK_PROCESSOR_TIMEOUT` | int | `timeout for one running task` | `"5s"` |
| `-` | `SERVICE_TASK_PROCESSOR_MAX_PARALLEL` | int | `count of running tasks in same time` | `128` |
| `-` | `SERVICE_TASK_PROCESSOR_DELAY` | string | `delay task before start`  | `0s` |
| `-` | `SERVICE_RECONNECT_INTERVAL` | string | `try to reconnect after interval to resource-directory when connection was closed` | `"10s"` |
| `-` | `SERVICE_RESUBSCRIBE_INTERVAL` | string | `try to resubscribe after interval to resource-directory when subscription not exist` | `"10s"` |
| `-` | `SERVICE_OAUTH_ENDPOINT_TOKEN_URL` | string | `url to get service access token via OAUTH client credential flow` | `""` |
| `-` | `SERVICE_OAUTH_CLIENT_ID` | string | `client id for authentication to get access token` | `""` |
| `-` | `SERVICE_OAUTH_CLIENT_SECRET` | string | `secrest for authentication to get access token` | `""` |
| `-` | `SERVICE_OAUTH_AUDIENCE` | string | `refer to the resource servers that should accept the token` | `""` |
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
| `-` | `LINKED_STORE_MONGO_HOST` | string | `host of mongo database - uri without scheme` | `"localhost:27017"` |
| `-` | `LINKED_STORE_MONGO_DATABASE` | string | `name of database` | `"cloud2cloudConnector"` |
| `-` | `LOG_ENABLE_DEBUG` | bool | `debug logging` | `false` |
