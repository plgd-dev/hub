# Cloud2Cloud connector

## Description

Connects the cloud to a OCF Cloud, populate devices from the OCF cloud, update devices of the OCF cloud, maintenance of linked clouds and linked accounts.

## Docker Image

```bash
docker pull plgd/cloud2cloud-connector:latest
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

A configuration template is available in [cloud2cloud-connector/config.yaml](https://github.com/plgd-dev/cloud/blob/v2/cloud2cloud-connector/config.yaml).
You can also see the `config.yaml` configuration file in the `cloud2cloud-connector` folder by downloading `git clone https://github.com/plgd-dev/cloud.git`.

```bash
# Copy & paste the configuration template from the link and save the file as `cloud2cloud-connector.yaml` to a local folder.
vi cloud2cloud-connector.yaml

# Or download the configuration template with curl.
curl https://github.com/plgd-dev/cloud/blob/v2/cloud2cloud-connector/config.yaml --output cloud2cloud-connector.yaml
```

### Edit configuration file

You can edit values in the configuration file such as server port, certificates, OAuth provider and so on.
Read more details about how to configure the OAuth Provider [here](https://github.com/plgd-dev/cloud/blob/v2/docs/guide/developing/authorization.md#how-to-configure-auth0).

The following example shows configuration of address, clients and the rest of configuration.

```yaml
...
apis:
  http:
    address: "0.0.0.0:9100"
    tls:
      caPool: "/data/certs/root_ca.crt"
      keyFile: "/data/certs/http.key"
      certFile: "/data/certs/http.crt"
      clientCertificateRequired: true
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
  authorizationServer:
    grpc:
      address: "localhost:9081"
      tls:
        caPool: "/data/certs/root_ca.crt"
        keyFile: "/data/certs/http.key"
        certFile: "/data/certs/http.crt"
...
  eventBus:
    nats:
      url: "nats://localhost:4222"
      tls:
        caPool: "/data/certs/root_ca.crt"
        keyFile: "/data/certs/http.key"
        certFile: "/data/certs/http.crt"
...
  grpcGateway:
    grpc:
      address: "localhost:9083"
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
  storage:
    mongoDB:
      uri: "mongodb://localhost:27017"
      database: "cloud2cloudConnector"
      tls:
        caPool: "/data/certs/root_ca.crt"
        keyFile: "/data/certs/http.key"
        certFile: "/data/certs/http.crt"
...
  subscription:
    http:
      reconnectInterval: "10s"
      resubscribeInterval: "10s"
...
taskProcessor:
  cacheSize: 2048
  timeout: "5s"
  maxParallel: 128
```

### Run docker image

You can run the plgd/cloud2cloud-connector image using certificates and configuration file in the folder you made certificates in.

```bash
docker run -d --network=host \
  --name=cloud2cloud-connector \
  -v $(pwd)/.tmp/certs:/data/certs \
  -v $(pwd)/cloud2cloud-connector.yaml:/data/cloud2cloud-connector.yaml \
  plgd/cloud2cloud-connector:latest --config=/data/cloud2cloud-connector.yaml
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
| `apis.http.eventsURL` | string | `External url where the events from another cloud will be send` | `""` |
| `apis.http.pullDevices.disabled` | bool | `Disable get devices via pull for all clouds` | `true` |
| `apis.http.pullDevices.interval` | string | `"Time interval between pulls` | `"5s"` |
| `apis.http.clientID` | string | `Client ID to exchange an authorization code for an access token.` | `""` |
| `apis.http.clientSecret` | string | `Client secret to exchange an authorization code for an access token.` | `""` |
| `apis.http.scopes` | string array | `List of required scopes.` | `""` |
| `apis.http.authorization.authority` | string | `Authority is the address of the token-issuing authentication server. Services will use this URI to find and retrieve the public key that can be used to validate the token’s signature.` | `""` |
| `apis.http.authorization.audience` | string | `Identifier of the API configured in your OAuth provider.` | `""` |
| `apis.http.authorization.redirectURL` | string | `External redirect url to acquire authorization code.` | `""` |
| `apis.http.authorization.http.maxIdleConns` | int | `It controls the maximum number of idle (keep-alive) connections across all hosts. Zero means no limit.` | `16` |
| `apis.http.authorization.http.maxConnsPerHost` | int | `It optionally limits the total number of connections per host, including connections in the dialing, active, and idle states. On limit violation, dials will block. Zero means no limit.` | `32` |
| `apis.http.authorization.http.maxIdleConnsPerHost` | int | `If non-zero, controls the maximum idle (keep-alive) connections to keep per-host. If zero, DefaultMaxIdleConnsPerHost is used.` | `16` |
| `apis.http.authorization.http.idleConnTimeout` | string | `The maximum amount of time an idle (keep-alive) connection will remain idle before closing itself. Zero means no limit.` | `30s` |
| `apis.http.authorization.http.timeout` | string | `A time limit for requests made by this Client. A Timeout of zero means no timeout.` | `10s` |
| `apis.http.authorization.http.tls.caPool` | string | `File path to the root certificate in PEM format which might contain multiple certificates in a single file.` |  `""` |
| `apis.http.authorization.http.tls.keyFile` | string | `File path to private key in PEM format.` | `""` |
| `apis.http.authorization.http.tls.certFile` | string | `File path to certificate in PEM format.` | `""` |
| `apis.http.authorization.http.tls.useSystemCAPool` | bool | `If true, use system certification pool.` | `false` |

### Authorization Server Client

Client configurations to internally connect to Authorization Server service.

| Property | Type | Description | Default |
| ---------- | -------- | -------------- | ------- |
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
| `clients.storage.mongoDB.database` | string | `Name of database.` | `"cloud2cloudConnector"` |
| `clients.storage.mongoDB.tls.caPool` | string | `File path to the root certificate in PEM format which might contain multiple certificates in a single file.` |  `""` |
| `clients.storage.mongoDB.tls.keyFile` | string | `File path to private key in PEM format.` | `""` |
| `clients.storage.mongoDB.tls.certFile` | string | `File path to certificate in PEM format.` | `""` |
| `clients.storage.mongoDB.tls.useSystemCAPool` | bool | `If true, use system certification pool.` | `false` |

### Subscription

| Property | Type | Description | Default |
| ---------- | -------- | -------------- | ------- |
| `clients.subscription.http.reconnectInterval` | string | `The interval after which to try to reconnect to grpc-gateway client when connection was closed.` | `"10s"` |
| `clients.subscription.http.resubscribeInterval` | string | `The interval after which to try to resubscribe to grpc-gateway client when subscription does not exist.` | `"10s"` |

### Task Queue

| Property | Type | Description | Default |
| ---------- | -------- | -------------- | ------- |
| `taskProcessor.cacheSize` | int | `Size of processor task queue` | `2048` |
| `taskProcessor.timeout` | string | `Timeout for one running task` | `"5s"` |
| `taskProcessor.maxParallel` | int | `Maximal number of running tasks at the same time` | `128` |
| `taskProcessor.delay` | string | `Task delay before start` | `0s` |

## API

Follow [OCF Cloud API For Cloud Services Specification](https://openconnectivity.org/specs/OCF_Cloud_API_For_Cloud_Services_Specification_v2.2.0.pdf)

### Commands

- maintenance of linked clouds
- maintenance of linked accounts
- [swagger](https://petstore.swagger.io/?url=https://raw.githubusercontent.com/plgd-dev/cloud/master/cloud2cloud-connector/swagger.yaml)
