# Certificate authority

## Docker Image

```bash
docker pull plgd/certificate-authority:latest
```

## Docker Run
### How to make certificates
Before you run docker image of plgd/certificate-authority, you make sure certificates exists on `.tmp/certs` folder. 
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
certificate-generator --cmd.generateIdentityCertificate=$CLOUD_SID --outCert=/certs/coap.crt --outKey=/certs/coap.key --cert.san.domain=localhost --signerCert=/certs/root_ca.crt --signerKey=/certs/root_ca.key
cat /certs/http.crt > /certs/mongo.key
cat /certs/http.key >> /certs/mongo.key

# Exit shell.
exit 
```
```bash
# See common certificates for plgd cloud services.
ls .tmp/certs
coap.crt	coap.key	http.crt	http.key	mongo.key	root_ca.crt	root_ca.key
```
### How to get configuration file
A configuration template is available on [certificate-authority/config.yaml](https://github.com/plgd-dev/cloud/blob/v2/certificate-authority/config.yaml). 
You can also see `config.yaml` configuration file on the `certificate-authority` folder by downloading `git clone https://github.com/plgd-dev/cloud.git`. 
```bash
# Copy & paste configuration template from the link and save the file named `certificate-authority.yaml` on the local folder.
vi certificate-authority.yaml

# Or download configuration template.
curl https://github.com/plgd-dev/cloud/blob/v2/certificate-authority/config.yaml --output certificate-authority.yaml 
```

### Edit configuration file 
You can edit configuration file including server port, certificates, OAuth provider and so on.
Read more detail about how to configure OAuth Provider [here](https://github.com/plgd-dev/cloud/blob/v2/docs/guide/developing/authorization.md#how-to-configure-auth0). 

See an example of address, tls on the followings.
```yaml
...
apis:
  grpc:
    address: "0.0.0.0:9087"
    tls:
      caPool: "/data/certs/root_ca.crt"
      keyFile: "/data/certs/http.key"
      certFile: "/data/certs/http.crt"
...
signer:
  keyFile: "/data/certs/root_ca.key"
  certFile: "/data/certs/root_ca.crt"
...
```

### Run docker image 
You can run plgd/certificate-authority image using certificates and configuration file on the folder you made certificates.
```bash
docker run -d --network=host \
	--name=certificate-authority \
	-v $(pwd)/.tmp/certs:/data/certs \
	-v $(pwd)/certificate-authority.yaml:/data/certificate-authority.yaml \
	plgd/certificate-authority:latest --config=/data/certificate-authority.yaml
```

## YAML Configuration
### Logging

| Property | Type | Description | Default |
| ---------- | -------- | -------------- | ------- |
| `log.debug` | bool | `Set to true if you would like to see extra information on logs.` | `false` |

### gRPC API
gRPC API of the Certificate Authority Service as defined [here](https://github.com/plgd-dev/cloud/blob/v2/certificate-authority/pb/service_grpc.pb.go#L19).

| Property | Type | Description | Default |
| ---------- | -------- | -------------- | ------- |
| `api.grpc.address` | string | `Listen specification <host>:<port> for grpc client connection.` | `"0.0.0.0:9100"` |
| `api.grpc.tls.caPool` | string | `File path to the root certificate in PEM format.` |  `""` |
| `api.grpc.tls.keyFile` | string | `File path to private key in PEM format.` | `""` |
| `api.grpc.tls.certFile` | string | `File path to certificate in PEM format.` | `""` |
| `api.grpc.tls.clientCertificateRequired` | bool | `If true, require client certificate.` | `true` |
| `api.grpc.authorization.authority` | string | `Endpoint of OAuth provider.` | `""` |
| `api.grpc.authorization.audience` | string | `Identifier of the API configured in your OAuth provider.` | `""` |
| `api.grpc.authorization.ownerClaim` | string | `Claim used to identify owner of the device.` | `"sub"` |
| `api.grpc.authorization.http.maxIdleConns` | int | `It controls the maximum number of idle (keep-alive) connections across all hosts. Zero means no limit.` | `16` |
| `api.grpc.authorization.http.maxConnsPerHost` | int | `It optionally limits the total number of connections per host, including connections in the dialing, active, and idle states. On limit violation, dials will block. Zero means no limit.` | `32` |
| `api.grpc.authorization.http.maxIdleConnsPerHost` | int | `If non-zero, controls the maximum idle (keep-alive) connections to keep per-host. If zero, DefaultMaxIdleConnsPerHost is used.` | `16` |
| `api.grpc.authorization.http.idleConnTimeout` | string | `The maximum amount of time an idle (keep-alive) connection will remain idle before closing itself. Zero means no limit.` | `30s` |
| `api.grpc.authorization.http.timeout` | string | `A time limit for requests made by this Client. A Timeout of zero means no timeout.` | `10s` |
| `api.grpc.authorization.http.tls.caPool` | string | `File path to the root certificate in PEM format which might contain multiple certificates in a single file.` |  `""` |
| `api.grpc.authorization.http.tls.keyFile` | string | `File path to private key in PEM format.` | `""` |
| `api.grpc.authorization.http.tls.certFile` | string | `File path to certificate in PEM format.` | `""` |
| `api.grpc.authorization.http.tls.useSystemCAPool` | bool | `If true, use system certification pool.` | `false` |

### Signer

| Property | Type | Description | Default |
| ---------- | -------- | -------------- | ------- |
| `signer.keyFile` | string | `File path to the root certificate in PEM format.` |  `""` |
| `signer.certFile` | string | `File path to the root private key in PEM format.` |  `""` |
| `signer.validFrom` | string | `The time from when the certificate is valid. (Format: https://github.com/karrick/tparse)` |  `"now-1h"` |
| `signer.expiresIn` | string | `The time up to which the certificate is valid.` |  `"87600h"` |

> Note that the string type related to time (i.e. timeout, idleConnTimeout, expirationTime) is decimal numbers, each with optional fraction and a unit suffix, such as "300ms", "1.5h" or "2h45m". Valid time units are "ns", "us", "ms", "s", "m", "h".

