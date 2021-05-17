# OAuth Server
Mocked OAuth2.0 Server used for automated tests and [bundle container](...)

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
| `oauthSigner.idTokenKeyFile` | string | `File path to a private RSA key in PEM format required for ID token signing.` | `""` |
| `oauthSigner.accessTokenKeyFile` | string | `File path to a private ECDSA key in PEM format required for access token signing.` | `""` |
| `oauthSigner.domain` | string | `Domain address <host>:<port> for OAuth APIs.` | `""` |

> Note that the string type related to time (i.e. timeout, idleConnTimeout, expirationTime) is decimal numbers, each with optional fraction and a unit suffix, such as "300ms", "1.5h" or "2h45m". Valid time units are "ns", "us", "ms", "s", "m", "h".
