
## plgd-cloud helm chart

The following tables lists the configurable parameters of the plgd-cloud chart and their default values.

#### Provisioned services

- [coap-gateway](https://github.com/plgd-dev/cloud/tree/v2/coap-gateway)
- [authorization](https://github.com/plgd-dev/cloud/tree/v2/authorization)
- [resource-aggregate](https://github.com/plgd-dev/cloud/tree/v2/resource-aggregate)
- [resource-directory](https://github.com/plgd-dev/cloud/tree/v2/resource-directory)

## Requirements

- [cert-manager](https://artifacthub.io/packages/helm/cert-manager/cert-manager)

## Parameters

### Required

| Name  | Description  |  
|---|---|
| coapgateway.cloudId  | CloudId used during coap-gateway service certificate generation. |

### Coap-gateway

In order to configure coap-gateway service, see [cloud/coap-gateway](https://github.com/plgd-dev/cloud/tree/v2/coap-gateway)
and [cloud/coap-gateway/config.yaml](https://github.com/plgd-dev/cloud/blob/v2/coap-gateway/config.yaml).
All parameter can be configured with `coapgateway.` prefix.

| Name  | Description  | Default  |
| coapgateway.enabled  | Enable coap-gateway  | `true` |
| coapgateway.name  | Name of component. Used in label selectors  | `coap-gateway` |
| coapgateway.fullnameOverride  | Full name to override  |  |
| coapgateway.replicas  | Number of replicas  | `1` |
| coapgateway.deploymentLabels  | Deployment extra labels  | `{}` |
| coapgateway.deploymentAnnotations  |Additional annotations for coap-gateway deployment | `{}` |
| coapgateway.podSecurityContext  | Pod security context | `{}` |
| coapgateway.podLabels  | Labels for coap-gateway pod | `{}` |
| coapgateway.podAnnotations  | Annotations for coap-gateway pod | `{}` |
| coapgateway.service.type  | Service type | `LoadBalancer` |
| coapgateway.service.labels  | Service labels | `{}` |
| coapgateway.service.annotations  | Service annotations | `{}` |
| coapgateway.rbac.enabled  | Enable RBAC | `false` |
| coapgateway.rbac.serviceAccountName  | Name of coap-gateway SA | `coap-gateway` |
| coapgateway.rbac.roleBindingDefinitionTpl  | Role binding definition resolved as template | `{}` |
| coapgateway.securityContext  | Security context for pod | `{}` |
| coapgateway.imagePullSecrets  | Image pull secrets | `{}` |
| coapgateway.restartPolicy  | Restart policy for pod | `{}` |
| coapgateway.initContainersTpl  | Init containers definition resolved as template | `{}` |
| coapgateway.image.registry  | Registry name |  |
| coapgateway.image.repository  | Repository name | `plgd/coap-gateway` |
| coapgateway.image.tag  | Tag name | `plgd/coap-gateway` |
| coapgateway.image.tag  | Tag name | `plgd/coap-gateway` |
| coapgateway.image.imagePullSecrets  | Pull secrets | `{}` |
| coapgateway.livenessProbe  | Liveness probe | `{}` |
| coapgateway.readinessProbe  | Readiness  probe | `{}` |
| coapgateway.resources  | Resource  limit | `{}` |
| coapgateway.nodeSelector  | Node selector | `{}` |
| coapgateway.tolerations  | Toleration  | `{}` |
| coapgateway.affinity  | Affinity | `{}` |
| coapgateway.extraVolumes  | Extra volume | `{}` |
| coapgateway.extraVolumeMounts  | Extra volume mounts | `{}` |
| coapgateway.config.fileName  | Name of config file for service | `service.yaml` |
| coapgateway.config.volume  | Name of volume for service | `config` |
| coapgateway.config.volume  | Mount path for config | `/config` |
| coapgateway.port  | Service port | `5584` |
| coapgateway.cloudId  | Cloud ID |  |

### Cert-manager integration

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