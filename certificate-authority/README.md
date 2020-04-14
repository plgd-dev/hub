[![codecov](https://codecov.io/gh/go-ocf/certificate-authority/branch/master/graph/badge.svg)](https://codecov.io/gh/go-ocf/certificate-authority)
[![Go Report](https://goreportcard.com/badge/github.com/go-ocf/cloud/certificate-authority)](https://goreportcard.com/report/github.com/go-ocf/cloud/certificate-authority)

# certificate-authority

## Configuration
| Option | ENV variable | Type | Description | Default |
| ------ | --------- | ----------- | ------- | ------- |
| `-` | `ADDRESS` | string | tbd | `"0.0.0.0:5684"` |
| `-` | `DIAL_ACME_CA_POOL` | string | tbd | `""` |
| `-` | `DIAL_ACME_DIRECTORY_URL` | string | tbd | `""` |
| `-` | `DIAL_ACME_DOMAINS` | string | tbd | `""` |
| `-` | `DIAL_ACME_REGISTRATION_EMAIL` | string | tbd | `""` |
| `-` | `DIAL_ACME_TICK_FREQUENCY` | string | tbd | `""` |
| `-` | `LISTEN_ACME_CA_POOL` | string | tbd | `""` |
| `-` | `LISTEN_ACME_DIRECTORY_URL` | string | tbd | `""` |
| `-` | `LISTEN_ACME_DOMAINS` | string | tbd | `""` |
| `-` | `LISTEN_ACME_REGISTRATION_EMAIL` | string | tbd | `""` |
| `-` | `LISTEN_ACME_TICK_FREQUENCY` | string | tbd | `""` |
| `-` | `LOG_ENABLE_DEBUG` | bool | tbd | `false` |
| `-` | `JWKS_URL` | string | url | `""` |
| `-` | `SIGNER_CERTIFICATE`| string | path to cert | `""` |
| `-` | `SIGNER_PRIVATE_KEY`| string | path to private key of cert | `""` |
| `-` | `SIGNER_VALID_DURATION` | string | signed certificate expire in | `"87600h"` |
