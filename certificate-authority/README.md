[![codecov](https://codecov.io/gh/plgd-dev/certificate-authority/branch/master/graph/badge.svg)](https://codecov.io/gh/plgd-dev/certificate-authority)
[![Go Report](https://goreportcard.com/badge/github.com/plgd-dev/cloud/certificate-authority)](https://goreportcard.com/report/github.com/plgd-dev/cloud/certificate-authority)

# certificate-authority

## Configuration
| Option | ENV variable | Type | Description | Default |
| ------ | --------- | ----------- | ------- | ------- |
| `-` | `ADDRESS` | string | tbd | `"0.0.0.0:7000"` |
| `-` | `DIAL_CA_POOL` | string | `path to pem file of CAs` |  `""` |
| `-` | `DIAL_CERT_KEY_NAME` | string | `name of pem certificate key file` | `""` |
| `-` | `DIAL_CERT_DIR_PATH` | string | `path to directory which contains DIAL_CERT_KEY_NAME and DIAL_CERT_NAME` | `""` |
| `-` | `DIAL_CERT_NAME` | string | `name of pem certificate file` | `""` |
| `-` | `DIAL_USE_SYSTEM_CERTIFICATION_POOL` | bool | `load CAs from system` | `false` |
| `-` | `LISTEN_CA_POOL` | string | `path to pem file of CAs` |  `""` |
| `-` | `LISTEN_CERT_KEY_NAME` | string | `name of pem certificate key file` | `""` |
| `-` | `LISTEN_CERT_DIR_PATH` | string | `path to directory which contains LISTEN_CERT_KEY_NAME and LISTEN_CERT_NAME` | `""` |
| `-` | `LISTEN_CERT_NAME` | string | `name of pem certificate file` | `""` |
| `-` | `LISTEN_USE_SYSTEM_CERTIFICATION_POOL` | bool | `load CAs from system` | `false` |
| `-` | `LOG_ENABLE_DEBUG` | bool | `debug logging` | `false` |
| `-` | `JWKS_URL` | string | url | `""` |
| `-` | `SIGNER_CERTIFICATE`| string | path to cert | `""` |
| `-` | `SIGNER_PRIVATE_KEY`| string | path to private key of cert | `""` |
| `-` | `SIGNER_VALID_DURATION` | string | signed certificate expire in | `"87600h"` |
