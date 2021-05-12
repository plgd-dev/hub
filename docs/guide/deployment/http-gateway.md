# HTTP gateway
Http gateway exposes REST API for devices, cloud configuration, certificate authority and websockets for events as well as user web UI (i.e. PLGD Dashboard).

## Docker Image

```bash
docker pull plgd/http-gateway:latest
```

## Docker Run
### How to make certificates
Before you run docker image of plgd/http-gateway, you make sure certificates exists on `.tmp/certs` folder.
If not exists, you can create certificates from plgd/bundle image by following step only once.
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

# Exit shell.
exit
```
```bash
# See common certificates for plgd cloud services.
ls .tmp/certs
http.crt	http.key	root_ca.crt	root_ca.key
```

### How to get configuration file
A configuration template is available on [http-gateway/config.yaml](https://github.com/plgd-dev/cloud/blob/v2/http-gateway/config.yaml).
You can also see `config.yaml` configuration file on the `http-gateway` folder by downloading `git clone https://github.com/plgd-dev/cloud.git`.
```bash
# Copy & paste configuration template from the link and save the file named `http-gateway.yaml` on the local folder.
vi http-gateway.yaml

# Or download configuration template.
curl https://github.com/plgd-dev/cloud/blob/v2/http-gateway/config.yaml --output http-gateway.yaml
```

### Edit configuration file 
You can edit configuration file such as server port, certificates, OAuth provider and so on.
Read more detail about how to configure OAuth Provider [here](https://github.com/plgd-dev/cloud/blob/v2/docs/guide/developing/authorization.md#how-to-configure-auth0). 

See an example of address, tls, event bus and service clients config on the followings.
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
  certificateAuthority:
    grpc:
      address: "localhost:9087"
      tls:
        caPool: "/data/certs/root_ca.crt"
        keyFile: "/data/certs/http.key"
        certFile: "/data/certs/http.crt"
...
ui:
  enabled: true
  directory: "/usr/local/var/www"
  oauthClient:
    domain: "auth.example.com"
    clientID: "412dsFf53Sj6$"
    audience: "https://api.example.com"
    scope: "openid,offline_access"
    httpGatewayAddress: "https://www.example.com"
```

### Run docker image 
You can run plgd/http-gateway image using certificates and configuration file on the folder you made certificates.
```bash
docker run -d --network=host \
	--name=http-gateway \
	-v $(pwd)/.tmp/certs:/data/certs \
	-v $(pwd)/http-gateway.yaml:/data/http-gateway.yaml \
	plgd/http-gateway:latest --config=/data/http-gateway.yaml
```

## YAML Configuration
### Logging

| Property | Type | Description | Default |
| ---------- | -------- | -------------- | ------- |
| `log.debug` | bool | `Set to true if you would like to see extra information on logs.` | `false` |

### HTTP API
HTTP API of the Http Gateway Service as defined [uri](https://github.com/plgd-dev/cloud/blob/v2/http-gateway/uri/uri.go) and [swagger](https://petstore.swagger.io/?url=https://raw.githubusercontent.com/plgd-dev/cloud/v2/http-gateway/swagger.yaml) for REST API.

| Property | Type | Description | Default |
| ---------- | -------- | -------------- | ------- |
| `api.http.address` | string | `Listen specification <host>:<port> for http client connection.` | `"0.0.0.0:9100"` |
| `api.http.tls.caPool` | string | `File path to the root certificate in PEM format which might contain multiple certificates in a single file.` |  `""` |
| `api.http.tls.keyFile` | string | `File path to private key in PEM format.` | `""` |
| `api.http.tls.certFile` | string | `File path to certificate in PEM format.` | `""` |
| `api.http.tls.clientCertificateRequired` | bool | `If true, require client certificate.` | `true` |
| `api.http.websocket.readLimit` | int | `The maximum size in bytes for a message read from the peer. If a message exceeds the limit, the connection sends a close message to the peer` | `8192` |
| `api.http.websocket.readTimeout` | string | `The read deadline on the underlying network connection. A zero value means reads will not time out.` | `4s` |
| `api.http.authorization.authority` | string | `Endpoint of OAuth provider.` | `""` |
| `api.http.authorization.audience` | string | `Identifier of the API configured in your OAuth provider.` | `""` |
| `api.http.authorization.http.maxIdleConns` | int | `It controls the maximum number of idle (keep-alive) connections across all hosts. Zero means no limit.` | `16` |
| `api.http.authorization.http.maxConnsPerHost` | int | `It optionally limits the total number of connections per host, including connections in the dialing, active, and idle states. On limit violation, dials will block. Zero means no limit.` | `32` |
| `api.http.authorization.http.maxIdleConnsPerHost` | int | `If non-zero, controls the maximum idle (keep-alive) connections to keep per-host. If zero, DefaultMaxIdleConnsPerHost is used.` | `16` |
| `api.http.authorization.http.idleConnTimeout` | string | `The maximum amount of time an idle (keep-alive) connection will remain idle before closing itself. Zero means no limit.` | `30s` |
| `api.http.authorization.http.timeout` | string | `A time limit for requests made by this Client. A Timeout of zero means no timeout.` | `10s` |
| `api.http.authorization.http.tls.caPool` | string | `File path to the root certificate in PEM format which might contain multiple certificates in a single file.` |  `""` |
| `api.http.authorization.http.tls.keyFile` | string | `File path to private key in PEM format.` | `""` |
| `api.http.authorization.http.tls.certFile` | string | `File path to certificate in PEM format.` | `""` |
| `api.http.authorization.http.tls.useSystemCAPool` | bool | `If true, use system certification pool.` | `false` |

### Event Bus
Plgd cloud uses NATS messaging system as a event bus.

| Property | Type | Description | Default |
| ---------- | -------- | -------------- | ------- |
| `clients.eventBus.goPoolSize` | int | `Number of routines to process events in projection.` | `16` |
| `clients.eventBus.nats.url` | string | `URL to nats messaging system.` | `"nats://localhost:4222"` |
| `clients.eventBus.nats.tls.caPool` | string | `root certificate the root certificate in PEM format.` |  `""` |
| `clients.eventBus.nats.tls.keyFile` | string | `File name of private key in PEM format.` | `""` |
| `clients.eventBus.nats.tls.certFile` | string | `File name of certificate in PEM format.` | `""` |
| `clients.eventBus.nats.tls.useSystemCAPool` | bool | `If true, use system certification pool.` | `false` |

### Resource Aggregate Client
Resource aggregate client configuration to connect internally.

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

### Resource Directory Client
Resource directory client configuration to connect internally.

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

### Certificate Authority Client
Certificate authority client configuration to connect internally.

| Property | Type | Description | Default |
| ---------- | -------- | -------------- | ------- |
| `clients.certificateAuthority.enabled` | bool | `If true, connect to certificate authority.` | `"false"` |
| `clients.certificateAuthority.grpc.address` | string | `Certificate authority service address.` | `"127.0.0.1:9100"` |
| `clients.certificateAuthority.grpc.tls.caPool` | string | `File path to the root certificate in PEM format which might contain multiple certificates in a single file.` |  `""` |
| `clients.certificateAuthority.grpc.tls.keyFile` | string | `File path to private key in PEM format.` | `""` |
| `clients.certificateAuthority.grpc.tls.certFile` | string | `File path to certificate in PEM format.` | `""` |
| `clients.certificateAuthority.grpc.tls.useSystemCAPool` | bool | `If true, use system certification pool.` | `false` |
| `clients.certificateAuthority.grpc.keepAlive.time` | string | `After a duration of this time if the client doesn't see any activity it pings the server to see if the transport is still alive.` | `10s` |
| `clients.certificateAuthority.grpc.keepAlive.timeout` | string | `After having pinged for keepalive check, the client waits for a duration of Timeout and if no activity is seen even after that the connection is closed.` | `20s` |
| `clients.certificateAuthority.grpc.keepAlive.permitWithoutStream` | bool | `If true, client sends keepalive pings even with no active RPCs. If false, when there are no active RPCs, Time and Timeout will be ignored and no keepalive pings will be sent.` | `false` |

### User Web UI
These configurations are for `PLGD Dashboard` as described in [here](https://github.com/plgd-dev/cloud/blob/v2/docs/guide/developing/dashboard.md).

| Property | Type | Description | Default |
| ---------- | -------- | -------------- | ------- |
| `ui.enabled` | string | `Set to true if you would like to run user web UI.` | `false` |
| `ui.directory` | string | `Path to default web ui built by nodejs` | `"/usr/local/var/www"` |
| `ui.oauthClient.domain` | string | `Domain address of OAuth Provider.` | `""` |
| `ui.oauthClient.clientID` | string | `Client ID to exchange an authorization code for an access token.` | `""` |
| `ui.oauthClient.audience` | string | `Identifier of the API configured in your OAuth provider.` | `""` |
| `ui.oauthClient.scopes` | string | `Comma separated list of required scopes.` | `""` |
| `ui.oauthClient.httpGatewayAddress` | string | `External address of Http gateway service.` | `""` |

> Note that the string type related to time (i.e. timeout, idleConnTimeout, expirationTime) is decimal numbers, each with optional fraction and a unit suffix, such as "300ms", "1.5h" or "2h45m". Valid time units are "ns", "us", "ms", "s", "m", "h".
