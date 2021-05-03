# Authorization server

## Docker Image

```bash
docker pull plgd/authorization:v2next
```
Or 
```bash
git clone https://github.com/plgd-dev/cloud.git 
cd cloud/
make build
```

## Docker Run
### How to make certificates
Before you run docker image of plgd/authorization, you make sure to execute below script only once. 
```bash
git clone https://github.com/plgd-dev/cloud.git 

cd cloud/
make certificates 
make privateKeys
```

### How to get configuration file
- A configuration template is available on [authorization/config.yaml](https://github.com/plgd-dev/cloud/blob/v2/authorization/config.yaml). 

You can also see configuration file via executing below script. 
```bash
git clone https://github.com/plgd-dev/cloud.git 

cd cloud/
cat authorization/conifg.yaml 
```

### Edit configuration file 
You can edit configuration file such as server port, certificates, oauth provider and so on.
Read more detail about how to configure OAuth Provider [here](https://github.com/plgd-dev/cloud/blob/v2/docs/guide/developing/authorization.md#how-to-configure-auth0) 

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
You can run plgd/authorization image using certificates and configuration file on the plgd/authorization directory.
```bash
docker run -d --network=host \
	--name=authorization \
	-v $(shell pwd)/../.tmp/certs:/data/certs \
	-v $(shell pwd)/config.yaml:/data/authorization.yaml \
	plgd/authorization:v2next --config=/data/authorization.yaml
```

## YAML Configuration
### Logging

| Property | Type | Description | Default |
| ---------- | -------- | -------------- | ------- |
| `log.debug` | bool | `set to true if you would like to see extra information on logs` | `false` |

### Grpc Connectivity & TLS

| Property | Type | Description | Default |
| ---------- | -------- | -------------- | ------- |
| `api.grpc.address` | string | `listen specification <host>:<port> for grpc client connection.` | `"0.0.0.0:9100"` |
| `api.grpc.tls.caPool` | string | `file path to the root certificates in PEM format` |  `""` |
| `api.grpc.tls.keyFile` | string | `file name of private key in PEM format` | `""` |
| `api.grpc.tls.certFile` | string | `file name of certificate in PEM format` | `""` |
| `api.grpc.tls.clientCertificateRequired` | bool | `require client certificate` | `true` |

### Authorization Client & TLS

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

### Http Connectivity & TLS

| Property | Type | Description | Default |
| ---------- | -------- | -------------- | ------- |
| `api.http.address` | string | `listen specification <host>:<port> for http client connection.` | `"0.0.0.0:9100"` |
| `api.http.tls.caPool` | string | `file path to the root certificates in PEM format` |  `""` |
| `api.http.tls.keyFile` | string | `file name of private key in PEM format` | `""` |
| `api.http.tls.certFile` | string | `file name of certificate in PEM format` | `""` |
| `api.http.tls.clientCertificateRequired` | bool | `require client certificate` | `true` |

### Authorization Client & TLS for Device OAuth Provider

| Property | Type | Description | Default |
| ---------- | -------- | -------------- | ------- |
| `oauthClients.device.provider` | string | `value which comes from the device during the sign-up ("apn")` | `"generic"` |
| `oauthClients.device.clientID` | string | `client id for authentication to get access token/authorization code` | `""` |
| `oauthClients.device.clientSecret` | string | `client secret for authentication to get access token` |  `""` |
| `oauthClients.device.scopes` | string | `Comma separated list of required scopes` | `""` |
| `oauthClients.device.authorizationURL` | string | `authorization endpoint` | `""` |
| `oauthClients.device.tokenURL` | string | `token endpoint` | `""` |
| `oauthClients.device.audience` | string | `audience of oauth provider` | `""` |
| `oauthClients.device.redirectURL` | string | `redirect url used to obtain device access token` | `""` |
| `oauthClients.device.responseType` | string | `one of "code/token"` | `"code"` |
| `oauthClients.device.responseMode` | string | `one of "query/post_form"` | `"post_form"` |
| `oauthClients.device.http.maxIdleConns` | int | `controls the maximum number of idle (keep-alive) connections across all hosts. Zero means no limit.` | `16` |
| `oauthClients.device.http.maxConnsPerHost` | int | `optionally limits the total number of connections per host, including connections in the dialing, active, and idle states. On limit violation, dials will block. Zero means no limit.` | `32` |
| `oauthClients.device.http.maxIdleConnsPerHost` | int | `if non-zero, controls the maximum idle (keep-alive) connections to keep per-host. If zero, DefaultMaxIdleConnsPerHost is used.` | `16` |
| `oauthClients.device.http.idleConnTimeout` | string | `the maximum amount of time an idle (keep-alive) connection will remain idle before closing itself. Zero means no limit.` | `30s` |
| `oauthClients.device.http.timeout` | string | `a time limit for requests made by this Client. A Timeout of zero means no timeout.` | `10s` |
| `oauthClients.device.http.tls.caPool` | string | `file path to the root certificates in PEM format` |  `""` |
| `oauthClients.device.http.tls.keyFile` | string | `file name of private key in PEM format` | `""` |
| `oauthClients.device.http.tls.certFile` | string | `file name of certificate in PEM format` | `""` |
| `oauthClients.device.http.tls.useSystemCAPool` | bool | `use system certification pool` | `false` |

### Authorization Client & TLS for Service OAuth Provider

| Property | Type | Description | Default |
| ---------- | -------- | -------------- | ------- |
| `oauthClients.client.clientID` | string | `client id for authentication to get access token/authorization code` | `""` |
| `oauthClients.client.clientSecret` | string | `client secret for authentication to get access token` |  `""` |
| `oauthClients.client.scopes` | string | `Comma separated list of required scopes` | `""` |
| `oauthClients.client.authorizationURL` | string | `authorization endpoint` | `""` |
| `oauthClients.client.audience` | string | `audience of oauth provider` | `""` |
| `oauthClients.client.redirectURL` | string | `redirect url used to obtain device access token` | `""` |
| `oauthClients.client.responseMode` | string | `one of "query/post_form"` | `"post_form"` |
| `oauthClients.client.http.maxIdleConns` | int | `controls the maximum number of idle (keep-alive) connections across all hosts. Zero means no limit.` | `16` |
| `oauthClients.client.http.maxConnsPerHost` | int | `optionally limits the total number of connections per host, including connections in the dialing, active, and idle states. On limit violation, dials will block. Zero means no limit.` | `32` |
| `oauthClients.client.http.maxIdleConnsPerHost` | int | `if non-zero, controls the maximum idle (keep-alive) connections to keep per-host. If zero, DefaultMaxIdleConnsPerHost is used.` | `16` |
| `oauthClients.client.http.idleConnTimeout` | string | `the maximum amount of time an idle (keep-alive) connection will remain idle before closing itself. Zero means no limit.` | `30s` |
| `oauthClients.client.http.timeout` | string | `a time limit for requests made by this Client. A Timeout of zero means no timeout.` | `10s` |
| `oauthClients.client.http.tls.caPool` | string | `file path to the root certificates in PEM format` |  `""` |
| `oauthClients.client.http.tls.keyFile` | string | `file name of private key in PEM format` | `""` |
| `oauthClients.client.http.tls.certFile` | string | `file name of certificate in PEM format` | `""` |
| `oauthClients.client.http.tls.useSystemCAPool` | bool | `use system certification pool` | `false` |

### Storage Database & TLS 

| Property | Type | Description | Default |
| ---------- | -------- | -------------- | ------- |
| `clients.storage.mongoDB.uri` | string | `uri to mongo database` | `"mongodb://localhost:27017"` |
| `clients.storage.mongoDB.database` | string | `name of database` | `"ownersDevices"` |
| `clients.storage.mongoDB.tls.caPool` | string | `file path to the root certificates in PEM format` |  `""` |
| `clients.storage.mongoDB.tls.keyFile` | string | `file name of private key in PEM format` | `""` |
| `clients.storage.mongoDB.tls.certFile` | string | `file name of certificate in PEM format` | `""` |
| `clients.storage.mongoDB.tls.useSystemCAPool` | bool | `use system certification pool` | `false` |

> Note that the string type related to time (i.e. timeout, idleConnTimeout, expirationTime) is decimal numbers, each with optional fraction and a unit suffix, such as "300ms", "1.5h" or "2h45m". Valid time units are "ns", "us", "ms", "s", "m", "h".

