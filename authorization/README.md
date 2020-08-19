[![GoDoc](https://godoc.org/github.com/plgd-dev/cloud/authorization?status.svg)](https://godoc.org/github.com/plgd-dev/cloud/authorization)
[![Go Report Card](https://goreportcard.com/badge/plgd-dev/authorization)](https://goreportcard.com/report/plgd-dev/authorization)
[![Gitter](https://badges.gitter.im/ocfcloud/Lobby.svg)](https://gitter.im/ocfcloud/Lobby?utm_source=badge&utm_medium=badge&utm_campaign=pr-badge)

# Authorization

- [Specification](https://wiki.iotivity.org/coapnativecloud#authorization_bounded_context)

## Authorization Web

The authorization web provides forms for the following manual actions:
- Obtain **Authorization Code** from GitHub 
- Demonstrate the exchange for an **Access Token**

Run the web server as follows and open http://127.0.0.1:7000/ in your browser.

```bash
docker build . --network=host -t authorization:web --target web
docker run --network=host authorization:web
```

The application can be configured using [environment variables](web/config.go)

## Authorization Service

The authorization service exposes Protobuf via HTTP/1.1 ([Open API](openapi.yaml)).

```bash
docker build . --network=host -t authorization:service --target service
docker run -e CLIENT_ID='my id' -e CLIENT_SECRET='my secret' --network=host authorization:service
```

The application can be configured using [environment variables](service/config.go)

To obtain the **client ID** and **secret**, register your application at 
[GitHub](https://github.com/settings/applications)
and set the callback to `/oauth_callback` (e.g. `http://127.0.0.1:7000/oauth_callback`).

# Limitations

This reference implementation lacks the following features:
- Shared access to devices by mutliple users
- Deletion of expired tokens
- Encryption in transit and at rest
- Clustered deployment

# Build

## Docker

```sh
make build-servicecontainer
```
## Local machine

```sh
go build ./cmd/service/
```

## Configuration
| Option | ENV variable | Type | Description | Default |
| ------ | --------- | ----------- | ------- | ------- |
| `-` | `ADDRESS` | string | tbd | `"0.0.0.0:9100"` |
| `-` | `DEVICE_PROVIDER` | string | `value which comes from the device during the sign-up ("apn")` | `"github"` |
| `-` | `DEVICE_OAUTH_CLIENT_ID` | string | tbd | `""` |
| `-` | `DEVICE_OAUTH_CLIENT_SECRET` | string | tbd | `""` |
| `-` | `DEVICE_OAUTH_REDIRECT_URL` | string | tbd | `""` |
| `-` | `DEVICE_OAUTH_ENDPOINT_AUTH_URL` | string | tbd | `""` |
| `-` | `DEVICE_OAUTH_ENDPOINT_TOKEN_URL` | string | tbd | `""` |
| `-` | `DEVICE_OAUTH_SCOPES` | string | Comma separated list of required scopes | `""` |
| `-` | `DEVICE_OAUTH_RESPONSE_MODE` | string | one of "query/post_form" | `"query"` |
| `-` | `SDK_OAUTH_CLIENT_ID` | string | tbd | `""` |
| `-` | `SDK_OAUTH_REDIRECT_URL` | string | tbd | `""` |
| `-` | `SDK_OAUTH_ENDPOINT_AUTH_URL` | string | tbd | `""` |
| `-` | `SDK_OAUTH_AUDIENCE` | string | tbd | `""` |
| `-` | `SDK_OAUTH_SCOPES` | string | Comma separated list of required scopes | `""` |
| `-` | `SDK_OAUTH_RESPONSE_MODE` | string | one of "query/post_form" | `"query"` |
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
| `-` | `MONGODB_URI` | string | tbd | `"mongodb://localhost:27017"` |
| `-` | `MONGODB_DATABASE` | string | tbd | `"authorization"` |
