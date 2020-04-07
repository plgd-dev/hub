[![Go Report](https://goreportcard.com/badge/github.com/go-ocf/cloud/openapi-connector)](https://goreportcard.com/report/github.com/go-ocf/cloud/openapi-connector)
[![Gitter](https://badges.gitter.im/ocfcloud/Lobby.svg)](https://gitter.im/ocfcloud/Lobby?utm_source=badge&utm_medium=badge&utm_campaign=pr-badge)

# openapi-connector

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