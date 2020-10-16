# 2. Resource aggregate

## Description
According to CQRS pattern it translates commands to events, store them to DB and publish them to messaging system.

## API
All requests to service must contains valid access token in (grpc metadata)[https://github.com/grpc/grpc-go/blob/master/Documentation/grpc-auth-support.md#oauth2]. Any success command creates event and it can create additional snapshot event. The event is stored in DB and published via messaging system.

### Commands:
- publish resource - create resource/republish of the device
- unpublish resource - unpublish resource from the cloud
- notify resource changed - set/update content of the resource in the cloud
- update resource - request to update resource in the device via cloud
- confirm resource update - response to update resource request
- retrieve resource - request to retrieve resource from the device via cloud
- confirm resource retrieve - response to retrieve resource request

### Contract
 - [service](https://github.com/plgd-dev/cloud/blob/master/resource-aggregate/pb/service.proto)
 - [requets/responses](https://github.com/plgd-dev/cloud/blob/master/resource-aggregate/pb/commands.proto)
 - [events](https://github.com/plgd-dev/cloud/blob/master/resource-aggregate/pb/events.proto)

## Configuration
| Option | ENV variable | Type | Description | Default |
| ------ | --------- | ----------- | ------- | ------- |
| `-` | `ADDRESS` | string | `listen address` | `"0.0.0.0:9100"` |
| `-` | `AUTH_SERVER_ADDRESS` | string | `authoriztion server address` | `"127.0.0.1:9100"` |
| `-` | `SNAPSHOT_THRESHOLD` | int | `number of events to spawn snapshot event` | `128` |
| `-` | `OCC_MAX_RETRY` | int | `maximum tries to store event to db` | `8` |
| `-` | `JWKS_URL` | string | `url to get JSON Web Key` | `""` |
| `-` | `NATS_URL` | string | `url to nats messaging system` | `"nats://localhost:4222"` |
| `-` | `MONGODB_URI` | string | `uri to mongo database` | `"mongodb://localhost:27017"` |
| `-` | `MONGODB_DATABASE` | string | `name of database` | `"eventstore"` |
| `-` | `MONGODB_BATCH_SIZE` | int | `maximum number resources in one batch request`  | `16` |
| `-` | `MONGODB_MAX_POOL_SIZE` | int | `maximum parallel request to DB` | `16` |
| `-` | `MONGODB_MAX_CONN_IDLE_TIME` | string |  `maximum time of idle connection` | `"240s"` |
| `-` | `DIAL_TYPE` | string | `defines how to obtain dial TLS certificates - options: acme|file` | `"acme"` |
| `-` | `DIAL_ACME_CA_POOL` | string | `path to pem file of CAs` | `""` |
| `-` | `DIAL_ACME_DIRECTORY_URL` | string |  `url of acme directory` | `""` |
| `-` | `DIAL_ACME_DOMAINS` | string | `list of domains for which will be in certificate provided from acme` | `""` |
| `-` | `DIAL_ACME_REGISTRATION_EMAIL` | string | `registration email for acme` | `""` |
| `-` | `DIAL_ACME_TICK_FREQUENCY` | string | `interval of validate certificate` | `""` |
| `-` | `DIAL_ACME_USE_SYSTEM_CERTIFICATION_POOL` | bool | `load CAs from system` | `false` |
| `-` | `DIAL_FILE_CA_POOL` | string | tbd | `path to pem file of CAs` |
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
| `-` | `LISTEN_FILE_CA_POOL` | string | tbd | `path to pem file of CAs` |
| `-` | `LISTEN_FILE_CERT_KEY_NAME` | string | `name of pem certificate key file` | `""` |
| `-` | `LISTEN_FILE_CERT_DIR_PATH` | string | `path to directory which contains LISTEN_FILE_CERT_KEY_NAME and LISTEN_FILE_CERT_NAME` | `""` |
| `-` | `LISTEN_FILE_CERT_NAME` | string | `name of pem certificate file` | `""` |
| `-` | `LISTEN_FILE_USE_SYSTEM_CERTIFICATION_POOL` | bool | `load CAs from system` | `false` |
| `-` | `LOG_ENABLE_DEBUG` | bool | `debug logging` | `false` |
