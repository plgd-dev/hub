[![Build Status](https://travis-ci.com/go-ocf/grpc-gateway.svg?branch=master)](https://travis-ci.com/go-ocf/grpc-gateway)
[![codecov](https://codecov.io/gh/go-ocf/grpc-gateway/branch/master/graph/badge.svg)](https://codecov.io/gh/go-ocf/grpc-gateway)
[![Go Report](https://goreportcard.com/badge/github.com/go-ocf/ocf-cloud/grpc-gateway)](https://goreportcard.com/report/github.com/go-ocf/ocf-cloud/grpc-gateway)
[![Gitter](https://badges.gitter.im/ocfcloud/Lobby.svg)](https://gitter.im/ocfcloud/Lobby?utm_source=badge&utm_medium=badge&utm_campaign=pr-badge)

# grpc-gateway

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
