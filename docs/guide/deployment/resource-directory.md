# Resource directory

## Description

According to CQRS pattern it creates/updates projection for resource directory and resource shadow.

## Docker Image

```bash
docker pull plgd/resource-directory:v2next
```

## API

All requests to service must contains valid access token in [grpc metadata](https://github.com/grpc/grpc-go/blob/master/Documentation/grpc-auth-support.md#oauth2).

### Commands

- get devices - list devices
- get resource links - list resource links
- retrieve resource from device - get content from the device
- retrieve resources values - get resources from the resource shadow
- update resources values - update resource at the device
- delete resource - delete resource at the device
- subscribe for events - provides notification about device registered/unregistered/online/offline, resource published/unpublished/content changed/ ...
- get client configuration - provides public configuration for clients(mobile, web, onboarding tool)
- create resource - create resource at the device (e.g. create new resource on collection resource on a device)

### Contract

- [service](https://github.com/plgd-dev/cloud/blob/v2/grpc-gateway/pb/service.proto)
- [requets/responses](https://github.com/plgd-dev/cloud/blob/v2/grpc-gateway/pb/devices.proto)
- [client configuration](https://github.com/plgd-dev/cloud/blob/v2/grpc-gateway/pb/clientConfiguration.proto)

## Configuration
- [resource-directory/config.yaml](https://github.com/plgd-dev/cloud/blob/v2/resource-directory/config.yaml) 

| Key | Type | Description | Default |
| --------- | ----------- | ------- | ------- |
| `log.debug` | bool | `enable debugging message` | `false` |
| `api.grpc.address` | string | `listen address` | `"0.0.0.0:9100"` |
| `api.grpc.tls.caPool` | string | `file path to the root certificates in PEM format` |  `""` |
| `api.grpc.tls.keyFile` | string | `file name of private key in PEM format` | `""` |
| `api.grpc.tls.certFile` | string | `file name of certificate in PEM format` | `""` |
| `api.grpc.tls.clientCertificateRequired` | bool | `require client certificate` | `true` |
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
| `clients.eventBus.nats.url` | string | `url to nats messaging system` | `"nats://localhost:4222"` |
| `clients.eventBus.nats.tls.caPool` | string | `file path to the root certificates in PEM format` |  `""` |
| `clients.eventBus.nats.tls.keyFile` | string | `file name of private key in PEM format` | `""` |
| `clients.eventBus.nats.tls.certFile` | string | `file name of certificate in PEM format` | `""` |
| `clients.eventBus.nats.tls.useSystemCAPool` | bool | `use system certification pool` | `false` |
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
| `publicConfiguration.caPool` | string | `path root CA which was used to sign coap-gw certificate` | `""` |
| `publicConfiguration.tokenURL` | string | `url where user can get OAuth token via implicit flow` | `""` |
| `publicConfiguration.authorizationURL` | string | `url where user can get OAuth authorization code for the device` | `""` |
| `publicConfiguration.ownerClaim` | string | `owner claim of oauth provider` | `"sub"` |
| `publicConfiguration.signingServerAddress` | string | `address of ceritificate authority for plgd-dev/sdk` | `""` |  
| `publicConfiguration.cloudID` | string | `cloud id which is stored in coap-gw certificate` | `""` |
| `publicConfiguration.cloudURL` | string | `cloud url for onboard device` | `""` |
| `publicConfiguration.cloudAuthorizationProvider` | string | `oauth authorization provider for onboard device` | `""` |
