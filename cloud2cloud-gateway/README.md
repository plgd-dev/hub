[![Go Report](https://goreportcard.com/badge/github.com/go-ocf/cloud/cloud2cloud-gateway)](https://goreportcard.com/report/github.com/go-ocf/cloud/cloud2cloud-gateway)
[![Gitter](https://badges.gitter.im/ocfcloud/Lobby.svg)](https://gitter.im/ocfcloud/Lobby?utm_source=badge&utm_medium=badge&utm_campaign=pr-badge)

# Cloud API for Cloud Services
## How to try?
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
```
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
```
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
