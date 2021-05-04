# Resource directory

## Docker Image

```bash
docker pull plgd/resource-directory:v2next
```
Or by using source
```bash
# Dowonload github source
git clone https://github.com/plgd-dev/cloud.git 

# Build the source
cd cloud/ 
make build
```

## Docker Run
### How to make certificates
Before you run docker image of plgd/resource-directory, you make sure to execute below script only once. 
```bash
# Create certificates on the source
make certificates 
```

### How to get configuration file
A configuration template is available on [resource-directory/config.yaml](https://github.com/plgd-dev/cloud/blob/v2/resource-directory/config.yaml). You can also see configuration file via executing below script.  
```bash
# See config file on the source
cat resource-directory/conifg.yaml 
```

### Edit configuration file 
You can edit configuration file such as server port, certificates, oauth provider and so on.
Read more detail about how to configure OAuth Provider [here](https://github.com/plgd-dev/cloud/blob/v2/docs/guide/developing/authorization.md#how-to-configure-auth0). 

See an example of tls config on the followings.
```yaml
...
    tls:
      caPool: "/data/certs/rootca.crt"
      keyFile: "/data/certs/http.key"
      certFile: "/data/certs/http.crt"
...
```

### Run docker image 
You can run plgd/resource-directory image using certificates and configuration file on the source directory of resource-directory.
```bash
docker run -d --network=host \
	--name=resource-directory \
	-v $(shell pwd)/../.tmp/certs:/data/certs \
	-v $(shell pwd)/config.yaml:/data/resource-directory.yaml \
	plgd/resource-directory:v2next --config=/data/resource-directory.yaml
```

## YAML Configuration
### Logging

| Property | Type | Description | Default |
| ---------- | -------- | -------------- | ------- |
| `log.debug` | bool | `set to true if you would like to see extra information on logs` | `false` |

### Grpc Connectivity

| Property | Type | Description | Default |
| ---------- | -------- | -------------- | ------- |
| `api.grpc.address` | string | `listen specification <host>:<port> for grpc client connection.` | `"0.0.0.0:9100"` |
| `api.grpc.tls.caPool` | string | `file path to the root certificates in PEM format` |  `""` |
| `api.grpc.tls.keyFile` | string | `file name of private key in PEM format` | `""` |
| `api.grpc.tls.certFile` | string | `file name of certificate in PEM format` | `""` |
| `api.grpc.tls.clientCertificateRequired` | bool | `require client certificate` | `true` |

### Authorization Client for Grpc Connectivity

| Property | Type | Description | Default |
| ---------- | -------- | -------------- | ------- |
| `api.grpc.authorization.authority` | string | `endpoint of oauth provider` | `""` |
| `api.grpc.authorization.audience` | string | `audience of oauth provider` | `""` |
| `api.grpc.authorization.ownerClaim` | string | `owner claim of oauth provider` | `"sub"` |
| `api.grpc.authorization.http.maxIdleConns` | int | `controls the maximum number of idle (keep-alive) connections across all hosts. Zero means no limit.` | `16` |
| `api.grpc.authorization.http.maxConnsPerHost` | int | `optionally limits the total number of connections per host, including connections in the dialing, active, and idle states. On limit violation, dials will block. Zero means no limit.` | `32` |
| `api.grpc.authorization.http.maxIdleConnsPerHost` | int | `if non-zero, controls the maximum idle (keep-alive) connections to keep per-host. If zero, DefaultMaxIdleConnsPerHost is used.` | `16` |
| `api.grpc.authorization.http.idleConnTimeout` | string | `the maximum amount of time an idle (keep-alive) connection will remain idle before closing itself. Zero means no limit.` | `30s` |
| `api.grpc.authorization.http.timeout` | string | `a time limit for requests made by this Client. A Timeout of zero means no timeout.` | `10s` |
| `api.grpc.authorization.http.tls.caPool` | string | `file path to the root certificates in PEM format` |  `""` |
| `api.grpc.authorization.http.tls.keyFile` | string | `file name of private key in PEM format` | `""` |
| `api.grpc.authorization.http.tls.certFile` | string | `file name of certificate in PEM format` | `""` |
| `api.grpc.authorization.http.tls.useSystemCAPool` | bool | `use system certification pool` | `false` |

### Event Bus Client

| Property | Type | Description | Default |
| ---------- | -------- | -------------- | ------- |
| `clients.eventBus.nats.url` | string | `url to nats messaging system` | `"nats://localhost:4222"` |
| `clients.eventBus.nats.tls.caPool` | string | `file path to the root certificates in PEM format` |  `""` |
| `clients.eventBus.nats.tls.keyFile` | string | `file name of private key in PEM format` | `""` |
| `clients.eventBus.nats.tls.certFile` | string | `file name of certificate in PEM format` | `""` |
| `clients.eventBus.nats.tls.useSystemCAPool` | bool | `use system certification pool` | `false` |

### Storage Client

| Property | Type | Description | Default |
| ---------- | -------- | -------------- | ------- |
| `clients.eventStore.cacheExpiration` | string | `expiration time of cached resource in projection` | `1m` |
| `clients.eventStore.goPoolSize` | int | `number of routines to process events in projection` | `1m` |
| `clients.eventStore.mongoDB.uri` | string | `uri to mongo database` | `"mongodb://localhost:27017"` |
| `clients.eventStore.mongoDB.database` | string | `name of database` | `"eventStore"` |
| `clients.eventStore.mongoDB.batchSize` | int | `limits number of queries in one find request` | `16` |
| `clients.eventStore.mongoDB.maxPoolSize` | int | `limits number of connections` | `16` |
| `clients.eventStore.mongoDB.maxConnIdleTime` | string | `close connection when idle time reach the value` | `240s` |
| `clients.eventStore.mongoDB.tls.caPool` | string | `file path to the root certificates in PEM format` |  `""` |
| `clients.eventStore.mongoDB.tls.keyFile` | string | `file name of private key in PEM format` | `""` |
| `clients.eventStore.mongoDB.tls.certFile` | string | `file name of certificate in PEM format` | `""` |
| `clients.eventStore.mongoDB.tls.useSystemCAPool` | bool | `use system certification pool` | `false` |

### Authorization Server Client

| Property | Type | Description | Default |
| ---------- | -------- | -------------- | ------- |
| `clients.authorizationServer.pullFrequency` | string | `frequency to pull changed user device` | `15s` |
| `clients.authorizationServer.cacheExpiration` | string | `expiration time of cached user device` | `1m` |
| `clients.authorizationServer.grpc.address` | string | `authoriztion server address` | `"127.0.0.1:9100"` |
| `clients.authorizationServer.grpc.tls.caPool` | string | `file path to the root certificates in PEM format` |  `""` |
| `clients.authorizationServer.grpc.tls.keyFile` | string | `file name of private key in PEM format` | `""` |
| `clients.authorizationServer.grpc.tls.certFile` | string | `file name of certificate in PEM format` | `""` |
| `clients.authorizationServer.grpc.tls.useSystemCAPool` | bool | `use system certification pool` | `false` |
| `clients.authorizationServer.grpc.keepAlive.time` | string | `After a duration of this time if the client doesn't see any activity it pings the server to see if the transport is still alive.` | `10s` |
| `clients.authorizationServer.grpc.keepAlive.timeout` | string | `After having pinged for keepalive check, the client waits for a duration of Timeout and if no activity is seen even after that the connection is closed.` | `20s` |
| `clients.authorizationServer.grpc.keepAlive.permitWithoutStream` | bool | `If true, client sends keepalive pings even with no active RPCs. If false, when there are no active RPCs, Time and Timeout will be ignored and no keepalive pings will be sent.` | `false` |

### Authorization Client for OAuth Provider

| Property | Type | Description | Default |
| ---------- | -------- | -------------- | ------- |
| `clients.authorizationServer.oauth.clientID` | string | `client id for authentication to get access token/authorization code` | `""` |
| `clients.authorizationServer.oauth.clientSecret` | string | `client secret for authentication to get access token` |  `""` |
| `clients.authorizationServer.oauth.scopes` | string | `Comma separated list of required scopes` | `""` |
| `clients.authorizationServer.oauth.tokenURL` | string | `token endpoint` | `""` |
| `clients.authorizationServer.oauth.audience` | string | `audience of oauth provider` | `""` |
| `clients.authorizationServer.oauth.verifyServiceTokenFrequency` | string | `frequency to verify service token` | `10s` |
| `clients.authorizationServer.oauth.http.maxIdleConns` | int | `controls the maximum number of idle (keep-alive) connections across all hosts. Zero means no limit.` | `16` |
| `clients.authorizationServer.oauth.http.maxConnsPerHost` | int | `optionally limits the total number of connections per host, including connections in the dialing, active, and idle states. On limit violation, dials will block. Zero means no limit.` | `32` |
| `clients.authorizationServer.oauth.http.maxIdleConnsPerHost` | int | `if non-zero, controls the maximum idle (keep-alive) connections to keep per-host. If zero, DefaultMaxIdleConnsPerHost is used.` | `16` |
| `clients.authorizationServer.oauth.http.idleConnTimeout` | string | `the maximum amount of time an idle (keep-alive) connection will remain idle before closing itself. Zero means no limit.` | `30s` |
| `clients.authorizationServer.oauth.http.timeout` | string | `a time limit for requests made by this Client. A Timeout of zero means no timeout.` | `10s` |
| `clients.authorizationServer.oauth.http.tls.caPool` | string | `file path to the root certificates in PEM format` |  `""` |
| `clients.authorizationServer.oauth.http.tls.keyFile` | string | `file name of private key in PEM format` | `""` |
| `clients.authorizationServer.oauth.http.tls.certFile` | string | `file name of certificate in PEM format` | `""` |
| `clients.authorizationServer.oauth.http.tls.useSystemCAPool` | bool | `use system certification pool` | `false` |

### Public Configuration

| Property | Type | Description | Default |
| ---------- | -------- | -------------- | ------- |
| `publicConfiguration.caPool` | string | `path root CA which was used to sign coap-gw certificate` | `""` |
| `publicConfiguration.tokenURL` | string | `url where user can get OAuth token via implicit flow` | `""` |
| `publicConfiguration.authorizationURL` | string | `url where user can get OAuth authorization code for the device` | `""` |
| `publicConfiguration.ownerClaim` | string | `owner claim of oauth provider` | `"sub"` |
| `publicConfiguration.signingServerAddress` | string | `address of ceritificate authority for plgd-dev/sdk` | `""` |  
| `publicConfiguration.cloudID` | string | `cloud id which is stored in coap-gw certificate` | `""` |
| `publicConfiguration.cloudURL` | string | `cloud url for onboard device` | `""` |
| `publicConfiguration.cloudAuthorizationProvider` | string | `oauth authorization provider for onboard device` | `""` |

> Note that the string type related to time (i.e. timeout, idleConnTimeout, expirationTime) is decimal numbers, each with optional fraction and a unit suffix, such as "300ms", "1.5h" or "2h45m". Valid time units are "ns", "us", "ms", "s", "m", "h".
