# Cloud2Cloud Gateway

## Description

Provides devices to another cloud.

## Docker Image

```bash
docker pull plgd/cloud2cloud-gateway:vnext
```

## API

Follow [OCF Cloud API For Cloud Services Specification](https://openconnectivity.org/specs/OCF_Cloud_API_For_Cloud_Services_Specification_v2.2.0.pdf)

### Commands

- get all devices
- get the device by ID
- retrieve / update resource values
- subscribe to / unsubscribe from events against the set of devices
- subscribe to / unsubscribe from events against a specific device
- subscribe to / unsubscribe from  events against a specific resource
- [swagger](https://petstore.swagger.io/?url=https://raw.githubusercontent.com/openconnectivityfoundation/core-extensions/ocfcloud-openapi/swagger2.0/oic.r.cloudopenapi.swagger.json)

## How to try

### Steps

1. Authorize the user: Request the user's authorization and redirect back to your application with an authorization code.
2. Request tokens: Exchange your authorization code for tokens.
3. Call your API: Use the retrieved Access Token to call your API.
4. Refresh Tokens: Use a Refresh Token to request new tokens when the existing ones expire.

### Authorize the User

- Authenticating the user;
- Redirecting the user to an Identity Provider to handle authentication;
- Checking for active Single Sign-on (SSO) sessions;
- Obtaining user consent for the requested permission level, unless consent has been previously given.

To authorize the user, your app must send the user to the authorization URL.

#### Try pluggedin.cloud

```bash
https://auth.plgd.cloud/authorize?
    response_type=code&
    client_id=9XjK2mCf2J0or4Ko0ow7wCmZeDTjC1mW&
    redirect_uri=http://localhost:8080/callback&
    scope=r:deviceinformation:* r:resources:* w:resources:* w:subscriptions:* offline_access&
    audience=https://openapi.try.plgd.cloud/&
    state=STATE
```

#### Response

If all goes well, you'll receive an HTTP 302 response. The authorization code is included at the end of the URL:

```url
http://localhost:8080/callback?code=s65bpdt-ry7QEh6O&state=STATE
```

### Request Tokens

Now that you have an Authorization Code, you must exchange it for tokens. Using the extracted Authorization Code (code) from the previous step, you will need to POST to the token URL.

#### Try pluggedin.cloud

```bash
curl --request POST \
  --url 'https://auth.plgd.cloud/oauth/token' \
  --header 'content-type: application/x-www-form-urlencoded' \
  --data grant_type=authorization_code \
  --data 'client_id=9XjK2mCf2J0or4Ko0ow7wCmZeDTjC1mW' \
  --data client_secret=UTeeIsSugTuDNbn4QMdBaNLDnMiBQzQaa6elm4SDuWOdZUou-aH00EPSbBhgppFD \
  --data code={YOUR_AUTHORIZATION_CODE} \
  --data 'redirect_uri=http://localhost:8080/callback'
```

#### Response

If all goes well, you'll receive an HTTP 200 response with a payload containing access_token, refresh_token, scope, expires_in and token_type values:

```json
{
  "access_token":"ey...ojg",
  "refresh_token":"pL...btL",
  "scope":"r:deviceinformation:* r:resources:* w:resources:* w:subscriptions:* offline_access",
  "expires_in":86400,
  "token_type":"Bearer"
}
```

### Call the C2C API

To call the C2C API as an authorized user, the application must pass the retrieved Access Token as a Bearer token in the Authorization header of your HTTP request.

```bash
curl --request GET \
  --url https://openapi.try.plgd.cloud/api/v1/devices \
  --header 'authorization: Bearer eyJ...lojg' \
  --header 'content-type: application/json' \
  --header 'accept: application/json'
```

### Refresh the token

You can use the Refresh Token to get a new Access Token. The application communicating with the C2C Endpoint needs a new Access Token only after the previous one expires. It's bad practice to call the endpoint to get a new Access Token every time you call an API, and pluggedin.cloud maintains rate limits that will throttle the amount of requests to the endpoint that can be executed using the same token from the same IP.

To refresh your token, make a POST request to the token endpoint, using grant_type=refresh_token.

```bash
curl --request POST \
  --url 'https://auth.plgd.cloud/oauth/token' \
  --header 'content-type: application/x-www-form-urlencoded' \
  --data grant_type=refresh_token \
  --data 'client_id=9XjK2mCf2J0or4Ko0ow7wCmZeDTjC1mW' \
  --data refresh_token={YOUR_REFRESH_TOKEN}
```

> Now you're able to authorize the user, request the token, communicate with the C2C API and refresh the token before it expires.

### Device Onboarding

To be able to see the devices through the `pluggedin.cloud` C2C API, first you need to onboard the device. When you have your device ready, go to the `https://pluggedin.cloud` and click `TRY`. This redirects you to the `pluggedin.cloud Portal`.

First thing you need is an authorization code. In the `pluggedin.cloud Portal` go to `Devices` and click `Onboard Device`. This displays you the code needed to onboard the device. Values which should be set to the [coapcloudconf](https://github.com/openconnectivityfoundation/cloud-services/blob/c2c/swagger2.0/oic.r.coapcloudconf.swagger.json) device resource are:

#### Unsecured device

- `apn` : `auth0`
- `cis` : `coap+tcp://try.plgd.cloud:5683`
- `sid` : `adebc667-1f2b-41e3-bf5c-6d6eabc68cc6`
- `at` : `CODE_FROM_PORTAL`

#### Secured device

- `apn` : `auth0`
- `cis` : `coaps+tcp://try.plgd.cloud:5684`
- `sid` : `adebc667-1f2b-41e3-bf5c-6d6eabc68cc6`
- `at` : `CODE_FROM_PORTAL`

Conditions:

- `Device must be owned.`
- `Cloud CA  must be set as TRUST CA with subject adebc667-1f2b-41e3-bf5c-6d6eabc68cc6 in device.`
- `Cloud CA in PEM:`

```pem
-----BEGIN CERTIFICATE-----
MIIBhDCCASmgAwIBAgIQdAMxveYP9Nb48xe9kRm3ajAKBggqhkjOPQQDAjAxMS8w
LQYDVQQDEyZPQ0YgQ2xvdWQgUHJpdmF0ZSBDZXJ0aWZpY2F0ZXMgUm9vdCBDQTAe
Fw0xOTExMDYxMjAzNTJaFw0yOTExMDMxMjAzNTJaMDExLzAtBgNVBAMTJk9DRiBD
bG91ZCBQcml2YXRlIENlcnRpZmljYXRlcyBSb290IENBMFkwEwYHKoZIzj0CAQYI
KoZIzj0DAQcDQgAEaNJi86t5QlZiLcJ7uRMNlcwIpmFiJf9MOqyz2GGnGVBypU6H
lwZHY2/l5juO/O4EH2s9h3HfcR+nUG2/tFzFEaMjMCEwDgYDVR0PAQH/BAQDAgEG
MA8GA1UdEwEB/wQFMAMBAf8wCgYIKoZIzj0EAwIDSQAwRgIhAM7gFe39UJPIjIDE
KrtyPSIGAk0OAO8txhow1BAGV486AiEAqszg1fTfOHdE/pfs8/9ZP5gEVVkexRHZ
JCYVaa2Spbg=
-----END CERTIFICATE-----
```

- `ACL for Cloud (Subject: adebc667-1f2b-41e3-bf5c-6d6eabc68cc6) must be set with full access to all published resources in device.`

## Configuration

| Option | ENV variable | Type | Description | Default |
| ------ | --------- | ----------- | ------- | ------- |
| `-` | `ADDRESS` | string | `listen address` | `"0.0.0.0:9100"` |
| `-` | `RESOURCE_DIRECTORY_ADDRESS` | string | `resource directory address` | `"127.0.0.1:9100"` |
| `-` | `JWKS_URL` | string | `url to get JSON Web Key` | `""` |
| `-` | `SERVICE_RECONNECT_INTERVAL` | string | `try to reconnect after interval to resource-directory when connection was closed` | `"10s"` |
| `-` | `SERVICE_OAUTH_ENDPOINT_TOKEN_URL` | string | `url to get service access token via OAUTH client credential flow` | `""` |
| `-` | `SERVICE_OAUTH_CLIENT_ID` | string | `client id for authentication to get access token` | `""` |
| `-` | `SERVICE_OAUTH_CLIENT_SECRET` | string | `secrest for authentication to get access token` | `""` |
| `-` | `SERVICE_OAUTH_AUDIENCE` | string | `refer to the resource servers that should accept the token` | `""` |
| `-` | `EMIT_EVENT_TIMEOUT` | string | `timeout for send event` | `"5s"` |
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
| `-` | `SUBSTORE_MONGO_HOST` | string | `host of mongo database - uri without scheme` | `"localhost:27017"` |
| `-` | `SUBSTORE_MONGO_DATABASE` | string | `name of database` | `"cloud2cloudGateway"` |
| `-` | `LOG_ENABLE_DEBUG` | bool | `debug logging` | `false` |
