# OAuth Server
Mocked OAuth2.0 Server used for automated tests and [bundle container](...)

## Docker Image

```bash
# Dowonload github source
git clone https://github.com/plgd-dev/cloud.git 

# Build the source
cd cloud/ 
make build
```

## Docker Run
### How to make certificates and private keys for tokens
Before you run docker image of plgd/authorization, you make sure certificates exists on `.tmp/certs` folder. 
If not exists, you can create certificates from plgd/bundle image by following step only once.

- Create certificates
```bash
# Create certificates on the source
make certificates 
```
Or 
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


- Create private keys for tokens
```bash
# Create private keys on the source
make privateKeys 
```
Or 
```bash
mkdir -p $(pwd)/.tmp/privKeys
openssl genrsa -out $(pwd)/.tmp/privKeys/idTokenKey.pem 4096
openssl ecparam -name prime256v1 -genkey -noout -out $(pwd)/.tmp/privKeys/accessTokenKey.pem
```

```bash
# See token keys for oauth-server.
ls .tmp/privKeys 
accessTokenKey.pem	idTokenKey.pem
```

### How to get configuration file
A configuration template is available on [test/oauth-server/config.yaml](https://github.com/plgd-dev/cloud/blob/v2/test/oauth-server/config.yaml). 
You can also see `config.yaml` configuration file on the `test/oauth-server` folder by downloading `git clone https://github.com/plgd-dev/cloud.git`. 
```bash
# See config file on the source
cat test/oauth-server/conifg.yaml > oauth-server.yaml
```

### Edit configuration file 
You can edit configuration file including server port, certificates, token keys and so on.

See an example of address, tls and keys on the followings.
```yaml
...
apis:
  grpc:
    address: "0.0.0.0:9088"
    tls:
      caPool: "/data/certs/root_ca.crt"
      keyFile: "/data/certs/http.key"
      certFile: "/data/certs/http.crt"
...
oauthSigner:
  idTokenKeyFile: "/secret/private/idToken.key"
  accessTokenKeyFile: "/secret/private/accessToken.key"
  domain: "localhost:9088"
```

### Run docker image 
You can run plgd/authorization image using certificates and configuration file on the folder you made certificates.
```bash
docker run -d --network=host \
	--name=authorization \
	-v $(pwd)/.tmp/certs:/data/certs \
	-v $(pwd)/.tmp/privKeys:/secret/private \
	-v $(pwd)/oauth-server.yaml:/data/oauth-server.yaml \
	plgd/oauth-server:latest --config=/data/oauth-server.yaml
```

## YAML Configuration
### Logging

| Property | Type | Description | Default |
| ---------- | -------- | -------------- | ------- |
| `log.debug` | bool | `Set to true if you would like to see extra information on logs.` | `false` |

### HTTP API
HTTP API of the OAuth Server service as defined [here](https://github.com/plgd-dev/cloud/blob/v2/test/oauth-server/uri/uri.go)

| Property | Type | Description | Default |
| ---------- | -------- | -------------- | ------- |
| `api.http.address` | string | `Listen specification <host>:<port> for http client connection.` | `"0.0.0.0:9100"` |
| `api.http.tls.caPool` | string | `File path to the root certificate in PEM format which might contain multiple certificates in a single file.` |  `""` |
| `api.http.tls.keyFile` | string | `File path to private key in PEM format.` | `""` |
| `api.http.tls.certFile` | string | `File path to certificate in PEM format.` | `""` |
| `api.http.tls.clientCertificateRequired` | bool | `If true, require client certificate.` | `true` |

### OAuth Signer
Signer configuration to issue ID/access tokens of OAuth provider for mock testing.

| Property | Type | Description | Default |
| ---------- | -------- | -------------- | ------- |
| `oauthSigner.idTokenKeyFile` | string | `File path to private key for ID token in PEM format.` | `""` |
| `oauthSigner.accessTokenKeyFile` | string | `File path to private key for access token in PEM format.` | `""` |
| `oauthSigner.domain` | string | `Audience for access token.` | `""` |

> Note that the string type related to time (i.e. timeout, idleConnTimeout, expirationTime) is decimal numbers, each with optional fraction and a unit suffix, such as "300ms", "1.5h" or "2h45m". Valid time units are "ns", "us", "ms", "s", "m", "h".
