[![Build Status](https://travis-ci.com/go-ocf/certificate-authority.svg?branch=master)](https://travis-ci.com/go-ocf/certificate-authority)
[![codecov](https://codecov.io/gh/go-ocf/certificate-authority/branch/master/graph/badge.svg)](https://codecov.io/gh/go-ocf/certificate-authority)
[![Go Report](https://goreportcard.com/badge/github.com/go-ocf/ocf-cloud/certificate-authority)](https://goreportcard.com/report/github.com/go-ocf/ocf-cloud/certificate-authority)

# certificate-authority

## Docker

### Secure build
```sh
docker build . --network=host -t certificate-authority:build-secure --target build-secure
```

### Insecure build
```sh
docker build . --network=host -t certificate-authority:build-insecure --target build-insecure
```

## Local machine

### Secure build
```sh
dep ensure -v --vendor-only
go generate ./vendor/github.com/go-ocf/kit/security
go build ./cmd/certificate-authority-service/
```

### Insecure build
```sh
dep ensure -v --vendor-only
OCF_INSECURE=TRUE go generate ./vendor/github.com/go-ocf/kit/security
go build ./cmd/certificate-authority-service/
```