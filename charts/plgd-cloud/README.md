# plgd-hub

A Helm chart for plgd-hub

![Version: 0.0.1](https://img.shields.io/badge/Version-0.0.1-informational?style=flat-square) ![Type: application](https://img.shields.io/badge/Type-application-informational?style=flat-square) ![AppVersion: v2next](https://img.shields.io/badge/AppVersion-v2next-informational?style=flat-square)

## Installing the Chart

### Cert-manager integration

Install [cert-manager](https://cert-manager.io/) via [https://artifacthub.io/packages/helm/cert-manager/cert-manager](https://artifacthub.io/packages/helm/cert-manager/cert-manager)

### Required variables:

```yaml
# -- Global config variables
global:
  # -- Override mongodb uri for every plgd-hub services
  mongoUri:
  # -- Override nats uri for every plgd-hub services
  natsUri:
  # -- Global domain
  domain:
  # -- CloudID. Used by coap-gateway. It must be unique
  cloudId:
  # -- OAuth authority
  authority:
  # -- OAuth audience
  audience:
  # Global OAuth configuration used by multiple services
  oauth:
    # -- OAuth configuration for internal oAuth service client
    service:
      clientID:
      clientSecret:
      scopes: []
      tokenURL:

coapgateway:
  apis:
    coap:
      authorization:
        deviceIdClaim: ""
        providers:
          - clientID: ...
            clientSecret: ...
```

## Requirements

| Repository | Name | Version |
|------------|------|---------|
| https://charts.bitnami.com/bitnami | mongodb | 10.21.2 |
| https://nats-io.github.io/k8s/helm/charts/ | nats | 0.8.2 |

## Values

| Key | Type | Default | Description |
|-----|------|---------|-------------|
| authorization.affinity | object | `{}` | Affinity definition |
| authorization.apis | object | `{"grpc":{"address":null,"authorization":{"audience":null,"authority":null,"http":{"idleConnTimeout":"30s","maxConnsPerHost":32,"maxIdleConns":16,"maxIdleConnsPerHost":16,"timeout":"10s","tls":{"caPool":null,"certFile":null,"keyFile":null,"useSystemCAPool":false}}},"enforcementPolicy":{"minTime":"5s","permitWithoutStream":true},"keepAlive":{"maxConnectionAge":"0s","maxConnectionAgeGrace":"0s","maxConnectionIdle":"0s","time":"2h","timeout":"20s"},"tls":{"caPool":null,"certFile":null,"clientCertificateRequired":true,"keyFile":null}}}` | For complete authorization service configuration see [plgd/authorization](https://github.com/plgd-dev/hub/tree/main/authorization) |
| authorization.clients | object | `{"eventBus":{"nats":{"jetstream":false,"tls":{"useSystemCAPool":false},"url":""}},"storage":{"mongoDB":{"database":"ownersDevices","tls":{"caPool":null,"certFile":null,"keyFile":null,"useSystemCAPool":false},"uri":null},"ownerClaim":"sub"}}` | For complete authorization service configuration see [plgd/authorization](https://github.com/plgd-dev/hub/tree/main/authorization) |
| authorization.config | object | `{"fileName":"service.yaml","mountPath":"/config","volume":"config"}` | yaml configuration |
| authorization.config.fileName | string | `"service.yaml"` | File name |
| authorization.config.mountPath | string | `"/config"` | Service configuration mount path |
| authorization.config.volume | string | `"config"` | Volume name |
| authorization.deploymentAnnotations | object | `{}` | Additional annotations for authorization deployment |
| authorization.deploymentLabels | object | `{}` | Additional labels for authorization deployment |
| authorization.enabled | bool | `true` | Enable authorization service |
| authorization.extraVolumeMounts | object | `{}` | Extra volume mounts |
| authorization.extraVolumes | object | `{}` | Extra volumes |
| authorization.fullnameOverride | string | `nil` | Full name to override |
| authorization.image | object | `{"imagePullSecrets":{},"pullPolicy":"IfNotPresent","registry":null,"repository":"plgd/authorization","tag":null}` | Authorization service image section |
| authorization.image.imagePullSecrets | object | `{}` | Image pull secrets |
| authorization.image.pullPolicy | string | `"IfNotPresent"` | Image pull policy |
| authorization.image.registry | string | `nil` | Image registry |
| authorization.image.repository | string | `"plgd/authorization"` | Image repository |
| authorization.image.tag | string | `nil` | Image tag. |
| authorization.imagePullSecrets | object | `{}` | Image pull secrets |
| authorization.initContainersTpl | object | `{}` | Init containers definition. Resolved as template |
| authorization.livenessProbe | object | `{}` | Liveness probe. authorization doesn't have any default liveness probe |
| authorization.log.debug | bool | `false` | Enable extended log messages |
| authorization.name | string | `"authorization"` | Name of component. Used in label selectors |
| authorization.nodeSelector | object | `{}` | Node selector |
| authorization.podAnnotations | object | `{}` | Annotations for authorization pod |
| authorization.podLabels | object | `{}` | Labels for authorization pod |
| authorization.podSecurityContext | object | `{}` | Pod security context |
| authorization.port | int | `9100` | Service and POD port |
| authorization.rbac | object | `{"enabled":false,"roleBindingDefitionTpl":null,"serviceAccountName":"authorization"}` | RBAC configuration |
| authorization.rbac.enabled | bool | `false` | Enable RBAC setup |
| authorization.rbac.roleBindingDefitionTpl | string | `nil` | Template definition for Role/binding etc.. Resolved as template |
| authorization.rbac.serviceAccountName | string | `"authorization"` | Name of authorization SA |
| authorization.readinessProbe | object | `{}` | Readiness probe. authorization doesn't have aby default readiness probe |
| authorization.replicas | int | `1` | Number of replicas |
| authorization.resources | object | `{}` | Resources limit |
| authorization.restartPolicy | string | `"Always"` | Restart policy for pod |
| authorization.securityContext | object | `{}` | Security context for pod |
| authorization.service | object | `{"annotations":{},"labels":{},"type":"ClusterIP"}` | Service configuration |
| authorization.service.annotations | object | `{}` | Service annotations |
| authorization.service.labels | object | `{}` | Service labels |
| authorization.service.type | string | `"ClusterIP"` | Service type |
| authorization.tolerations | object | `{}` | Toleration definition |
| certificateauthority.affinity | string | `nil` | Affinity definition |
| certificateauthority.apis | object | `{"grpc":{"address":null,"authorization":{"audience":"","authority":"","http":{"idleConnTimeout":"30s","maxConnsPerHost":32,"maxIdleConns":16,"maxIdleConnsPerHost":16,"timeout":"10s","tls":{"caPool":null,"certFile":null,"keyFile":null,"useSystemCAPool":false}},"ownerClaim":"sub"},"enforcementPolicy":{"minTime":"5s","permitWithoutStream":true},"keepAlive":{"maxConnectionAge":"0s","maxConnectionAgeGrace":"0s","maxConnectionIdle":"0s","time":"2h","timeout":"20s"},"tls":{"caPool":null,"certFile":null,"clientCertificateRequired":true,"keyFile":null}}}` | For complete certificate-authority service configuration see [plgd/certificate-authority](https://github.com/plgd-dev/hub/tree/main/certificate-authority) |
| certificateauthority.ca | object | `{"cert":"tls.crt","default":{"commonName":"coap-device-ca","duration":"87600h","renewBefore":"360h","secret":{"name":"coap-device-ca"}},"key":"tls.key","secret":{"name":null},"volume":{"mountPath":"/certs/coap-device-ca","name":"coap-device-ca"}}` | CA section |
| certificateauthority.ca.cert | string | `"tls.crt"` | Cert file name |
| certificateauthority.ca.default | object | `{"commonName":"coap-device-ca","duration":"87600h","renewBefore":"360h","secret":{"name":"coap-device-ca"}}` | Default configuration for cert/key CA used for signing device/identity certificates |
| certificateauthority.ca.default.commonName | string | `"coap-device-ca"` | Common name for CA created as default issuer |
| certificateauthority.ca.default.renewBefore | string | `"360h"` | Renew before for default CA |
| certificateauthority.ca.default.secret.name | string | `"coap-device-ca"` | Name of secret |
| certificateauthority.ca.key | string | `"tls.key"` | Cert key file name |
| certificateauthority.ca.secret.name | string | `nil` | Custom CA secret name |
| certificateauthority.ca.volume.mountPath | string | `"/certs/coap-device-ca"` | CA certificate mount path |
| certificateauthority.ca.volume.name | string | `"coap-device-ca"` | CA certificate volume name |
| certificateauthority.config | object | `{"fileName":"service.yaml","mountPath":"/config","volume":"config"}` | Service configuration |
| certificateauthority.config.fileName | string | `"service.yaml"` | File name for config file |
| certificateauthority.config.mountPath | string | `"/config"` | Mount path |
| certificateauthority.config.volume | string | `"config"` | Config file volume name |
| certificateauthority.deploymentAnnotations | object | `{}` | Additional annotations for certificate-authority deployment |
| certificateauthority.deploymentLabels | object | `{}` | Additional labels for certificate-authority deployment |
| certificateauthority.domain | string | `nil` | External domain for certificate-authority. Default: api.{{ global.domain }} |
| certificateauthority.enabled | bool | `true` | Enable certificate-authority service |
| certificateauthority.extraVolumeMounts | string | `nil` | Optional extra volume mounts |
| certificateauthority.extraVolumes | string | `nil` | Optional extra volumes |
| certificateauthority.fullnameOverride | string | `nil` | Full name to override |
| certificateauthority.image.imagePullSecrets | string | `nil` | Image pull secrets |
| certificateauthority.image.pullPolicy | string | `"Always"` | Image pull policy |
| certificateauthority.image.registry | string | `nil` | Image registry |
| certificateauthority.image.repository | string | `"plgd/certificate-authority"` | Image repository |
| certificateauthority.image.tag | string | `nil` | Image tag. |
| certificateauthority.imagePullSecrets | string | `nil` | Image pull secrets |
| certificateauthority.ingress.annotations | object | `{}` | Ingress annotations |
| certificateauthority.ingress.enabled | bool | `true` | Enable ingress |
| certificateauthority.ingress.paths | list | `["/ocf.cloud.certificateauthority.pb.CertificateAuthority"]` | Paths |
| certificateauthority.initContainersTpl | string | `nil` | Init containers definition |
| certificateauthority.livenessProbe | string | `nil` | Liveness probe. certificate-authority doesn't have any default liveness probe |
| certificateauthority.log.debug | bool | `false` | Enable extended debug messages |
| certificateauthority.name | string | `"certificate-authority"` | Name of component. Used in label selectors |
| certificateauthority.nodeSelector | string | `nil` | Node selector |
| certificateauthority.podAnnotations | object | `{}` | Annotations for certificate-authority pod |
| certificateauthority.podLabels | object | `{}` | Labels for certificate-authority pod |
| certificateauthority.podSecurityContext | object | `{}` | Pod security context |
| certificateauthority.port | int | `9100` | Service and POD port |
| certificateauthority.rbac | object | `{"enabled":false,"roleBindingDefitionTpl":null,"serviceAccountName":"certificate-authority"}` | RBAC configuration |
| certificateauthority.rbac.enabled | bool | `false` | Enable RBAC |
| certificateauthority.rbac.roleBindingDefitionTpl | string | `nil` | Template definition for Role/binding etc.. |
| certificateauthority.rbac.serviceAccountName | string | `"certificate-authority"` | Name of certificate-authority SA |
| certificateauthority.readinessProbe | string | `nil` | Readiness probe. certificate-authority doesn't have aby default readiness probe |
| certificateauthority.replicas | int | `1` | Number of replicas |
| certificateauthority.resources | string | `nil` | Resources limit |
| certificateauthority.restartPolicy | string | `"Always"` | Restart policy for pod |
| certificateauthority.securityContext | string | `nil` | Security context for pod |
| certificateauthority.service.annotations | object | `{}` | Annotations for certificate-authority service |
| certificateauthority.service.labels | object | `{}` | Labels for certificate-authority service |
| certificateauthority.service.type | string | `"ClusterIP"` | Service type |
| certificateauthority.signer | object | `{"certFile":null,"expiresIn":"87600h","keyFile":null,"validFrom":"now-1h"}` | For complete certificate-authority service configuration see [plgd/certificate-authority](https://github.com/plgd-dev/hub/tree/main/certificate-authority) |
| certificateauthority.tolerations | string | `nil` | Toleration definition |
| certmanager | object | `{"coap":{"cert":{"duration":null,"key":{"algorithm":null,"size":null},"renewBefore":null},"issuer":{"annotations":{},"kind":null,"labels":{},"name":null,"spec":null}},"default":{"cert":{"annotations":{},"duration":"8760h","key":{"algorithm":"ECDSA","size":256},"labels":{},"renewBefore":"360h"},"issuer":{"annotations":{},"enabled":true,"kind":"Issuer","labels":{},"name":"selfsigned-issuer","spec":{"selfSigned":{}}}},"enabled":true,"external":{"cert":{"duration":null,"key":{"algorithm":null,"size":null},"renewBefore":null},"issuer":{"annotations":{},"kind":null,"labels":{},"name":null,"spec":null}},"internal":{"cert":{"duration":null,"key":{"algorithm":null,"size":null},"renewBefore":null},"issuer":{"annotations":{},"kind":null,"labels":{},"name":null,"spec":null}}}` | Cert-manager integration section |
| certmanager.coap.cert.duration | string | `nil` | Certificate duration |
| certmanager.coap.cert.key.algorithm | string | `nil` | Certificate key algorithm |
| certmanager.coap.cert.key.size | string | `nil` | Certificate key size |
| certmanager.coap.cert.renewBefore | string | `nil` | Certificate renew before |
| certmanager.coap.issuer.annotations | object | `{}` | Annotations |
| certmanager.coap.issuer.kind | string | `nil` | Kind |
| certmanager.coap.issuer.labels | object | `{}` | Labels |
| certmanager.coap.issuer.name | string | `nil` | Name |
| certmanager.coap.issuer.spec | string | `nil` | cert-manager issuer spec |
| certmanager.default | object | `{"cert":{"annotations":{},"duration":"8760h","key":{"algorithm":"ECDSA","size":256},"labels":{},"renewBefore":"360h"},"issuer":{"annotations":{},"enabled":true,"kind":"Issuer","labels":{},"name":"selfsigned-issuer","spec":{"selfSigned":{}}}}` | Default cert-manager section |
| certmanager.default.cert | object | `{"annotations":{},"duration":"8760h","key":{"algorithm":"ECDSA","size":256},"labels":{},"renewBefore":"360h"}` | Default certificate specification |
| certmanager.default.cert.annotations | object | `{}` | Certificate annotations |
| certmanager.default.cert.duration | string | `"8760h"` | Certificate duration |
| certmanager.default.cert.key | object | `{"algorithm":"ECDSA","size":256}` | Certificate key spec |
| certmanager.default.cert.key.algorithm | string | `"ECDSA"` | Algorithm |
| certmanager.default.cert.key.size | int | `256` | Key size |
| certmanager.default.cert.labels | object | `{}` | Certificate labels |
| certmanager.default.cert.renewBefore | string | `"360h"` | Certificate renew before |
| certmanager.default.issuer | object | `{"annotations":{},"enabled":true,"kind":"Issuer","labels":{},"name":"selfsigned-issuer","spec":{"selfSigned":{}}}` | Default cert-manager issuer |
| certmanager.default.issuer.annotations | object | `{}` | Annotation for default issuer |
| certmanager.default.issuer.enabled | bool | `true` | Enable Default issuer |
| certmanager.default.issuer.labels | object | `{}` | Labels for default issuer |
| certmanager.default.issuer.name | string | `"selfsigned-issuer"` | Name of default issuer |
| certmanager.default.issuer.spec | object | `{"selfSigned":{}}` | Default issuer specification. |
| certmanager.enabled | bool | `true` | Enable cert-manager integration |
| certmanager.external.cert.duration | string | `nil` | Certificate duration |
| certmanager.external.cert.key.algorithm | string | `nil` | Certificate key algorithm |
| certmanager.external.cert.key.size | string | `nil` | Certificate key size |
| certmanager.external.cert.renewBefore | string | `nil` | Certificate renew before |
| certmanager.external.issuer.annotations | object | `{}` | Annotations |
| certmanager.external.issuer.kind | string | `nil` | Kind |
| certmanager.external.issuer.labels | object | `{}` | Labels |
| certmanager.external.issuer.name | string | `nil` | Name |
| certmanager.external.issuer.spec | string | `nil` | cert-manager issuer spec |
| certmanager.internal.cert.duration | string | `nil` | Certificate duration |
| certmanager.internal.cert.key.algorithm | string | `nil` | Certificate key algorithm |
| certmanager.internal.cert.key.size | string | `nil` | Certificate key size |
| certmanager.internal.cert.renewBefore | string | `nil` | Certificate renew before |
| certmanager.internal.issuer | object | `{"annotations":{},"kind":null,"labels":{},"name":null,"spec":null}` | Internal issuer. In case you want to create your own issuer for internal certs |
| certmanager.internal.issuer.annotations | object | `{}` | Annotations |
| certmanager.internal.issuer.kind | string | `nil` | Kind |
| certmanager.internal.issuer.labels | object | `{}` | Labels |
| certmanager.internal.issuer.name | string | `nil` | Name |
| certmanager.internal.issuer.spec | string | `nil` | cert-manager issuer spec |
| cluster.dns | string | `"cluster.local"` | Cluster internal DNS prefix |
| coapgateway | object | `{"affinity":{},"apis":{"coap":{"authorization":{"deviceIdClaim":"","providers":null},"blockwiseTransfer":{"blockSize":"1024","enabled":true},"externalAddress":"","goroutineSocketHeartbeat":"4s","keepAlive":{"timeout":"20s"},"maxMessageSize":262144,"ownerCacheExpiration":"1m","tls":{"caPool":null,"certFile":null,"clientCertificateRequired":true,"enabled":true,"keyFile":null}}},"clients":{"authorizationServer":{"grpc":{"address":"","keepAlive":{"permitWithoutStream":true,"time":"10s","timeout":"20s"},"tls":{"caPool":null,"certFile":null,"keyFile":null,"useSystemCAPool":false}},"ownerClaim":"sub"},"eventBus":{"nats":{"pendingLimits":{"bytesLimit":"67108864","msgLimit":"524288"},"tls":{"caPool":null,"certFile":null,"keyFile":null,"useSystemCAPool":false},"url":""}},"resourceAggregate":{"deviceStatusExpiration":{"enabled":false,"expiresIn":"0s"},"grpc":{"address":"","keepAlive":{"permitWithoutStream":true,"time":"10s","timeout":"20s"},"tls":{"caPool":null,"certFile":null,"keyFile":null,"useSystemCAPool":false}}},"resourceDirectory":{"grpc":{"address":"","keepAlive":{"permitWithoutStream":true,"time":"10s","timeout":"20s"},"tls":{"caPool":null,"certFile":null,"keyFile":null,"useSystemCAPool":false}}}},"cloudId":null,"config":{"fileName":"service.yaml","mountPath":"/config","volume":"config"},"deploymentAnnotations":{},"deploymentLabels":{},"enabled":true,"extraVolumeMounts":{},"extraVolumes":{},"fullnameOverride":null,"image":{"imagePullSecrets":{},"pullPolicy":"Always","registry":null,"repository":"plgd/coap-gateway","tag":null},"imagePullSecrets":{},"initContainersTpl":{},"livenessProbe":{},"log":{"debug":false,"dumpCoapMessages":true},"name":"coap-gateway","nodeSelector":{},"podAnnotations":{},"podLabels":{},"podSecurityContext":{},"port":5684,"rbac":{"enabled":false,"roleBindingDefinitionTpl":null,"serviceAccountName":"coap-gateway"},"readinessProbe":{},"replicas":1,"resources":{},"restartPolicy":"Always","securityContext":{},"service":{"annotations":{},"labels":{},"type":"LoadBalancer"},"taskQueue":{"goPoolSize":1600,"maxIdleTime":"10m","size":"2097152"},"tolerations":{}}` | CoAP gateway parameters |
| coapgateway.affinity | object | `{}` | Affinity definition |
| coapgateway.apis | object | `{"coap":{"authorization":{"deviceIdClaim":"","providers":null},"blockwiseTransfer":{"blockSize":"1024","enabled":true},"externalAddress":"","goroutineSocketHeartbeat":"4s","keepAlive":{"timeout":"20s"},"maxMessageSize":262144,"ownerCacheExpiration":"1m","tls":{"caPool":null,"certFile":null,"clientCertificateRequired":true,"enabled":true,"keyFile":null}}}` | For complete coap-gateway service configuration see [plgd/coap-gateway](https://github.com/plgd-dev/hub/tree/main/coap-gateway) |
| coapgateway.clients | object | `{"authorizationServer":{"grpc":{"address":"","keepAlive":{"permitWithoutStream":true,"time":"10s","timeout":"20s"},"tls":{"caPool":null,"certFile":null,"keyFile":null,"useSystemCAPool":false}},"ownerClaim":"sub"},"eventBus":{"nats":{"pendingLimits":{"bytesLimit":"67108864","msgLimit":"524288"},"tls":{"caPool":null,"certFile":null,"keyFile":null,"useSystemCAPool":false},"url":""}},"resourceAggregate":{"deviceStatusExpiration":{"enabled":false,"expiresIn":"0s"},"grpc":{"address":"","keepAlive":{"permitWithoutStream":true,"time":"10s","timeout":"20s"},"tls":{"caPool":null,"certFile":null,"keyFile":null,"useSystemCAPool":false}}},"resourceDirectory":{"grpc":{"address":"","keepAlive":{"permitWithoutStream":true,"time":"10s","timeout":"20s"},"tls":{"caPool":null,"certFile":null,"keyFile":null,"useSystemCAPool":false}}}}` | For complete coap-gateway service configuration see [plgd/coap-gateway](https://github.com/plgd-dev/hub/tree/main/coap-gateway) |
| coapgateway.cloudId | string | `nil` | Cloud ID. Can be override via global.cloudId |
| coapgateway.config.fileName | string | `"service.yaml"` | Service configuration file name |
| coapgateway.config.mountPath | string | `"/config"` | Configuration mount path |
| coapgateway.config.volume | string | `"config"` | Volume name |
| coapgateway.deploymentAnnotations | object | `{}` | Additional annotations for coap-gateway deployment |
| coapgateway.deploymentLabels | object | `{}` | Additional labels for coap-gateway deployment |
| coapgateway.enabled | bool | `true` | Enable coap-gateway service |
| coapgateway.extraVolumeMounts | object | `{}` | Optional extra volume mounts |
| coapgateway.extraVolumes | object | `{}` | Optional extra volumes |
| coapgateway.fullnameOverride | string | `nil` | Full name to override |
| coapgateway.image.imagePullSecrets | object | `{}` | Image pull secrets |
| coapgateway.image.pullPolicy | string | `"Always"` | Image pull policy |
| coapgateway.image.registry | string | `nil` | Image registry |
| coapgateway.image.repository | string | `"plgd/coap-gateway"` | Image repository |
| coapgateway.image.tag | string | `nil` | Image tag |
| coapgateway.imagePullSecrets | object | `{}` | Image pull secrets |
| coapgateway.initContainersTpl | object | `{}` | Init containers definition |
| coapgateway.livenessProbe | object | `{}` | Liveness probe. coap-gateway doesn't have any default liveness probe |
| coapgateway.log.debug | bool | `false` | Enable extended log messages |
| coapgateway.log.dumpCoapMessages | bool | `true` | Dump copp messages |
| coapgateway.name | string | `"coap-gateway"` | Name of component. Used in label selectors |
| coapgateway.nodeSelector | object | `{}` | Node selector |
| coapgateway.podAnnotations | object | `{}` | Annotations for coap-gateway pod |
| coapgateway.podLabels | object | `{}` | Labels for coap-gateway pod |
| coapgateway.podSecurityContext | object | `{}` | Pod security context |
| coapgateway.port | int | `5684` | Service and POD port |
| coapgateway.rbac | object | `{"enabled":false,"roleBindingDefinitionTpl":null,"serviceAccountName":"coap-gateway"}` | RBAC configuration |
| coapgateway.rbac.enabled | bool | `false` | Create RBAC config |
| coapgateway.rbac.roleBindingDefinitionTpl | string | `nil` | template definition for Role/binding etc.. |
| coapgateway.rbac.serviceAccountName | string | `"coap-gateway"` | Name of coap-gateway SA |
| coapgateway.readinessProbe | object | `{}` | Readiness probe. coap-gateway doesn't have aby default readiness probe |
| coapgateway.replicas | int | `1` | Number of replicas |
| coapgateway.resources | object | `{}` | Resources limit |
| coapgateway.restartPolicy | string | `"Always"` | Restart policy for pod |
| coapgateway.securityContext | object | `{}` | Security context for pod |
| coapgateway.service.annotations | object | `{}` | Annotations for coap-gateway service |
| coapgateway.service.labels | object | `{}` | Labels for coap-gateway service |
| coapgateway.service.type | string | `"LoadBalancer"` | Service type |
| coapgateway.taskQueue | object | `{"goPoolSize":1600,"maxIdleTime":"10m","size":"2097152"}` | For complete coap-gateway service configuration see [plgd/coap-gateway](https://github.com/plgd-dev/hub/tree/main/coap-gateway) |
| coapgateway.tolerations | object | `{}` | Toleration definition |
| extraDeploy | string | `nil` | Extra deploy. Resolved as template |
| global | object | `{"audience":null,"authority":null,"cloudId":null,"domain":null,"oauth":{"service":{"clientID":null,"clientSecret":null,"scopes":[],"tokenURL":null}}}` | Global config variables |
| global.audience | string | `nil` | OAuth audience |
| global.authority | string | `nil` | OAuth authority |
| global.cloudId | string | `nil` | CloudID. Used by coap-gateway. It must be unique |
| global.domain | string | `nil` | Global domain |
| global.oauth.service | object | `{"clientID":null,"clientSecret":null,"scopes":[],"tokenURL":null}` | OAuth configuration for internal oAuth service client |
| grpcgateway.affinity | object | `{}` | Affinity definition |
| grpcgateway.apis | object | `{"grpc":{"address":null,"authorization":{"audience":"","authority":"","http":{"idleConnTimeout":"30s","maxConnsPerHost":32,"maxIdleConns":16,"maxIdleConnsPerHost":16,"timeout":"10s","tls":{"caPool":null,"certFile":null,"keyFile":null,"useSystemCAPool":false}}},"enforcementPolicy":{"minTime":"5s","permitWithoutStream":true},"keepAlive":{"maxConnectionAge":"0s","maxConnectionAgeGrace":"0s","maxConnectionIdle":"0s","time":"2h","timeout":"20s"},"ownerCacheExpiration":"1m","tls":{"caPool":null,"certFile":null,"clientCertificateRequired":true,"keyFile":null}}}` | For complete grpc-gateway service configuration see [plgd/grpc-gateway](https://github.com/plgd-dev/hub/tree/main/grpc-gateway) |
| grpcgateway.clients | object | `{"authorizationServer":{"grpc":{"address":"","keepAlive":{"permitWithoutStream":true,"time":"10s","timeout":"20s"},"tls":{"caPool":null,"certFile":null,"keyFile":null,"useSystemCAPool":false}},"ownerClaim":"sub"},"eventBus":{"goPoolSize":16,"nats":{"pendingLimits":{"bytesLimit":67108864,"msgLimit":524288},"tls":{"caPool":null,"certFile":null,"keyFile":null,"useSystemCAPool":false},"url":null}},"resourceAggregate":{"grpc":{"address":"","keepAlive":{"permitWithoutStream":true,"time":"10s","timeout":"20s"},"tls":{"caPool":null,"certFile":null,"keyFile":null,"useSystemCAPool":false}}},"resourceDirectory":{"grpc":{"address":"","keepAlive":{"permitWithoutStream":true,"time":"10s","timeout":"20s"},"tls":{"caPool":null,"certFile":null,"keyFile":null,"useSystemCAPool":false}}}}` | For complete grpc-gateway service configuration see [plgd/grpc-gateway](https://github.com/plgd-dev/hub/tree/main/grpc-gateway) |
| grpcgateway.config | object | `{"fileName":"service.yaml","mountPath":"/config","volume":"config"}` | Service yaml configuration section |
| grpcgateway.config.fileName | string | `"service.yaml"` | Service configuration file name |
| grpcgateway.config.mountPath | string | `"/config"` | Service configuration mount path |
| grpcgateway.config.volume | string | `"config"` | Service configuration volume name |
| grpcgateway.deploymentAnnotations | object | `{}` | Additional annotations for grpc-gateway deployment |
| grpcgateway.deploymentLabels | object | `{}` | Additional labels for grpc-gateway deployment |
| grpcgateway.domain | string | `nil` | External domain for grpc-gateway. Default: api.{{ global.domain }} |
| grpcgateway.enabled | bool | `true` | Enable grpc-gateway service |
| grpcgateway.extraVolumeMounts | object | `{}` | Optional extra volume mounts |
| grpcgateway.extraVolumes | object | `{}` | Optional extra volumes |
| grpcgateway.fullnameOverride | string | `nil` | Full name to override |
| grpcgateway.image.imagePullSecrets | object | `{}` | Image pull secrets |
| grpcgateway.image.pullPolicy | string | `"Always"` | Image pull policy |
| grpcgateway.image.registry | string | `nil` | Image registry |
| grpcgateway.image.repository | string | `"plgd/grpc-gateway"` | Image repository |
| grpcgateway.image.tag | string | `nil` | Image tag. |
| grpcgateway.imagePullSecrets | object | `{}` | Image pull secrets |
| grpcgateway.ingress.annotations | object | `{}` | Ingress annotations |
| grpcgateway.ingress.enabled | bool | `true` | Enable ingress |
| grpcgateway.ingress.paths | list | `["/ocf.cloud.grpcgateway.pb.GrpcGateway"]` | Default ingress paths |
| grpcgateway.initContainersTpl | object | `{}` | Init containers definition |
| grpcgateway.livenessProbe | object | `{}` | Liveness probe. grpc-gateway doesn't have any default liveness probe |
| grpcgateway.log.debug | bool | `false` | Enable extended log messages |
| grpcgateway.name | string | `"grpc-gateway"` | Name of component. Used in label selectors |
| grpcgateway.nodeSelector | object | `{}` | Node selector |
| grpcgateway.podAnnotations | object | `{}` | Annotations for grpc-gateway pod |
| grpcgateway.podLabels | object | `{}` | Labels for grpc-gateway pod |
| grpcgateway.podSecurityContext | object | `{}` | Pod security context |
| grpcgateway.port | int | `9100` | Service and POD port |
| grpcgateway.rbac | object | `{"enabled":false,"roleBindingDefitionTpl":null,"serviceAccountName":"grpc-gateway"}` | RBAC configuration |
| grpcgateway.rbac.roleBindingDefitionTpl | string | `nil` | Template definition for Role/binding etc.. |
| grpcgateway.rbac.serviceAccountName | string | `"grpc-gateway"` | Name of grpc-gateway SA |
| grpcgateway.readinessProbe | object | `{}` | Readiness probe. grpc-gateway doesn't have aby default readiness probe |
| grpcgateway.replicas | int | `1` | Number of replicas |
| grpcgateway.resources | object | `{}` | Resources limit |
| grpcgateway.restartPolicy | string | `"Always"` | Restart policy for pod |
| grpcgateway.securityContext | object | `{}` | Security context for pod |
| grpcgateway.service.annotations | object | `{}` | Annotations for grpc-gateway service |
| grpcgateway.service.labels | object | `{}` | Labels for grpc-gateway service |
| grpcgateway.service.type | string | `"ClusterIP"` | Service type |
| grpcgateway.tolerations | object | `{}` | Toleration definition |
| httpgateway.affinity | object | `{}` | Affinity definition |
| httpgateway.apis | object | `{"http":{"address":null,"authorization":{"audience":"","authority":"","http":{"idleConnTimeout":"30s","maxConnsPerHost":32,"maxIdleConns":16,"maxIdleConnsPerHost":16,"timeout":"10s","tls":{"caPool":null,"certFile":null,"keyFile":null,"useSystemCAPool":false}}},"tls":{"caPool":null,"certFile":null,"clientCertificateRequired":true,"keyFile":null},"webSocket":{"pingFrequency":"10s","streamBodyLimit":262144}}}` | For complete http-gateway service configuration see [plgd/http-gateway](https://github.com/plgd-dev/hub/tree/main/http-gateway) |
| httpgateway.clients | object | `{"grpcGateway":{"grpc":{"address":"","keepAlive":{"permitWithoutStream":true,"time":"10s","timeout":"20s"},"tls":{"caPool":null,"certFile":null,"keyFile":null,"useSystemCAPool":false}}}}` | For complete http-gateway service configuration see [plgd/http-gateway](https://github.com/plgd-dev/hub/tree/main/http-gateway) |
| httpgateway.config | object | `{"fileName":"service.yaml","mountPath":"/config","volume":"config"}` | Http-gateway service yaml config section |
| httpgateway.config.fileName | string | `"service.yaml"` | Name of configuration file |
| httpgateway.config.mountPath | string | `"/config"` | Mount path |
| httpgateway.config.volume | string | `"config"` | Volume for configuration file |
| httpgateway.deploymentAnnotations | object | `{}` | Additional annotations for http-gateway deployment |
| httpgateway.deploymentLabels | object | `{}` | Additional labels for http-gateway deployment |
| httpgateway.domain | string | `nil` | Http-gateway domain. Default: api.{{ global.domain }} |
| httpgateway.enabled | bool | `true` | Enable http-gateway service |
| httpgateway.extraVolumeMounts | object | `{}` | Optional extra volume mounts |
| httpgateway.extraVolumes | object | `{}` | Optional extra volumes |
| httpgateway.fullnameOverride | string | `nil` | Full name to override |
| httpgateway.image.imagePullSecrets | object | `{}` | Image pull secrets |
| httpgateway.image.pullPolicy | string | `"Always"` | Image pull policy |
| httpgateway.image.registry | string | `nil` | Image registry |
| httpgateway.image.repository | string | `"plgd/http-gateway"` | Image repository |
| httpgateway.image.tag | string | `nil` | Image tag. |
| httpgateway.imagePullSecrets | object | `{}` | Image pull secrets |
| httpgateway.ingress.annotations | object | `{}` | Ingress annotation |
| httpgateway.ingress.enabled | bool | `true` | Enable ingress |
| httpgateway.ingress.paths | list | `["/api","/.well-known/"]` | Ingress path |
| httpgateway.initContainersTpl | object | `{}` | Init containers definition. Render as template |
| httpgateway.livenessProbe | object | `{}` | Liveness probe. http-gateway doesn't have any default liveness probe |
| httpgateway.log.debug | bool | `false` | Enable extended debug messages |
| httpgateway.name | string | `"http-gateway"` | Name of component. Used in label selectors |
| httpgateway.nodeSelector | object | `{}` | Node selector |
| httpgateway.podAnnotations | object | `{}` | Annotations for http-gateway pod |
| httpgateway.podLabels | object | `{}` | Labels for http-gateway pod |
| httpgateway.podSecurityContext | object | `{}` | Pod security context |
| httpgateway.port | int | `9100` | Port for service and POD |
| httpgateway.rbac | object | `{"enabled":false,"roleBindingDefitionTpl":null,"serviceAccountName":"http-gateway"}` | RBAC configuration |
| httpgateway.rbac.enabled | bool | `false` | Enable RBAC setup |
| httpgateway.rbac.roleBindingDefitionTpl | string | `nil` | Definition for Role/binding etc.. Render as template |
| httpgateway.rbac.serviceAccountName | string | `"http-gateway"` | Name of http-gateway SA |
| httpgateway.readinessProbe | object | `{}` | Readiness probe. http-gateway doesn't have aby default readiness probe |
| httpgateway.replicas | int | `1` | Number of replicas |
| httpgateway.resources | object | `{}` | Resources limit |
| httpgateway.restartPolicy | string | `"Always"` | Restart policy for pod |
| httpgateway.securityContext | object | `{}` | Security context for pod |
| httpgateway.service.annotations | object | `{}` | Annotations for http-gateway service |
| httpgateway.service.labels | object | `{}` | Labels for http-gateway service |
| httpgateway.service.type | string | `"ClusterIP"` |  |
| httpgateway.tolerations | object | `{}` | Toleration definition |
| httpgateway.ui | object | `{"directory":"/usr/local/var/www","enabled":false,"webConfiguration":{"deviceOAuthClient":{"audience":"","clientID":"","providerName":"plgd","scopes":[]},"domain":"","httpGatewayAddress":"","webOAuthClient":{"audience":"","clientID":"","scopes":[]}}}` | For complete http-gateway service configuration see [plgd/http-gateway](https://github.com/plgd-dev/hub/tree/main/http-gateway) |
| mongodb | object | `{"arbiter":{"enabled":false},"architecture":"replicaset","auth":{"enabled":false},"customLivenessProbe":{"exec":{"command":["mongo","--disableImplicitSessions","--tls","--tlsCertificateKeyFile=/certs/cert.pem","--tlsCAFile=/certs/ca.pem","--eval","db.adminCommand('ping')"]},"failureThreshold":6,"initialDelaySeconds":30,"periodSeconds":10,"successThreshold":1,"timeoutSeconds":5},"customReadinessProbe":{"exec":{"command":["bash","-ec","TLS_OPTIONS='--tls --tlsCertificateKeyFile=/certs/cert.pem --tlsCAFile=/certs/ca.pem'\nmongo --disableImplicitSessions $TLS_OPTIONS --eval 'db.hello().isWritablePrimary || db.hello().secondary' | grep -q 'true'\n"]},"failureThreshold":6,"initialDelaySeconds":5,"periodSeconds":10,"successThreshold":1,"timeoutSeconds":5},"enabled":true,"extraEnvVars":[{"name":"MONGODB_EXTRA_FLAGS","value":"--tlsMode=requireTLS --tlsCertificateKeyFile=/certs/cert.pem --tlsCAFile=/certs/ca.pem"},{"name":"MONGODB_CLIENT_EXTRA_FLAGS","value":"--tls --tlsCertificateKeyFile=/certs/cert.pem --tlsCAFile=/certs/ca.pem"}],"extraVolumeMounts":[{"mountPath":"/certs","name":"mongodb-crt"}],"extraVolumes":[{"emptyDir":{},"name":"mongodb-crt"},{"name":"mongodb-cm-crt","secret":{"secretName":"mongodb-cm-crt"}}],"fullnameOverride":"mongodb","image":{"debug":true,"net":{"port":27017}},"initContainers":[{"command":["sh","-c","/bin/bash <<'EOF'\ncat /tmp/certs/tls.crt >> /certs/cert.pem\ncat /tmp/certs/tls.key >> /certs/cert.pem\ncp /tmp/certs/ca.crt  /certs/ca.pem\nEOF\n"],"image":"docker.io/bitnami/nginx:1.19.10-debian-10-r63","imagePullPolicy":"IfNotPresent","name":"convert-cm-crt","volumeMounts":[{"mountPath":"/certs","name":"mongodb-crt"},{"mountPath":"/tmp/certs","name":"mongodb-cm-crt"}]}],"livenessProbe":{"enabled":false},"persistence":{"enabled":true},"readinessProbe":{"enabled":false},"replicaCount":3,"replicaSetName":"rs0","tls":{"enabled":false}}` | External mongodb-replica dependency setup |
| nats | object | `{"cluster":{"enabled":false,"noAdvertise":false},"enabled":true,"leafnodes":{"enabled":false,"noAdvertise":false},"nats":{"tls":{"ca":"ca.crt","cert":"tls.crt","key":"tls.key","secret":{"name":"nats-service-crt"},"verify":true}},"natsbox":{"enabled":false}}` | External nats dependency setup |
| resourceaggregate.affinity | object | `{}` | Affinity definition |
| resourceaggregate.apis.grpc.authorization.audience | string | `""` |  |
| resourceaggregate.apis.grpc.authorization.authority | string | `""` |  |
| resourceaggregate.apis.grpc.authorization.http.idleConnTimeout | string | `"30s"` |  |
| resourceaggregate.apis.grpc.authorization.http.maxConnsPerHost | int | `32` |  |
| resourceaggregate.apis.grpc.authorization.http.maxIdleConns | int | `16` |  |
| resourceaggregate.apis.grpc.authorization.http.maxIdleConnsPerHost | int | `16` |  |
| resourceaggregate.apis.grpc.authorization.http.timeout | string | `"10s"` |  |
| resourceaggregate.apis.grpc.authorization.http.tls.caPool | string | `""` |  |
| resourceaggregate.apis.grpc.authorization.http.tls.certFile | string | `""` |  |
| resourceaggregate.apis.grpc.authorization.http.tls.keyFile | string | `""` |  |
| resourceaggregate.apis.grpc.authorization.http.tls.useSystemCAPool | bool | `false` |  |
| resourceaggregate.apis.grpc.enforcementPolicy.minTime | string | `"5s"` |  |
| resourceaggregate.apis.grpc.enforcementPolicy.permitWithoutStream | bool | `true` |  |
| resourceaggregate.apis.grpc.keepAlive.maxConnectionAge | string | `"0s"` |  |
| resourceaggregate.apis.grpc.keepAlive.maxConnectionAgeGrace | string | `"0s"` |  |
| resourceaggregate.apis.grpc.keepAlive.maxConnectionIdle | string | `"0s"` |  |
| resourceaggregate.apis.grpc.keepAlive.time | string | `"2h"` |  |
| resourceaggregate.apis.grpc.keepAlive.timeout | string | `"20s"` |  |
| resourceaggregate.apis.grpc.ownerCacheExpiration | string | `"1m"` |  |
| resourceaggregate.apis.grpc.tls.caPool | string | `""` |  |
| resourceaggregate.apis.grpc.tls.certFile | string | `""` |  |
| resourceaggregate.apis.grpc.tls.clientCertificateRequired | bool | `true` |  |
| resourceaggregate.apis.grpc.tls.keyFile | string | `""` |  |
| resourceaggregate.clients | object | `{"authorizationServer":{"cacheExpiration":"1m","grpc":{"address":"","keepAlive":{"permitWithoutStream":true,"time":"10s","timeout":"20s"},"tls":{"caPool":"","certFile":"","keyFile":"","useSystemCAPool":false}},"oauth":{"audience":"","clientID":"","clientSecret":"","http":{"idleConnTimeout":"30s","maxConnsPerHost":32,"maxIdleConns":16,"maxIdleConnsPerHost":16,"timeout":"10s","tls":{"caPool":"","certFile":"","keyFile":"","useSystemCAPool":false}},"scopes":[],"tokenURL":"","verifyServiceTokenFrequency":"10s"},"ownerClaim":"sub","pullFrequency":"15s"},"eventBus":{"nats":{"jetstream":false,"pendingLimits":{"bytesLimit":67108864,"msgLimit":524288},"tls":{"caPool":"","certFile":"","keyFile":"","useSystemCAPool":false},"url":""}},"eventStore":{"defaultCommandTTL":"0s","mongoDB":{"batchSize":128,"database":"eventStore","maxConnIdleTime":"4m0s","maxPoolSize":16,"tls":{"caPool":"","certFile":"","keyFile":"","useSystemCAPool":false},"uri":null},"occMaxRetry":8,"snapshotThreshold":16}}` | For complete resource-aggregate service configuration see [plgd/resource-aggregate](https://github.com/plgd-dev/hub/tree/main/resource-aggregate) |
| resourceaggregate.config | object | `{"fileName":"service.yaml","mountPath":"/config","volume":"config"}` | Service configuration |
| resourceaggregate.config.fileName | string | `"service.yaml"` | Service configuration file name |
| resourceaggregate.config.mountPath | string | `"/config"` | Configuration mount path |
| resourceaggregate.config.volume | string | `"config"` | Volume name |
| resourceaggregate.deploymentAnnotations | object | `{}` | Additional annotations for resource-aggregate deployment |
| resourceaggregate.deploymentLabels | object | `{}` | Additional labels for resource-aggregate deployment |
| resourceaggregate.enabled | bool | `true` | Enable resource-aggregate service |
| resourceaggregate.extraVolumeMounts | object | `{}` | Optional extra volume mounts |
| resourceaggregate.extraVolumes | object | `{}` | Optional extra volumes |
| resourceaggregate.fullnameOverride | string | `nil` | Full name to override |
| resourceaggregate.image.imagePullSecrets | object | `{}` | Image pull secrets |
| resourceaggregate.image.pullPolicy | string | `"Always"` | Image pull policy |
| resourceaggregate.image.registry | string | `nil` | Image registry |
| resourceaggregate.image.repository | string | `"plgd/resource-aggregate"` | Image repository |
| resourceaggregate.image.tag | string | `nil` | Image tag. |
| resourceaggregate.imagePullSecrets | object | `{}` | Image pull secrets |
| resourceaggregate.initContainersTpl | object | `{}` | Init containers definition. Resolved as template |
| resourceaggregate.livenessProbe | object | `{}` | Liveness probe. resource-aggregate doesn't have any default liveness probe |
| resourceaggregate.log.debug | bool | `false` | Enable extended message log |
| resourceaggregate.name | string | `"resource-aggregate"` | Name of component. Used in label selectors |
| resourceaggregate.nodeSelector | object | `{}` | Node selector |
| resourceaggregate.podAnnotations | object | `{}` | Annotations for resource-aggregate pod |
| resourceaggregate.podLabels | object | `{}` | Labels for resource-aggregate pod |
| resourceaggregate.podSecurityContext | object | `{}` | Pod security context |
| resourceaggregate.port | int | `9100` | Service and POD port |
| resourceaggregate.rbac | object | `{"enabled":false,"roleBindingDefitionTpl":null,"serviceAccountName":"resource-aggregate"}` | RBAC configuration |
| resourceaggregate.rbac.enabled | bool | `false` | Create RBAC config |
| resourceaggregate.rbac.roleBindingDefitionTpl | string | `nil` | template definition for Role/binding etc.. |
| resourceaggregate.rbac.serviceAccountName | string | `"resource-aggregate"` | Name of resource-aggregate SA |
| resourceaggregate.readinessProbe | object | `{}` | Readiness probe. resource-aggregate doesn't have aby default readiness probe |
| resourceaggregate.replicas | int | `1` | Number of replicas |
| resourceaggregate.resources | object | `{}` | Resources limit |
| resourceaggregate.restartPolicy | string | `"Always"` | Restart policy for pod |
| resourceaggregate.securityContext | object | `{}` | Security context for pod |
| resourceaggregate.service.annotations | object | `{}` | Annotations for resource-aggregate service |
| resourceaggregate.service.labels | object | `{}` | Labels for resource-aggregate service |
| resourceaggregate.service.type | string | `"ClusterIP"` | Service type |
| resourceaggregate.tolerations | object | `{}` | Toleration definition |
| resourcedirectory.affinity | object | `{}` | Affinity definition |
| resourcedirectory.apis | object | `{"grpc":{"address":null,"authorization":{"audience":"","authority":"","http":{"idleConnTimeout":"30s","maxConnsPerHost":32,"maxIdleConns":16,"maxIdleConnsPerHost":16,"timeout":"10s","tls":{"useSystemCAPool":false}}},"enforcementPolicy":{"minTime":"5s","permitWithoutStream":true},"keepAlive":{"maxConnectionAge":"0s","maxConnectionAgeGrace":"0s","maxConnectionIdle":"0s","time":"2h","timeout":"20s"},"tls":{"clientCertificateRequired":true}}}` | For complete resource-directory service configuration see [plgd/resource-directory](https://github.com/plgd-dev/hub/tree/main/resource-directory) |
| resourcedirectory.clients | object | `{"authorizationServer":{"cacheExpiration":"1m","grpc":{"address":"","keepAlive":{"permitWithoutStream":true,"time":"10s","timeout":"20s"},"tls":{"caPool":null,"certFile":null,"keyFile":null,"useSystemCAPool":false}},"oauth":{"audience":"","clientID":null,"clientSecret":null,"http":{"idleConnTimeout":"30s","maxConnsPerHost":32,"maxIdleConns":16,"maxIdleConnsPerHost":16,"timeout":"10s","tls":{"caPool":null,"certFile":null,"keyFile":null,"useSystemCAPool":false}},"scopes":[],"tokenURL":"","verifyServiceTokenFrequency":"10s"},"ownerClaim":"sub","pullFrequency":"15s"},"eventBus":{"goPoolSize":16,"nats":{"jetstream":false,"pendingLimits":{"bytesLimit":"67108864","msgLimit":"524288"},"tls":{"caPool":null,"certFile":null,"keyFile":null,"useSystemCAPool":false},"url":""}},"eventStore":{"cacheExpiration":"20m","mongoDB":{"batchSize":128,"database":"eventStore","maxConnIdleTime":"4m0s","maxPoolSize":16,"tls":{"caPool":null,"certFile":null,"keyFile":null,"useSystemCAPool":false},"uri":""}}}` | For complete resource-directory service configuration see [plgd/resource-directory](https://github.com/plgd-dev/hub/tree/main/resource-directory) |
| resourcedirectory.config | object | `{"fileName":"service.yaml","mountPath":"/config","volume":"config"}` | Service configuration |
| resourcedirectory.config.fileName | string | `"service.yaml"` | Service configuration file |
| resourcedirectory.config.mountPath | string | `"/config"` | Configuration mount path |
| resourcedirectory.config.volume | string | `"config"` | Service configuration volume name |
| resourcedirectory.deploymentAnnotations | object | `{}` | Additional annotations for resource-directory deployment |
| resourcedirectory.deploymentLabels | object | `{}` | Additional labels for resource-directory deployment |
| resourcedirectory.enabled | bool | `true` | Enable resource-directory service |
| resourcedirectory.extraVolumeMounts | object | `{}` | Optional extra volume mounts |
| resourcedirectory.extraVolumes | object | `{}` | Optional extra volumes |
| resourcedirectory.fullnameOverride | string | `nil` | Full name to override |
| resourcedirectory.image.command | string | `nil` | Container command |
| resourcedirectory.image.imagePullSecrets | object | `{}` |  |
| resourcedirectory.image.pullPolicy | string | `"Always"` | Image pull policy |
| resourcedirectory.image.registry | string | `nil` | Image registry |
| resourcedirectory.image.repository | string | `"plgd/resource-directory"` | Image repository |
| resourcedirectory.image.tag | string | `nil` |  |
| resourcedirectory.initContainersTpl | object | `{}` | Init containers definition. Resolved as template |
| resourcedirectory.livenessProbe | object | `{}` | Liveness probe. resource-directory doesn't have any default liveness probe |
| resourcedirectory.log | object | `{"debug":false}` | Log section |
| resourcedirectory.log.debug | bool | `false` | Enable extended log messages |
| resourcedirectory.name | string | `"resource-directory"` | Name of component. Used in label selectors |
| resourcedirectory.nodeSelector | object | `{}` | Node selector |
| resourcedirectory.podAnnotations | object | `{}` | Annotations for resource-directory pod |
| resourcedirectory.podLabels | object | `{}` | Labels for resource-directory pod |
| resourcedirectory.podSecurityContext | object | `{}` | Pod security context |
| resourcedirectory.port | int | `9100` | Service and POD port |
| resourcedirectory.publicConfiguration | object | `{"authorizationServer":null,"caPool":null,"cloudAuthorizationProvider":"plgd","cloudID":null,"cloudURL":null,"defaultCommandTimeToLive":"0s","ownerClaim":"sub","signingServerAddress":null}` | For complete resource-directory service configuration see [plgd/resource-directory](https://github.com/plgd-dev/hub/tree/main/resource-directory) |
| resourcedirectory.rbac | object | `{"enabled":false,"roleBindingDefitionTpl":null,"serviceAccountName":"resource-directory"}` | RBAC configuration |
| resourcedirectory.rbac.roleBindingDefitionTpl | string | `nil` | template definition for Role/binding etc.. |
| resourcedirectory.rbac.serviceAccountName | string | `"resource-directory"` | Name of resource-directory SA |
| resourcedirectory.readinessProbe | object | `{}` | Readiness probe. resource-directory doesn't have aby default readiness probe |
| resourcedirectory.replicas | int | `1` | Number of replicas |
| resourcedirectory.resources | object | `{}` | Resources limit |
| resourcedirectory.restartPolicy | string | `"Always"` |  |
| resourcedirectory.securityContext | object | `{}` | Security context for pod |
| resourcedirectory.service.annotations | object | `{}` | Annotations for resource-directory service |
| resourcedirectory.service.labels | object | `{}` | Labels for resource-directory service |
| resourcedirectory.service.type | string | `"ClusterIP"` | resource-directory service type |
| resourcedirectory.tolerations | object | `{}` | Toleration definition |

----------------------------------------------
Autogenerated from chart metadata using [helm-docs v1.5.0](https://github.com/norwoodj/helm-docs/releases/v1.5.0)
