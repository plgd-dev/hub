# Authorization server

## Description

Authorize access for users to devices.

### API

All requests to service must contains valid access token in [grpc metadata](https://github.com/grpc/grpc-go/blob/master/Documentation/grpc-auth-support.md#oauth2).

#### Commands

- sign up - exchange authorization code for opaque token
- sign in - validate access token of the device
- sign out - invalidate access token of the device
- sign off - remove device fron DB and invalidate all credendtials
- refresh token - refresh access token with refresh token
- get user devices - returns list of users devices

#### Contract

- [service](https://github.com/plgd-dev/cloud/blob/master/authorization/pb/service.proto)
- [requets/responses](https://github.com/plgd-dev/cloud/blob/master/authorization/pb/auth.proto)

## Docker Image

```bash
docker pull plgd/authorization:vnext
```

## Configuration

| Option | ENV variable | Type | Description | Default |
| ------ | --------- | ----------- | ------- | ------- |
| `-` | `ADDRESS` | string | `listen address` | `"0.0.0.0:9100"` |
| `-` | `DEVICE_PROVIDER` | string | `value which comes from the device during the sign-up ("apn")` | `"github"` |
| `-` | `DEVICE_OAUTH_CLIENT_ID` | string | `client id for authentication to get access token/authorization code` | `""` |
| `-` | `DEVICE_OAUTH_CLIENT_SECRET` | string | `client id for authentication to get access token` |  `""` |
| `-` | `DEVICE_OAUTH_REDIRECT_URL` | string | `redirect url used to obtain device access token` | `""` |
| `-` | `DEVICE_OAUTH_ENDPOINT_AUTH_URL` | string | `authorization endpoint` | `""` |
| `-` | `DEVICE_OAUTH_ENDPOINT_TOKEN_URL` | string | `token endpoint` | `""` |
| `-` | `DEVICE_OAUTH_SCOPES` | string | `Comma separated list of required scopes` | `""` |
| `-` | `DEVICE_OAUTH_RESPONSE_MODE` | string | `one of "query/post_form"` | `"query"` |
| `-` | `SDK_OAUTH_CLIENT_ID` | string | `client id for authentication to get access token` | `""` |
| `-` | `SDK_OAUTH_REDIRECT_URL` | string | `redirect url used to obtain access token` | `""` |
| `-` | `SDK_OAUTH_ENDPOINT_AUTH_URL` | string | `authorization endpoint` | `""` |
| `-` | `SDK_OAUTH_AUDIENCE` | string |  `refer to the resource servers that should accept the token` | `""` |
| `-` | `SDK_OAUTH_SCOPES` | string | `Comma separated list of required scopes` | `""` |
| `-` | `SDK_OAUTH_RESPONSE_MODE` | string | `one of "query/post_form"`| `"query"` |
| `-` | `LISTEN_TYPE` | string | `defines how to obtain listen TLS certificates - options: acme|file` | `"acme"` |
| `-` | `LISTEN_ACME_CA_POOL` | string | `path to pem file of CAs` | `""` |
| `-` | `LISTEN_ACME_DIRECTORY_URL` | string |  `url of acme directory` | `""` |
| `-` | `LISTEN_ACME_DOMAINS` | string | `list of domains for which will be in certificate provided from acme` | `""` |
| `-` | `LISTEN_ACME_REGISTRATION_EMAIL` | string | `registration email for acme` | `""` |
| `-` | `LISTEN_ACME_TICK_FREQUENCY` | string | `interval of validate certificate` | `""` |
| `-` | `LISTEN_ACME_USE_SYSTEM_CERTIFICATION_POOL` | bool | `load CAs from system` | `false` |
| `-` | `LISTEN_FILE_CA_POOL` | string | `path to pem file of CAs` |  `""` |
| `-` | `LISTEN_FILE_CERT_KEY_NAME` | string | `name of pem certificate key file` | `""` |
| `-` | `LISTEN_FILE_CERT_DIR_PATH` | string | `path to directory which contains LISTEN_FILE_CERT_KEY_NAME and LISTEN_FILE_CERT_NAME` | `""` |
| `-` | `LISTEN_FILE_CERT_NAME` | string | `name of pem certificate file` | `""` |
| `-` | `LISTEN_FILE_USE_SYSTEM_CERTIFICATION_POOL` | bool | `load CAs from system` | `false` |
| `-` | `LOG_ENABLE_DEBUG` | bool | `enable debugging message` | `false` |
| `-` | `MONGODB_URI` | string | `uri to mongo database` | `"mongodb://localhost:27017"` |
| `-` | `MONGODB_DATABASE` | string | `name of database` | `"authorization"` |
| `-` | `LOG_ENABLE_DEBUG` | bool | `debug logging` | `false` |
