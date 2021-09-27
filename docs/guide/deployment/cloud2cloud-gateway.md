# Cloud2Cloud Gateway

## Description

Provides devices to another cloud.

## Docker Image

```bash
docker pull plgd/cloud2cloud-gateway:latest
```

## Docker Run

### How to make certificates

Before you run the docker image of plgd/cloud2cloud-gateway, you make sure that certificates exist in `.tmp/certs` folder.
If they do not exist, you can create certificates from the plgd/bundle image by executing the following step once:

```bash
# Create a local folder for certificates and run the plgd/bundle image to execute a shell.
mkdir -p $(pwd).tmp/certs
docker run -it \
  --network=host \
  -v $(pwd)/.tmp/certs:/certs \
  -e CLOUD_SID=00000000-0000-0000-0000-000000000001 \
  --entrypoint /bin/bash \
  plgd/bundle:latest

# Copy & paste the commands below to the bash shell of the plgd/bundle container.
certificate-generator --cmd.generateRootCA --outCert=/certs/root_ca.crt --outKey=/certs/root_ca.key --cert.subject.cn=RootCA
certificate-generator --cmd.generateCertificate --outCert=/certs/http.crt --outKey=/certs/http.key --cert.subject.cn=localhost --cert.san.domain=localhost --signerCert=/certs/root_ca.crt --signerKey=/certs/root_ca.key

# Exit the shell.
exit
```

```bash
# See common certificates for plgd cloud services.
ls .tmp/certs
http.crt  http.key  root_ca.crt  root_ca.key
```

### How to get configuration file

A configuration template is available in [cloud2cloud-gateway/config.yaml](https://github.com/plgd-dev/cloud/blob/v2/cloud2cloud-gateway/config.yaml).
You can also see the `config.yaml` configuration file in the `cloud2cloud-gateway` folder by downloading `git clone https://github.com/plgd-dev/cloud.git`.

```bash
# Copy & paste the configuration template from the link and save the file as `cloud2cloud-gateway.yaml` to a local folder.
vi cloud2cloud-gateway.yaml

# Or download the configuration template with curl.
curl https://github.com/plgd-dev/cloud/blob/v2/cloud2cloud-gateway/config.yaml --output cloud2cloud-gateway.yaml
```

### Edit configuration file

You can edit values in the configuration file such as server port, certificates, OAuth provider and so on.
Read more details about how to configure the OAuth Provider [here](https://github.com/plgd-dev/cloud/blob/v2/docs/guide/developing/authorization.md#how-to-configure-auth0).

The following example shows configuration of address, tls, event bus and service clients.

```yaml
...
apis:
  http:
    address: "0.0.0.0:9086"
    tls:
      caPool: "/data/certs/root_ca.crt"
      keyFile: "/data/certs/http.key"
      certFile: "/data/certs/http.crt"
      clientCertificateRequired: false
    authorization:
      authority: "https://auth.example.com/authorize"
      audience: "https://api.example.com"
      http:
        tls:
          caPool: "/data/certs/root_ca.crt"
          keyFile: "/data/certs/http.key"
          certFile: "/data/certs/http.crt"
...
clients:
  eventBus:
    nats:
      url: "nats://localhost:4222"
      tls:
        caPool: "/data/certs/root_ca.crt"
        keyFile: "/data/certs/http.key"
        certFile: "/data/certs/http.crt"
...
  resourceAggregate:
    grpc:
      address: "localhost:9083"
      tls:
        caPool: "/data/certs/root_ca.crt"
        keyFile: "/data/certs/http.key"
        certFile: "/data/certs/http.crt"
...
  resourceDirectory:
    grpc:
      address: "localhost:9082"
      tls:
        caPool: "/data/certs/root_ca.crt"
        keyFile: "/data/certs/http.key"
        certFile: "/data/certs/http.crt"
...
```

### Run docker image

You can run the plgd/cloud2cloud-gateway image using certificates and configuration file in the folder you made certificates in.

```bash
docker run -d --network=host \
  --name=cloud2cloud-gateway \
  -v $(pwd)/.tmp/certs:/data/certs \
  -v $(pwd)/cloud2cloud-gateway.yaml:/data/cloud2cloud-gateway.yaml \
  plgd/cloud2cloud-gateway:latest --config=/data/cloud2cloud-gateway.yaml
```

## YAML Configuration

### Logging

| Property | Type | Description | Default |
| ---------- | -------- | -------------- | ------- |
| `log.debug` | bool | `Set to true if you would like to see extra information in logs.` | `false` |

### HTTP API

| Property | Type | Description | Default |
| ---------- | -------- | -------------- | ------- |
| `apis.http.address` | string | `Listen specification <host>:<port> for http client connection.` | `"0.0.0.0:9100"` |
| `apis.http.tls.caPool` | string | `File path to the root certificate in PEM format which might contain multiple certificates in a single file.` |  `""` |
| `apis.http.tls.keyFile` | string | `File path to private key in PEM format.` | `""` |
| `apis.http.tls.certFile` | string | `File path to certificate in PEM format.` | `""` |
| `apis.http.tls.clientCertificateRequired` | bool | `If true, require client certificate.` | `true` |
| `apis.http.authorization.authority` | string | `Authority is the address of the token-issuing authentication server. Services will use this URI to find and retrieve the public key that can be used to validate the token’s signature.` | `""` |
| `apis.http.authorization.audience` | string | `Identifier of the API configured in your OAuth provider.` | `""` |
| `apis.http.authorization.http.maxIdleConns` | int | `It controls the maximum number of idle (keep-alive) connections across all hosts. Zero means no limit.` | `16` |
| `apis.http.authorization.http.maxConnsPerHost` | int | `It optionally limits the total number of connections per host, including connections in the dialing, active, and idle states. On limit violation, dials will block. Zero means no limit.` | `32` |
| `apis.http.authorization.http.maxIdleConnsPerHost` | int | `If non-zero, controls the maximum idle (keep-alive) connections to keep per-host. If zero, DefaultMaxIdleConnsPerHost is used.` | `16` |
| `apis.http.authorization.http.idleConnTimeout` | string | `The maximum amount of time an idle (keep-alive) connection will remain idle before closing itself. Zero means no limit.` | `30s` |
| `apis.http.authorization.http.timeout` | string | `A time limit for requests made by this Client. A Timeout of zero means no timeout.` | `10s` |
| `apis.http.authorization.http.tls.caPool` | string | `File path to the root certificate in PEM format which might contain multiple certificates in a single file.` |  `""` |
| `apis.http.authorization.http.tls.keyFile` | string | `File path to private key in PEM format.` | `""` |
| `apis.http.authorization.http.tls.certFile` | string | `File path to certificate in PEM format.` | `""` |
| `apis.http.authorization.http.tls.useSystemCAPool` | bool | `If true, use system certification pool.` | `false` |

### Event Bus

Plgd cloud uses NATS messaging system as an event bus.

| Property | Type | Description | Default |
| ---------- | -------- | -------------- | ------- |
| `clients.eventBus.nats.url` | string | `URL to nats messaging system.` | `"nats://localhost:4222"` |
| `clients.eventBus.nats.pendingLimits.msgLimit` | int | `Limit number of messages in queue. -1 means unlimited` | `524288` |
| `clients.eventBus.nats.pendingLimits.bytesLimit` | int | `Limit buffer size of queue. -1 means unlimited` | `67108864` |
| `clients.eventBus.nats.tls.caPool` | string | `root certificate the root certificate in PEM format.` |  `""` |
| `clients.eventBus.nats.tls.keyFile` | string | `File name of private key in PEM format.` | `""` |
| `clients.eventBus.nats.tls.certFile` | string | `File name of certificate in PEM format.` | `""` |
| `clients.eventBus.nats.tls.useSystemCAPool` | bool | `If true, use system certification pool.` | `false` |

### GRPC Gateway Client

Client configurations to internally connect to GRPC Gateway service.

| Property | Type | Description | Default |
| ---------- | -------- | -------------- | ------- |
| `clients.grpcGateway.grpc.address` | string | `GRPC Gateway service address.` | `"127.0.0.1:9100"` |
| `clients.grpcGateway.grpc.tls.caPool` | string | `File path to the root certificate in PEM format which might contain multiple certificates in a single file.` |  `""` |
| `clients.grpcGateway.grpc.tls.keyFile` | string | `File path to private key in PEM format.` | `""` |
| `clients.grpcGateway.grpc.tls.certFile` | string | `File path to certificate in PEM format.` | `""` |
| `clients.grpcGateway.grpc.tls.useSystemCAPool` | bool | `If true, use system certification pool.` | `false` |
| `clients.grpcGateway.grpc.keepAlive.time` | string | `After a duration of this time if the client doesn't see any activity it pings the server to see if the transport is still alive.` | `10s` |
| `clients.grpcGateway.grpc.keepAlive.timeout` | string | `After having pinged for keepalive check, the client waits for a duration of Timeout and if no activity is seen even after that the connection is closed.` | `20s` |
| `clients.grpcGateway.grpc.keepAlive.permitWithoutStream` | bool | `If true, client sends keepalive pings even with no active RPCs. If false, when there are no active RPCs, Time and Timeout will be ignored and no keepalive pings will be sent.` | `false` |

### Resource Aggregate Client

Client configurations to internally connect to the Resource Aggregate service.

| Property | Type | Description | Default |
| ---------- | -------- | -------------- | ------- |
| `clients.resourceAggregate.grpc.address` | string | `Resource aggregate service address.` | `"127.0.0.1:9100"` |
| `clients.resourceAggregate.grpc.keepAlive.time` | string | `After a duration of this time if the client doesn't see any activity it pings the server to see if the transport is still alive.` | `10s` |
| `clients.resourceAggregate.grpc.keepAlive.timeout` | string | `After having pinged for keepalive check, the client waits for a duration of Timeout and if no activity is seen even after that the connection is closed.` | `20s` |
| `clients.resourceAggregate.grpc.keepAlive.permitWithoutStream` | bool | `If true, client sends keepalive pings even with no active RPCs. If false, when there are no active RPCs, Time and Timeout will be ignored and no keepalive pings will be sent.` | `false` |
| `clients.resourceAggregate.grpc.tls.caPool` | string | `File path to the root certificate in PEM format which might contain multiple certificates in a single file.` |  `""` |
| `clients.resourceAggregate.grpc.tls.keyFile` | string | `File path to private key in PEM format.` | `""` |
| `clients.resourceAggregate.grpc.tls.certFile` | string | `File path to certificate in PEM format.` | `""` |
| `clients.resourceAggregate.grpc.tls.useSystemCAPool` | bool | `If true, use system certification pool.` | `false` |

### Storage

Plgd cloud uses MongoDB database as the owner's device store.

| Property | Type | Description | Default |
| ---------- | -------- | -------------- | ------- |
| `clients.storage.mongoDB.uri` | string | `URI to mongo database.` | `"mongodb://localhost:27017"` |
| `clients.storage.mongoDB.database` | string | `Name of database.` | `"ownersDevices"` |
| `clients.storage.mongoDB.tls.caPool` | string | `File path to the root certificate in PEM format which might contain multiple certificates in a single file.` |  `""` |
| `clients.storage.mongoDB.tls.keyFile` | string | `File path to private key in PEM format.` | `""` |
| `clients.storage.mongoDB.tls.certFile` | string | `File path to certificate in PEM format.` | `""` |
| `clients.storage.mongoDB.tls.useSystemCAPool` | bool | `If true, use system certification pool.` | `false` |

### Subscription

| Property | Type | Description | Default |
| ---------- | -------- | -------------- | ------- |
| `clients.subscription.http.reconnectInterval` | string | `try to reconnect after interval to resource-directory when connection was closed` | `"10s"` |
| `clients.subscription.http.emitEventTimeout` | string | `timeout for send event` | `"5s"` |
| `clients.subscription.http.tls.caPool` | string | `File path to the root certificate in PEM format which might contain multiple certificates in a single file.` |  `""` |
| `clients.subscription.http.tls.keyFile` | string | `File path to private key in PEM format.` | `""` |
| `clients.subscription.http.tls.certFile` | string | `File path to certificate in PEM format.` | `""` |
| `clients.subscription.http.tls.useSystemCAPool` | bool | `If true, use system certification pool.` | `false` |

### Task Queue

| Property | Type | Description | Default |
| ---------- | -------- | -------------- | ------- |
| `taskQueue.goPoolSize` | int | `Maximum number of running goroutine instances.` | `1600` |
| `taskQueue.size` | int | `Size of queue. If it exhausted, submit returns error.` | `2097152` |
| `taskQueue.maxIdleTime` | string | `Sets up the interval time of cleaning up goroutines. Zero means never cleanup.` | `10m` |

> Note that the string type related to time (i.e. timeout, idleConnTimeout, expirationTime) is decimal numbers, each with optional fraction and a unit suffix, such as "300ms", "1.5h" or "2h45m". Valid time units are "ns", "us", "ms", "s", "m", "h".

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

#### Authorize with try.plgd.cloud

```bash
https://auth.plgd.cloud/authorize?
    response_type=code&
    client_id=9XjK2mCf2J0or4Ko0ow7wCmZeDTjC1mW&
    redirect_uri=http://localhost:8080/callback&
    scope=r:deviceinformation:* r:resources:* w:resources:* w:subscriptions:* offline_access&
    audience=https://openapi.try.plgd.cloud/&
    state=STATE
```

If all goes well, you'll receive an HTTP 302 response. The authorization code is included at the end of the URL:

```url
http://localhost:8080/callback?code=s65bpdt-ry7QEh6O&state=STATE
```

### Request Tokens

Now that you have an Authorization Code, you must exchange it for tokens. Using the extracted Authorization Code (code) from the previous step, you will need to POST to the token URL.

#### Request Tokens with try.plgd.cloud

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

To be able to see the devices through the `try.plgd.cloud` C2C API, first you need to onboard the device. When you have your device ready, go to the `https://plgd.dev` website and click `Try Live`. This redirects you to the `try.plgd.cloud Portal`.

First thing you need is an authorization code. In the `try.plgd.cloud Portal` go to `Devices` and click `Onboard Device`. This displays you the code needed to onboard the device. Values which should be set to the [coapcloudconf](https://github.com/openconnectivityfoundation/cloud-services/blob/c2c/swagger2.0/oic.r.coapcloudconf.swagger.json) device resource are:

#### Unsecured device

- `apn` : `plgd`
- `cis` : `coap+tcp://try.plgd.cloud:5683`
- `sid` : `adebc667-1f2b-41e3-bf5c-6d6eabc68cc6`
- `at` : `CODE_FROM_PORTAL`

#### Secured device

- `apn` : `plgd`
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
