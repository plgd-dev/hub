
## Configuration

The following tables lists the configurable parameters of the plgd-cloud chart and their default values.

### Required parameters

| Name  | Description  |  
|---|---|
| coapgateway.cloudId  | CloudId used during coap-gateway service certificate generation. |


### Cert-manager

For issuing internal and external certificate, [cert-manager.io](https://cert-manager.io/) is used as default option and 
integration can be extended with following values:

Default issuer/certificate configuration is applied in default setup

| Name  | Description  | Default  |
|---|---|---|
| certmanager.enabled  | Enable cert-manager integration  | true  | 
| certmanager.default.issuer.enabled  | Enable default issuer integration  | true  | 
| certmanager.default.issuer.labels  | Labels  | {}  | 
| certmanager.default.issuer.annotations  | Labels  | {}  | 
| certmanager.default.issuer.name  | Name of issuer  | selfsigned-issuer  | 
| certmanager.default.issuer.kind  | Kind of issuer  | ClusterIssuer  | 
| certmanager.default.issuer.spec  | Spec  | selfSigned: {} | 
| certmanager.default.cert.labels  | Labels  | {} | 
| certmanager.default.cert.annotations  | Labels  | {} | 
| certmanager.default.cert.duration  | Certificate duration  | 8760h | 
| certmanager.default.cert.renewBefore  | Certificate renew before  | 360h | 
| certmanager.default.cert.key.algorithm  | Type of cert key  | ECDSA | 
| certmanager.default.cert.key.size  | Size of cert key  | 256 | 


Cert-manager integration for coap certificate

| Name  | Description  | Default  |
|---|---|---|
| certmanager.coap.issuer.labels  | Labels  | {}  | 
| certmanager.coap.issuer.annotations  | Labels  | {}  | 
| certmanager.coap.issuer.name  | Name of issuer |   | 
| certmanager.coap.issuer.kind  | Kind of issuer  |   | 
| certmanager.coap.issuer.spec  | Spec. In case this value is specified. New issuer will be created  | {} | 
| certmanager.coap.cert.labels  | Labels  | {} | 
| certmanager.coap.cert.annotations  | Labels  | {} | 
| certmanager.coap.cert.duration  | Certificate duration  | 8760h | 
| certmanager.coap.cert.renewBefore  | Certificate renew before  | 360h | 
| certmanager.coap.cert.key.algorithm  | Type of cert key  | ECDSA | 
| certmanager.coap.cert.key.size  | Size of cert key  | 256 | 

Cert-manager integration for internal certificates

| Name  | Description  | Default  |
|---|---|---|
| certmanager.internal.issuer.labels  | Labels  | {}  | 
| certmanager.internal.issuer.annotations  | Labels  | {}  | 
| certmanager.internal.issuer.name  | Name of issuer |   | 
| certmanager.internal.issuer.kind  | Kind of issuer  |   | 
| certmanager.internal.issuer.spec  | Spec. In case this value is specified. New issuer will be created  | {} | 
| certmanager.internal.cert.labels  | Labels  | {} | 
| certmanager.internal.cert.annotations  | Labels  | {} | 
| certmanager.internal.cert.duration  | Certificate duration  | 8760h | 
| certmanager.internal.cert.renewBefore  | Certificate renew before  | 360h | 
| certmanager.internal.cert.key.algorithm  | Type of cert key  | ECDSA | 
| certmanager.internal.cert.key.size  | Size of cert key  | 256 |