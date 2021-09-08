# CoAP Gateway

CoAP Gateway provides API for devices and clients following the [Open Connectivity Foundation](https://openconnectivity.org/) specification.

## Docker Image

```bash
docker pull plgd/coap-gateway:latest
```

## Docker Run

### How to make certificates

Before you run docker image of plgd/coap-gateway, you make sure certificates exists on `.tmp/certs` folder.
If the certificates don't exist, you can create them using plgd/bundle image and running the following commands.

```bash
# Create local folder for certificates and run plgd/bundle image to execute shell.
mkdir -p $(pwd).tmp/certs
docker run -it \
  --network=host \
  -v $(pwd)/.tmp/certs:/certs \
  -e CLOUD_SID=00000000-0000-0000-0000-000000000001 \
  --entrypoint /bin/bash \
  plgd/bundle:latest

# Copy & paste below commands on the bash shell of plgd/bundle container.
certificate-generator --cmd.generateRootCA --outCert=/certs/root_ca.crt --outKey=/certs/root_ca.key --cert.subject.cn=RootCA
certificate-generator --cmd.generateCertificate --outCert=/certs/http.crt --outKey=/certs/http.key --cert.subject.cn=localhost --cert.san.domain=localhost --signerCert=/certs/root_ca.crt --signerKey=/certs/root_ca.key
certificate-generator --cmd.generateIdentityCertificate=$CLOUD_SID --outCert=/certs/coap.crt --outKey=/certs/coap.key --cert.san.domain=localhost --signerCert=/certs/root_ca.crt --signerKey=/certs/root_ca.key

# Exit shell.
exit
```

```bash
# See common certificates for plgd cloud services.
ls .tmp/certs
coap.crt    coap.key    http.crt    http.key    root_ca.crt    root_ca.key
```

### How to get configuration file

A configuration template is available on [coap-gateway/config.yaml](https://github.com/plgd-dev/cloud/blob/v2/coap-gateway/config.yaml).
You can also see `config.yaml` configuration file in the `coap-gateway` folder by downloading `git clone https://github.com/plgd-dev/cloud.git`.

```bash
# Copy & paste configuration template from the link and save it as a file named coap-gateway.yaml in the local folder.
vi coap-gateway.yaml

# Or download configuration template.
curl https://github.com/plgd-dev/cloud/blob/v2/coap-gateway/config.yaml --output coap-gateway.yaml
```

### Edit configuration file

You can edit configuration file including server port, certificates, OAuth provider and so on.
Read more details about how to configure OAuth Provider [here](https://github.com/plgd-dev/cloud/blob/v2/docs/guide/developing/authorization.md#how-to-configure-auth0).

See an example of address, tls and OAuth config on the following snippets:

```yaml
...
apis:
  coap:
    address: "0.0.0.0:5684"
    externalAddress: "coap-gw.example.com:5684"
    tls:
      caPool: "/data/certs/root_ca.crt"
      keyFile: "/data/certs/coap.key"
      certFile: "/data/certs/coap.crt"
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
  authorizationServer:
    grpc:
      address: "localhost:9081"
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

You can run plgd/coap-gateway image using certificates and configuration file from the folder you made certificates in.

```bash
docker run -d --network=host \
  --name=coap-gateway \
  -v $(pwd)/.tmp/certs:/data/certs \
  -v $(pwd)/coap-gateway.yaml:/data/coap-gateway.yaml \
  plgd/coap-gateway:latest --config=/data/coap-gateway.yaml
```

## YAML Configuration

### Logging

| Property | Type | Description | Default |
| ---------- | -------- | -------------- | ------- |
| `log.debug` | bool | `Set to true if you would like to see extra information in logs.` | `false` |
| `log.dumpCoapMessages` | bool | `Set to true if you would like to dump raw messages.` | `false` |

### CoAP API

CoAP API as specified in the [Open Connectivity Foundation - Device to Cloud Services](https://openconnectivity.org/specs/OCF_Device_To_Cloud_Services_Specification_v2.2.3.pdf).

| Property | Type | Description | Default |
| ---------- | -------- | -------------- | ------- |
| `api.coap.address` | string | `Listen specification <host>:<port> for coap client connection.` | `"0.0.0.0:5684"` |
| `api.coap.externalAddress` | string | `External address including public domain/IP for coap client connection.` | `"coap-gw.example.com:5684"` |
| `api.coap.maxMessageSize` | int | `Max message size which can be send/receive via coap. i.e. 256*1024 = 262144 bytes.` | `262144` |
| `api.coap.ownerCacheExpiration` | string | `Time limit of how long to keep subscribed to device updates after last use of the given cache item.` | `1m` |
| `api.grpc.subscriptionCacheExpiration` | string | `Time limit of how long to keep initialized events in subscription cache for duplicating events.` | `1m` |
| `api.grpc.subscriptionBufferSize` | int | `The maximum buffer size for one events subscription.` | `1000` |
| `api.coap.goroutineSocketHeartbeat` | string | `Interval time to check live service.` | `4s` |
| `api.coap.keepAlive.timeout` | string | `Time limit to close inactive connection.` | `20s` |
| `api.coap.blockwiseTransfer.enabled` | bool | `If true, enable blockwise transfer of coap message.` | `true` |
| `api.coap.blockwiseTransfer.blockSize` | int | `Size of blockwise transfer block.` | `1024` |
| `api.coap.tls.enabled` | bool | `If true, require server certificate for ssl connection.` | `true` |
| `api.coap.tls.caPool` | string | `File path to the root certificate in PEM format which might contain multiple certificates in a single file.` |  `""` |
| `api.coap.tls.keyFile` | string | `File path to private key in PEM format.` | `""` |
| `api.coap.tls.certFile` | string | `File path to certificate in PEM format.` | `""` |
| `api.coap.tls.clientCertificateRequired` | bool | `If true, require client certificate.` | `true` |

#### OAuth2.0 Client

OAuth2.0 Client is used to issuing an authorization code used by the Onboarding Tool during the device provisioning which is then exchanged for an access token during the [SignUp operation](https://plgd.dev/architecture/component-overview/#cloud-registration).

| Property | Type | Description | Default |
| ---------- | -------- | -------------- | ------- |
| `api.coap.authorization.clientID` | string | `Client ID to exchange an authorization code for an access token.` | `""` |
| `api.coap.authorization.clientSecret` | string | `Client secret to exchange an authorization code for an access token.` |  `""` |
| `api.coap.authorization.scopes` | string | `Comma separated list of required scopes.` | `""` |
| `api.coap.authorization.authority` | string | `Endpoint of OAuth provider.` | `""` |
| `api.coap.authorization.audience` | string | `Audience of OAuth provider.` | `""` |
| `api.coap.authorization.redirectURL` | string | `Redirect url used to obtain device access token.` | `""` |
| `api.coap.authorization.http.maxIdleConns` | int | `It controls the maximum number of idle (keep-alive) connections across all hosts. Zero means no limit.` | `16` |
| `api.coap.authorization.http.maxConnsPerHost` | int | `It optionally limits the total number of connections per host, including connections in the dialing, active, and idle states. On limit violation, dials will block. Zero means no limit.` | `32` |
| `api.coap.authorization.http.maxIdleConnsPerHost` | int | `If non-zero, controls the maximum idle (keep-alive) connections to keep per-host. If zero, DefaultMaxIdleConnsPerHost is used.` | `16` |
| `api.coap.authorization.http.idleConnTimeout` | string | `The maximum amount of time an idle (keep-alive) connection will remain idle before closing itself. Zero means no limit.` | `30s` |
| `api.coap.authorization.http.timeout` | string | `A time limit for requests made by this Client. A Timeout of zero means no timeout.` | `10s` |
| `api.coap.authorization.http.tls.caPool` | string | `File path to the root certificate in PEM format which might contain multiple certificates in a single file.` |  `""` |
| `api.coap.authorization.http.tls.keyFile` | string | `File path to private key in PEM format.` | `""` |
| `api.coap.authorization.http.tls.certFile` | string | `File path to certificate in PEM format.` | `""` |
| `api.coap.authorization.http.tls.useSystemCAPool` | bool | `If true, use system certification pool.` | `false` |

::: tip Audience
You might have one client, but multiple APIs in the OAuth system. What you want to prevent is to be able to contact all the APIs of your system with one token. This audience allows you to request the token for a specific API. If you configure it to myplgdc2c.api in the Auth0, you have to set it here if you want to also validate it.
:::

### Event Bus

Plgd cloud uses NATS messaging system as an event bus.

| Property | Type | Description | Default |
| ---------- | -------- | -------------- | ------- |
| `clients.eventBus.goPoolSize` | int | `Number of routines to process events in projection.` | `16` |
| `clients.eventBus.nats.url` | string | `URL to nats messaging system.` | `"nats://localhost:4222"` |
| `clients.eventBus.nats.pendingLimits.msgLimit` | int | `Limit number of messages in queue. -1 means unlimited` | `524288` |
| `clients.eventBus.nats.pendingLimits.bytesLimit` | int | `Limit buffer size of queue. -1 means unlimited` | `67108864` |
| `clients.eventBus.nats.tls.caPool` | string | `root certificate the root certificate in PEM format.` |  `""` |
| `clients.eventBus.nats.tls.keyFile` | string | `File name of private key in PEM format.` | `""` |
| `clients.eventBus.nats.tls.certFile` | string | `File name of certificate in PEM format.` | `""` |
| `clients.eventBus.nats.tls.useSystemCAPool` | bool | `If true, use system certification pool.` | `false` |

### Authorization Server Client

Client configurations to internally connect to Authorization Server service.

| Property | Type | Description | Default |
| ---------- | -------- | -------------- | ------- |
| `clients.authorizationServer.ownerClaim` | string | `Claim used to identify owner of the device.` | `"sub"` |
| `clients.authorizationServer.grpc.address` | string | `Authorization service address.` | `"127.0.0.1:9100"` |
| `clients.authorizationServer.grpc.tls.caPool` | string | `File path to the root certificate in PEM format which might contain multiple certificates in a single file.` |  `""` |
| `clients.authorizationServer.grpc.tls.keyFile` | string | `File path to private key in PEM format.` | `""` |
| `clients.authorizationServer.grpc.tls.certFile` | string | `File path to certificate in PEM format.` | `""` |
| `clients.authorizationServer.grpc.tls.useSystemCAPool` | bool | `If true, use system certification pool.` | `false` |
| `clients.authorizationServer.grpc.keepAlive.time` | string | `After a duration of this time if the client doesn't see any activity it pings the server to see if the transport is still alive.` | `10s` |
| `clients.authorizationServer.grpc.keepAlive.timeout` | string | `After having pinged for keepalive check, the client waits for a duration of Timeout and if no activity is seen even after that the connection is closed.` | `20s` |
| `clients.authorizationServer.grpc.keepAlive.permitWithoutStream` | bool | `If true, client sends keepalive pings even with no active RPCs. If false, when there are no active RPCs, Time and Timeout will be ignored and no keepalive pings will be sent.` | `false` |

::: tip Audience
You might have one client, but multiple APIs in the OAuth system. What you want to prevent is to be able to contact all the APIs of your system with one token. This audience allows you to request the token for a specific API. If you configure it to myplgdc2c.api in the Auth0, you have to set it here if you want to also validate it.
:::

### Resource Aggregate Client

Client configurations to internally connect to Resource Aggregate service.

| Property | Type | Description | Default |
| ---------- | -------- | -------------- | ------- |
| `clients.resourceAggregate.grpc.address` | string | `Resource aggregate service address.` | `"127.0.0.1:9100"` |
| `clients.resourceAggregate.grpc.tls.caPool` | string | `File path to the root certificate in PEM format which might contain multiple certificates in a single file.` |  `""` |
| `clients.resourceAggregate.grpc.tls.keyFile` | string | `File path to private key in PEM format.` | `""` |
| `clients.resourceAggregate.grpc.tls.certFile` | string | `File path to certificate in PEM format.` | `""` |
| `clients.resourceAggregate.grpc.tls.useSystemCAPool` | bool | `If true, use system certification pool.` | `false` |
| `clients.resourceAggregate.grpc.keepAlive.time` | string | `After a duration of this time if the client doesn't see any activity it pings the server to see if the transport is still alive.` | `10s` |
| `clients.resourceAggregate.grpc.keepAlive.timeout` | string | `After having pinged for keepalive check, the client waits for a duration of Timeout and if no activity is seen even after that the connection is closed.` | `20s` |
| `clients.resourceAggregate.grpc.keepAlive.permitWithoutStream` | bool | `If true, client sends keepalive pings even with no active RPCs. If false, when there are no active RPCs, Time and Timeout will be ignored and no keepalive pings will be sent.` | `false` |
| `clients.resourceAggregate.deviceStatusExpiration.enabled` | bool | `If ture, refresh device online status in short time than sign token expires.` | `false` |
| `clients.resourceAggregate.deviceStatusExpiration.ExpiresIn` | bool | `Interval time to refresh device online status.` | `24h` |

### Resource Directory Client

Client configurations to internally connect to Resource Directory service.

| Property | Type | Description | Default |
| ---------- | -------- | -------------- | ------- |
| `clients.resourceDirectory.grpc.address` | string | `Resource directory service address.` | `"127.0.0.1:9100"` |
| `clients.resourceDirectory.grpc.tls.caPool` | string | `File path to the root certificate in PEM format which might contain multiple certificates in a single file.` |  `""` |
| `clients.resourceDirectory.grpc.tls.keyFile` | string | `File path to private key in PEM format.` | `""` |
| `clients.resourceDirectory.grpc.tls.certFile` | string | `File path to certificate in PEM format.` | `""` |
| `clients.resourceDirectory.grpc.tls.useSystemCAPool` | bool | `If true, use system certification pool.` | `false` |
| `clients.resourceDirectory.grpc.keepAlive.time` | string | `After a duration of this time if the client doesn't see any activity it pings the server to see if the transport is still alive.` | `10s` |
| `clients.resourceDirectory.grpc.keepAlive.timeout` | string | `After having pinged for keepalive check, the client waits for a duration of Timeout and if no activity is seen even after that the connection is closed.` | `20s` |
| `clients.resourceDirectory.grpc.keepAlive.permitWithoutStream` | bool | `If true, client sends keepalive pings even with no active RPCs. If false, when there are no active RPCs, Time and Timeout will be ignored and no keepalive pings will be sent.` | `false` |

### Task Queue

| Property | Type | Description | Default |
| ---------- | -------- | -------------- | ------- |
| `taskQueue.goPoolSize` | int | `Maximum number of running goroutine instances.` | `1600` |
| `taskQueue.size` | int | `Size of queue. If it exhausted, submit returns error.` | `2097152` |
| `taskQueue.maxIdleTime` | string | `Sets up the interval time of cleaning up goroutines. Zero means never cleanup.` | `10m` |

> Note that the string type related to time (i.e. timeout, idleConnTimeout, expirationTime) is decimal numbers, each with optional fraction and a unit suffix, such as "300ms", "1.5h" or "2h45m". Valid time units are "ns", "us", "ms", "s", "m", "h".
