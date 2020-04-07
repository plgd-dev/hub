[![Go Report](https://goreportcard.com/badge/github.com/go-ocf/cloud/resource-aggregate)](https://goreportcard.com/report/github.com/go-ocf/cloud/resource-aggregate)
[![Gitter](https://badges.gitter.im/ocfcloud/Lobby.svg)](https://gitter.im/ocfcloud/Lobby?utm_source=badge&utm_medium=badge&utm_campaign=pr-badge)

# resource - aggregate

## service initialization
Service can be initialized with different Databases(EventStore) or Message systems(Publisher) via function `New`. For example `cmd/service` it uses mongodb/kafka. To initialize package with other eventstore/publisher it must satisfy interfaces:

# Build

## Docker

```sh
make build-servicecontainer
```
## Local machine

```sh
dep ensure -v --vendor-only
go build ./cmd/coap-gateway-service/
```

## Configuration
| Option | ENV variable | Type | Description | Default |
| ------ | --------- | ----------- | ------- | ------- |
| `-` | `ADDRESS` | string | tbd | `"0.0.0.0:9100"` |
| `-` | `AUTH_SERVER_ADDRESS` | string | tbd | `"127.0.0.1:9100"` |
| `-` | `SNAPSHOT_THRESHOLD` | int | tbd | `128` |
| `-` | `OCC_MAX_RETRY` | int | tbd | `8` |
| `-` | `NATS_URL` | string | tbd | `"nats://localhost:4222"` |
| `-` | `MONGODB_URI` | string | tbd | `"mongodb://localhost:27017"` |
| `-` | `MONGODB_DATABASE` | string | tbd | `"eventstore"` |
| `-` | `MONGODB_BATCH_SIZE` | int | tbd | `16` |
| `-` | `MONGODB_MAX_POOL_SIZE` | int | tbd | `16` |
| `-` | `MONGODB_MAX_CONN_IDLE_TIME` | string | tbd | `"240s"` |
| `-` | `DIAL_TYPE` | string | tbd | `"acme"` |
| `-` | `DIAL_ACME_CA_POOL` | string | tbd | `""` |
| `-` | `DIAL_ACME_DIRECTORY_URL` | string | tbd | `""` |
| `-` | `DIAL_ACME_DOMAINS` | string | tbd | `""` |
| `-` | `DIAL_ACME_REGISTRATION_EMAIL` | string | tbd | `""` |
| `-` | `DIAL_ACME_TICK_FREQUENCY` | string | tbd | `""` |
| `-` | `DIAL_FILE_CA_POOL` | string | tbd | `""` |
| `-` | `DIAL_FILE_CERT_KEY_NAME` | string | tbd | `""` |
| `-` | `DIAL_FILE_CERT_DIR_PATH` | string | tbd | `""` |
| `-` | `DIAL_FILE_CERT_NAME` | string | tbd | `""` |
| `-` | `LISTEN_TYPE` | string | tbd | `"acme"` |
| `-` | `LISTEN_ACME_CA_POOL` | string | tbd | `""` |
| `-` | `LISTEN_ACME_DIRECTORY_URL` | string | tbd | `""` |
| `-` | `LISTEN_ACME_DOMAINS` | string | tbd | `""` |
| `-` | `LISTEN_ACME_REGISTRATION_EMAIL` | string | tbd | `""` |
| `-` | `LISTEN_ACME_TICK_FREQUENCY` | string | tbd | `""` |
| `-` | `LISTEN_FILE_CA_POOL` | string | tbd | `""` |
| `-` | `LISTEN_FILE_CERT_KEY_NAME` | string | tbd | `""` |
| `-` | `LISTEN_FILE_CERT_DIR_PATH` | string | tbd | `""` |
| `-` | `LISTEN_FILE_CERT_NAME` | string | tbd | `""` |
| `-` | `LOG_ENABLE_DEBUG` | bool | tbd | `false` |