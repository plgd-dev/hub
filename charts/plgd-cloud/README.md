# plgd-cloud

A Helm chart for plgd-cloud

![Version: 0.0.1](https://img.shields.io/badge/Version-0.0.1-informational?style=flat-square) ![Type: application](https://img.shields.io/badge/Type-application-informational?style=flat-square) ![AppVersion: v2next](https://img.shields.io/badge/AppVersion-v2next-informational?style=flat-square)

## Additional Information

## Installing the Chart

## Requirements

| Repository | Name | Version |
|------------|------|---------|
| https://charts.bitnami.com/bitnami | mongodb | 10.21.2 |
| https://nats-io.github.io/k8s/helm/charts/ | nats | 0.8.2 |

## Values

| Key | Type | Default | Description |
|-----|------|---------|-------------|
| authorization.affinity | object | `{}` | Affinity definition |
| authorization.apis | object | `{"grpc":{"address":null,"authorization":{"audience":null,"authority":null,"http":{"idleConnTimeout":"30s","maxConnsPerHost":32,"maxIdleConns":16,"maxIdleConnsPerHost":16,"timeout":"10s","tls":{"caPool":null,"certFile":null,"keyFile":null,"useSystemCAPool":false}}},"enforcementPolicy":{"minTime":"5s","permitWithoutStream":true},"keepAlive":{"maxConnectionAge":"0s","maxConnectionAgeGrace":"0s","maxConnectionIdle":"0s","time":"2h","timeout":"20s"},"tls":{"caPool":null,"certFile":null,"clientCertificateRequired":true,"keyFile":null}}}` | For complete authorization service configuration see [plgd/authorization](https://github.com/plgd-dev/cloud/tree/v2/authorization) |
| authorization.clients | object | `{"eventBus":{"nats":{"jetstream":false,"tls":{"useSystemCAPool":false},"url":""}},"storage":{"mongoDB":{"database":"ownersDevices","tls":{"caPool":null,"certFile":null,"keyFile":null,"useSystemCAPool":false},"uri":null},"ownerClaim":"sub"}}` | For complete authorization service configuration see [plgd/authorization](https://github.com/plgd-dev/cloud/tree/v2/authorization) |
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
| certificateauthority.apis | object | `{"grpc":{"address":null,"authorization":{"audience":"","authority":"","http":{"idleConnTimeout":"30s","maxConnsPerHost":32,"maxIdleConns":16,"maxIdleConnsPerHost":16,"timeout":"10s","tls":{"caPool":null,"certFile":null,"keyFile":null,"useSystemCAPool":false}},"ownerClaim":"sub"},"enforcementPolicy":{"minTime":"5s","permitWithoutStream":true},"keepAlive":{"maxConnectionAge":"0s","maxConnectionAgeGrace":"0s","maxConnectionIdle":"0s","time":"2h","timeout":"20s"},"tls":{"caPool":null,"certFile":null,"clientCertificateRequired":true,"keyFile":null}}}` | For complete certificate-authority service configuration see [plgd/certificate-authority](https://github.com/plgd-dev/cloud/tree/v2/certificate-authority) |
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
| certificateauthority.signer | object | `{"certFile":null,"expiresIn":"87600h","keyFile":null,"validFrom":"now-1h"}` | For complete certificate-authority service configuration see [plgd/certificate-authority](https://github.com/plgd-dev/cloud/tree/v2/certificate-authority) |
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
| coapgateway.affinity | object | `{}` |  |
| coapgateway.apis.coap.authorization.audience | string | `nil` |  |
| coapgateway.apis.coap.authorization.authority | string | `nil` |  |
| coapgateway.apis.coap.authorization.clientID | string | `nil` |  |
| coapgateway.apis.coap.authorization.clientSecret | string | `nil` |  |
| coapgateway.apis.coap.authorization.http.idleConnTimeout | string | `"30s"` |  |
| coapgateway.apis.coap.authorization.http.maxConnsPerHost | int | `32` |  |
| coapgateway.apis.coap.authorization.http.maxIdleConns | int | `16` |  |
| coapgateway.apis.coap.authorization.http.maxIdleConnsPerHost | int | `16` |  |
| coapgateway.apis.coap.authorization.http.timeout | string | `"10s"` |  |
| coapgateway.apis.coap.authorization.http.tls.caPool | string | `nil` |  |
| coapgateway.apis.coap.authorization.http.tls.certFile | string | `nil` |  |
| coapgateway.apis.coap.authorization.http.tls.keyFile | string | `nil` |  |
| coapgateway.apis.coap.authorization.http.tls.useSystemCAPool | bool | `true` |  |
| coapgateway.apis.coap.authorization.redirectURL | string | `nil` |  |
| coapgateway.apis.coap.authorization.scopes | list | `[]` |  |
| coapgateway.apis.coap.blockwiseTransfer.blockSize | string | `"1024"` |  |
| coapgateway.apis.coap.blockwiseTransfer.enabled | bool | `true` |  |
| coapgateway.apis.coap.externalAddress | string | `""` |  |
| coapgateway.apis.coap.goroutineSocketHeartbeat | string | `"4s"` |  |
| coapgateway.apis.coap.keepAlive.timeout | string | `"20s"` |  |
| coapgateway.apis.coap.maxMessageSize | int | `262144` |  |
| coapgateway.apis.coap.ownerCacheExpiration | string | `"1m"` |  |
| coapgateway.apis.coap.tls.caPool | string | `nil` |  |
| coapgateway.apis.coap.tls.certFile | string | `nil` |  |
| coapgateway.apis.coap.tls.clientCertificateRequired | bool | `true` |  |
| coapgateway.apis.coap.tls.enabled | bool | `true` |  |
| coapgateway.apis.coap.tls.keyFile | string | `nil` |  |
| coapgateway.clients.authorizationServer.grpc.address | string | `""` |  |
| coapgateway.clients.authorizationServer.grpc.keepAlive.permitWithoutStream | bool | `true` |  |
| coapgateway.clients.authorizationServer.grpc.keepAlive.time | string | `"10s"` |  |
| coapgateway.clients.authorizationServer.grpc.keepAlive.timeout | string | `"20s"` |  |
| coapgateway.clients.authorizationServer.grpc.tls.caPool | string | `nil` |  |
| coapgateway.clients.authorizationServer.grpc.tls.certFile | string | `nil` |  |
| coapgateway.clients.authorizationServer.grpc.tls.keyFile | string | `nil` |  |
| coapgateway.clients.authorizationServer.grpc.tls.useSystemCAPool | bool | `false` |  |
| coapgateway.clients.authorizationServer.ownerClaim | string | `"sub"` |  |
| coapgateway.clients.eventBus.nats.pendingLimits.bytesLimit | string | `"67108864"` |  |
| coapgateway.clients.eventBus.nats.pendingLimits.msgLimit | string | `"524288"` |  |
| coapgateway.clients.eventBus.nats.tls.caPool | string | `nil` |  |
| coapgateway.clients.eventBus.nats.tls.certFile | string | `nil` |  |
| coapgateway.clients.eventBus.nats.tls.keyFile | string | `nil` |  |
| coapgateway.clients.eventBus.nats.tls.useSystemCAPool | bool | `false` |  |
| coapgateway.clients.eventBus.nats.url | string | `""` |  |
| coapgateway.clients.resourceAggregate.deviceStatusExpiration.enabled | bool | `false` |  |
| coapgateway.clients.resourceAggregate.deviceStatusExpiration.expiresIn | string | `"0s"` |  |
| coapgateway.clients.resourceAggregate.grpc.address | string | `""` |  |
| coapgateway.clients.resourceAggregate.grpc.keepAlive.permitWithoutStream | bool | `true` |  |
| coapgateway.clients.resourceAggregate.grpc.keepAlive.time | string | `"10s"` |  |
| coapgateway.clients.resourceAggregate.grpc.keepAlive.timeout | string | `"20s"` |  |
| coapgateway.clients.resourceAggregate.grpc.tls.caPool | string | `nil` |  |
| coapgateway.clients.resourceAggregate.grpc.tls.certFile | string | `nil` |  |
| coapgateway.clients.resourceAggregate.grpc.tls.keyFile | string | `nil` |  |
| coapgateway.clients.resourceAggregate.grpc.tls.useSystemCAPool | bool | `false` |  |
| coapgateway.clients.resourceDirectory.grpc.address | string | `""` |  |
| coapgateway.clients.resourceDirectory.grpc.keepAlive.permitWithoutStream | bool | `true` |  |
| coapgateway.clients.resourceDirectory.grpc.keepAlive.time | string | `"10s"` |  |
| coapgateway.clients.resourceDirectory.grpc.keepAlive.timeout | string | `"20s"` |  |
| coapgateway.clients.resourceDirectory.grpc.tls.caPool | string | `nil` |  |
| coapgateway.clients.resourceDirectory.grpc.tls.certFile | string | `nil` |  |
| coapgateway.clients.resourceDirectory.grpc.tls.keyFile | string | `nil` |  |
| coapgateway.clients.resourceDirectory.grpc.tls.useSystemCAPool | bool | `false` |  |
| coapgateway.cloudId | string | `nil` |  |
| coapgateway.config.fileName | string | `"service.yaml"` |  |
| coapgateway.config.mountPath | string | `"/config"` |  |
| coapgateway.config.volume | string | `"config"` |  |
| coapgateway.deploymentAnnotations | object | `{}` |  |
| coapgateway.deploymentLabels | object | `{}` |  |
| coapgateway.enabled | bool | `true` |  |
| coapgateway.extraVolumeMounts | object | `{}` |  |
| coapgateway.extraVolumes | object | `{}` |  |
| coapgateway.fullnameOverride | string | `nil` |  |
| coapgateway.image.imagePullSecrets | object | `{}` |  |
| coapgateway.image.pullPolicy | string | `"Always"` |  |
| coapgateway.image.registry | string | `nil` |  |
| coapgateway.image.repository | string | `"plgd/coap-gateway"` |  |
| coapgateway.image.tag | string | `nil` |  |
| coapgateway.imagePullSecrets | object | `{}` |  |
| coapgateway.initContainersTpl | object | `{}` |  |
| coapgateway.livenessProbe | object | `{}` |  |
| coapgateway.log.debug | bool | `false` |  |
| coapgateway.log.dumpCoapMessages | bool | `true` |  |
| coapgateway.name | string | `"coap-gateway"` |  |
| coapgateway.nodeSelector | object | `{}` |  |
| coapgateway.podAnnotations | object | `{}` |  |
| coapgateway.podLabels | object | `{}` |  |
| coapgateway.podSecurityContext | object | `{}` |  |
| coapgateway.port | int | `5684` |  |
| coapgateway.rbac.enabled | bool | `false` |  |
| coapgateway.rbac.roleBindingDefinitionTpl | string | `nil` |  |
| coapgateway.rbac.serviceAccountName | string | `"coap-gateway"` |  |
| coapgateway.readinessProbe | object | `{}` |  |
| coapgateway.replicas | int | `1` |  |
| coapgateway.resources | object | `{}` |  |
| coapgateway.restartPolicy | string | `"Always"` |  |
| coapgateway.securityContext | object | `{}` |  |
| coapgateway.service.annotations | object | `{}` |  |
| coapgateway.service.labels | object | `{}` |  |
| coapgateway.service.type | string | `"LoadBalancer"` |  |
| coapgateway.taskQueue.goPoolSize | int | `1600` |  |
| coapgateway.taskQueue.maxIdleTime | string | `"10m"` |  |
| coapgateway.taskQueue.size | string | `"2097152"` |  |
| coapgateway.tolerations | object | `{}` |  |
| extraDeploy | string | `nil` | Extra deploy. Resolved as template |
| global | object | `{"audience":null,"authority":null,"authorizationServer":{"oauth":{"clientID":null,"clientSecret":null,"scopes":[],"tokenURL":null}},"cloudId":null,"device":{"oauth":{"clientID":null,"clientSecret":null,"redirectURL":null,"scopes":[],"tokenURL":null}},"domain":null,"mongoUri":null,"natsUri":null}` | Global config variables |
| global.audience | string | `nil` | OAuth audience |
| global.authority | string | `nil` | OAuth authority |
| global.authorizationServer | object | `{"oauth":{"clientID":null,"clientSecret":null,"scopes":[],"tokenURL":null}}` | OAuth configuration for internal oAuth client |
| global.cloudId | string | `nil` | CloudID. Used by coap-gateway. It must be unique |
| global.device | object | `{"oauth":{"clientID":null,"clientSecret":null,"redirectURL":null,"scopes":[],"tokenURL":null}}` | OAuth configuration for internal oAuth device client |
| global.domain | string | `nil` | Global domain |
| global.mongoUri | string | `nil` | Override mongodb uri for every plgd-cloud services |
| global.natsUri | string | `nil` | Override nats uri for every plgd-cloud services |
| grpcgateway.affinity | object | `{}` | Affinity definition |
| grpcgateway.apis | object | `{"grpc":{"address":null,"authorization":{"audience":"","authority":"","http":{"idleConnTimeout":"30s","maxConnsPerHost":32,"maxIdleConns":16,"maxIdleConnsPerHost":16,"timeout":"10s","tls":{"caPool":null,"certFile":null,"keyFile":null,"useSystemCAPool":false}}},"enforcementPolicy":{"minTime":"5s","permitWithoutStream":true},"keepAlive":{"maxConnectionAge":"0s","maxConnectionAgeGrace":"0s","maxConnectionIdle":"0s","time":"2h","timeout":"20s"},"ownerCacheExpiration":"1m","tls":{"caPool":null,"certFile":null,"clientCertificateRequired":true,"keyFile":null}}}` | For complete grpc-gateway service configuration see [plgd/grpc-gateway](https://github.com/plgd-dev/cloud/tree/v2/grpc-gateway) |
| grpcgateway.clients | object | `{"authorizationServer":{"grpc":{"address":"","keepAlive":{"permitWithoutStream":true,"time":"10s","timeout":"20s"},"tls":{"caPool":null,"certFile":null,"keyFile":null,"useSystemCAPool":false}},"ownerClaim":"sub"},"eventBus":{"goPoolSize":16,"nats":{"pendingLimits":{"bytesLimit":67108864,"msgLimit":524288},"tls":{"caPool":null,"certFile":null,"keyFile":null,"useSystemCAPool":false},"url":null}},"resourceAggregate":{"grpc":{"address":"","keepAlive":{"permitWithoutStream":true,"time":"10s","timeout":"20s"},"tls":{"caPool":null,"certFile":null,"keyFile":null,"useSystemCAPool":false}}},"resourceDirectory":{"grpc":{"address":"","keepAlive":{"permitWithoutStream":true,"time":"10s","timeout":"20s"},"tls":{"caPool":null,"certFile":null,"keyFile":null,"useSystemCAPool":false}}}}` | For complete grpc-gateway service configuration see [plgd/grpc-gateway](https://github.com/plgd-dev/cloud/tree/v2/grpc-gateway) |
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
| httpgateway.apis | object | `{"http":{"address":null,"authorization":{"audience":"","authority":"","http":{"idleConnTimeout":"30s","maxConnsPerHost":32,"maxIdleConns":16,"maxIdleConnsPerHost":16,"timeout":"10s","tls":{"caPool":null,"certFile":null,"keyFile":null,"useSystemCAPool":false}}},"tls":{"caPool":null,"certFile":null,"clientCertificateRequired":true,"keyFile":null},"webSocket":{"pingFrequency":"10s","streamBodyLimit":262144}}}` | For complete http-gateway service configuration see [plgd/http-gateway](https://github.com/plgd-dev/cloud/tree/v2/http-gateway) |
| httpgateway.clients | object | `{"grpcGateway":{"grpc":{"address":"","keepAlive":{"permitWithoutStream":true,"time":"10s","timeout":"20s"},"tls":{"caPool":null,"certFile":null,"keyFile":null,"useSystemCAPool":false}}}}` | For complete http-gateway service configuration see [plgd/http-gateway](https://github.com/plgd-dev/cloud/tree/v2/http-gateway) |
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
| httpgateway.ui | object | `{"directory":"/usr/local/var/www","enabled":false,"oauthClient":{"audience":"","clientID":"","domain":"","httpGatewayAddress":"","scope":""}}` | For complete http-gateway service configuration see [plgd/http-gateway](https://github.com/plgd-dev/cloud/tree/v2/http-gateway) |
| mongodb | object | `{"arbiter":{"enabled":false},"architecture":"replicaset","auth":{"enabled":false},"customLivenessProbe":{"exec":{"command":["mongo","--disableImplicitSessions","--tls","--tlsCertificateKeyFile=/certs/cert.pem","--tlsCAFile=/certs/ca.pem","--eval","db.adminCommand('ping')"]},"failureThreshold":6,"initialDelaySeconds":30,"periodSeconds":10,"successThreshold":1,"timeoutSeconds":5},"customReadinessProbe":{"exec":{"command":["bash","-ec","TLS_OPTIONS='--tls --tlsCertificateKeyFile=/certs/cert.pem --tlsCAFile=/certs/ca.pem'\nmongo --disableImplicitSessions $TLS_OPTIONS --eval 'db.hello().isWritablePrimary || db.hello().secondary' | grep -q 'true'\n"]},"failureThreshold":6,"initialDelaySeconds":5,"periodSeconds":10,"successThreshold":1,"timeoutSeconds":5},"enabled":true,"extraEnvVars":[{"name":"MONGODB_EXTRA_FLAGS","value":"--tlsMode=requireTLS --tlsCertificateKeyFile=/certs/cert.pem --tlsCAFile=/certs/ca.pem"},{"name":"MONGODB_CLIENT_EXTRA_FLAGS","value":"--tls --tlsCertificateKeyFile=/certs/cert.pem --tlsCAFile=/certs/ca.pem"}],"extraVolumeMounts":[{"mountPath":"/certs","name":"mongodb-crt"}],"extraVolumes":[{"emptyDir":{},"name":"mongodb-crt"},{"name":"mongodb-cm-crt","secret":{"secretName":"mongodb-cm-crt"}}],"fullnameOverride":"mongodb","image":{"debug":true,"net":{"port":27017}},"initContainers":[{"command":["sh","-c","/bin/bash <<'EOF'\ncat /tmp/certs/tls.crt >> /certs/cert.pem\ncat /tmp/certs/tls.key >> /certs/cert.pem\ncp /tmp/certs/ca.crt  /certs/ca.pem\nEOF\n"],"image":"docker.io/bitnami/nginx:1.19.10-debian-10-r63","imagePullPolicy":"IfNotPresent","name":"convert-cm-crt","volumeMounts":[{"mountPath":"/certs","name":"mongodb-crt"},{"mountPath":"/tmp/certs","name":"mongodb-cm-crt"}]}],"livenessProbe":{"enabled":false},"persistence":{"enabled":true},"readinessProbe":{"enabled":false},"replicaCount":3,"replicaSetName":"rs0","tls":{"enabled":false}}` | External mongodb-replica dependency setup |
| nats | object | `{"cluster":{"enabled":false,"noAdvertise":false},"enabled":true,"leafnodes":{"enabled":false,"noAdvertise":false},"nats":{"tls":{"ca":"ca.crt","cert":"tls.crt","key":"tls.key","secret":{"name":"nats-service-crt"},"verify":true}},"natsbox":{"enabled":false}}` | External nats dependency setup |
| resourceaggregate.affinity | object | `{}` |  |
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
| resourceaggregate.clients.authorizationServer.cacheExpiration | string | `"1m"` |  |
| resourceaggregate.clients.authorizationServer.grpc.address | string | `""` |  |
| resourceaggregate.clients.authorizationServer.grpc.keepAlive.permitWithoutStream | bool | `true` |  |
| resourceaggregate.clients.authorizationServer.grpc.keepAlive.time | string | `"10s"` |  |
| resourceaggregate.clients.authorizationServer.grpc.keepAlive.timeout | string | `"20s"` |  |
| resourceaggregate.clients.authorizationServer.grpc.tls.caPool | string | `""` |  |
| resourceaggregate.clients.authorizationServer.grpc.tls.certFile | string | `""` |  |
| resourceaggregate.clients.authorizationServer.grpc.tls.keyFile | string | `""` |  |
| resourceaggregate.clients.authorizationServer.grpc.tls.useSystemCAPool | bool | `false` |  |
| resourceaggregate.clients.authorizationServer.oauth.audience | string | `""` |  |
| resourceaggregate.clients.authorizationServer.oauth.clientID | string | `""` |  |
| resourceaggregate.clients.authorizationServer.oauth.clientSecret | string | `""` |  |
| resourceaggregate.clients.authorizationServer.oauth.http.idleConnTimeout | string | `"30s"` |  |
| resourceaggregate.clients.authorizationServer.oauth.http.maxConnsPerHost | int | `32` |  |
| resourceaggregate.clients.authorizationServer.oauth.http.maxIdleConns | int | `16` |  |
| resourceaggregate.clients.authorizationServer.oauth.http.maxIdleConnsPerHost | int | `16` |  |
| resourceaggregate.clients.authorizationServer.oauth.http.timeout | string | `"10s"` |  |
| resourceaggregate.clients.authorizationServer.oauth.http.tls.caPool | string | `""` |  |
| resourceaggregate.clients.authorizationServer.oauth.http.tls.certFile | string | `""` |  |
| resourceaggregate.clients.authorizationServer.oauth.http.tls.keyFile | string | `""` |  |
| resourceaggregate.clients.authorizationServer.oauth.http.tls.useSystemCAPool | bool | `false` |  |
| resourceaggregate.clients.authorizationServer.oauth.scopes | list | `[]` |  |
| resourceaggregate.clients.authorizationServer.oauth.tokenURL | string | `""` |  |
| resourceaggregate.clients.authorizationServer.oauth.verifyServiceTokenFrequency | string | `"10s"` |  |
| resourceaggregate.clients.authorizationServer.ownerClaim | string | `"sub"` |  |
| resourceaggregate.clients.authorizationServer.pullFrequency | string | `"15s"` |  |
| resourceaggregate.clients.eventBus.nats.jetstream | bool | `false` |  |
| resourceaggregate.clients.eventBus.nats.pendingLimits.bytesLimit | int | `67108864` |  |
| resourceaggregate.clients.eventBus.nats.pendingLimits.msgLimit | int | `524288` |  |
| resourceaggregate.clients.eventBus.nats.tls.caPool | string | `""` |  |
| resourceaggregate.clients.eventBus.nats.tls.certFile | string | `""` |  |
| resourceaggregate.clients.eventBus.nats.tls.keyFile | string | `""` |  |
| resourceaggregate.clients.eventBus.nats.tls.useSystemCAPool | bool | `false` |  |
| resourceaggregate.clients.eventBus.nats.url | string | `""` |  |
| resourceaggregate.clients.eventStore.defaultCommandTTL | string | `"0s"` |  |
| resourceaggregate.clients.eventStore.mongoDB.batchSize | int | `128` |  |
| resourceaggregate.clients.eventStore.mongoDB.database | string | `"eventStore"` |  |
| resourceaggregate.clients.eventStore.mongoDB.maxConnIdleTime | string | `"4m0s"` |  |
| resourceaggregate.clients.eventStore.mongoDB.maxPoolSize | int | `16` |  |
| resourceaggregate.clients.eventStore.mongoDB.tls.caPool | string | `""` |  |
| resourceaggregate.clients.eventStore.mongoDB.tls.certFile | string | `""` |  |
| resourceaggregate.clients.eventStore.mongoDB.tls.keyFile | string | `""` |  |
| resourceaggregate.clients.eventStore.mongoDB.tls.useSystemCAPool | bool | `false` |  |
| resourceaggregate.clients.eventStore.mongoDB.uri | string | `nil` |  |
| resourceaggregate.clients.eventStore.occMaxRetry | int | `8` |  |
| resourceaggregate.clients.eventStore.snapshotThreshold | int | `16` |  |
| resourceaggregate.config.fileName | string | `"service.yaml"` |  |
| resourceaggregate.config.mountPath | string | `"/config"` |  |
| resourceaggregate.config.volume | string | `"config"` |  |
| resourceaggregate.deploymentAnnotations | object | `{}` |  |
| resourceaggregate.deploymentLabels | object | `{}` |  |
| resourceaggregate.enabled | bool | `true` |  |
| resourceaggregate.extraVolumeMounts | object | `{}` |  |
| resourceaggregate.extraVolumes | object | `{}` |  |
| resourceaggregate.fullnameOverride | string | `nil` |  |
| resourceaggregate.image.imagePullSecrets | object | `{}` |  |
| resourceaggregate.image.pullPolicy | string | `"Always"` |  |
| resourceaggregate.image.registry | string | `nil` |  |
| resourceaggregate.image.repository | string | `"plgd/resource-aggregate"` |  |
| resourceaggregate.image.tag | string | `nil` |  |
| resourceaggregate.imagePullSecrets | object | `{}` |  |
| resourceaggregate.initContainersTpl | object | `{}` |  |
| resourceaggregate.livenessProbe | object | `{}` |  |
| resourceaggregate.log.debug | bool | `false` |  |
| resourceaggregate.name | string | `"resource-aggregate"` |  |
| resourceaggregate.nodeSelector | object | `{}` |  |
| resourceaggregate.podAnnotations | object | `{}` |  |
| resourceaggregate.podLabels | object | `{}` |  |
| resourceaggregate.podSecurityContext | object | `{}` |  |
| resourceaggregate.port | int | `9100` |  |
| resourceaggregate.rbac.enabled | bool | `false` |  |
| resourceaggregate.rbac.roleBindingDefitionTpl | string | `nil` |  |
| resourceaggregate.rbac.serviceAccountName | string | `"resource-aggregate"` |  |
| resourceaggregate.readinessProbe | object | `{}` |  |
| resourceaggregate.replicas | int | `1` |  |
| resourceaggregate.resources | object | `{}` |  |
| resourceaggregate.restartPolicy | string | `"Always"` |  |
| resourceaggregate.securityContext | object | `{}` |  |
| resourceaggregate.service.annotations | object | `{}` |  |
| resourceaggregate.service.labels | object | `{}` |  |
| resourceaggregate.service.type | string | `"ClusterIP"` |  |
| resourceaggregate.tolerations | object | `{}` |  |
| resourcedirectory.affinity | object | `{}` |  |
| resourcedirectory.apis.grpc.address | string | `nil` |  |
| resourcedirectory.apis.grpc.authorization.audience | string | `""` |  |
| resourcedirectory.apis.grpc.authorization.authority | string | `""` |  |
| resourcedirectory.apis.grpc.authorization.http.idleConnTimeout | string | `"30s"` |  |
| resourcedirectory.apis.grpc.authorization.http.maxConnsPerHost | int | `32` |  |
| resourcedirectory.apis.grpc.authorization.http.maxIdleConns | int | `16` |  |
| resourcedirectory.apis.grpc.authorization.http.maxIdleConnsPerHost | int | `16` |  |
| resourcedirectory.apis.grpc.authorization.http.timeout | string | `"10s"` |  |
| resourcedirectory.apis.grpc.authorization.http.tls.useSystemCAPool | bool | `false` |  |
| resourcedirectory.apis.grpc.enforcementPolicy.minTime | string | `"5s"` |  |
| resourcedirectory.apis.grpc.enforcementPolicy.permitWithoutStream | bool | `true` |  |
| resourcedirectory.apis.grpc.keepAlive.maxConnectionAge | string | `"0s"` |  |
| resourcedirectory.apis.grpc.keepAlive.maxConnectionAgeGrace | string | `"0s"` |  |
| resourcedirectory.apis.grpc.keepAlive.maxConnectionIdle | string | `"0s"` |  |
| resourcedirectory.apis.grpc.keepAlive.time | string | `"2h"` |  |
| resourcedirectory.apis.grpc.keepAlive.timeout | string | `"20s"` |  |
| resourcedirectory.apis.grpc.tls.clientCertificateRequired | bool | `true` |  |
| resourcedirectory.clients.authorizationServer.cacheExpiration | string | `"1m"` |  |
| resourcedirectory.clients.authorizationServer.grpc.address | string | `""` |  |
| resourcedirectory.clients.authorizationServer.grpc.keepAlive.permitWithoutStream | bool | `true` |  |
| resourcedirectory.clients.authorizationServer.grpc.keepAlive.time | string | `"10s"` |  |
| resourcedirectory.clients.authorizationServer.grpc.keepAlive.timeout | string | `"20s"` |  |
| resourcedirectory.clients.authorizationServer.grpc.tls.caPool | string | `nil` |  |
| resourcedirectory.clients.authorizationServer.grpc.tls.certFile | string | `nil` |  |
| resourcedirectory.clients.authorizationServer.grpc.tls.keyFile | string | `nil` |  |
| resourcedirectory.clients.authorizationServer.grpc.tls.useSystemCAPool | bool | `false` |  |
| resourcedirectory.clients.authorizationServer.oauth.audience | string | `""` |  |
| resourcedirectory.clients.authorizationServer.oauth.clientID | string | `nil` |  |
| resourcedirectory.clients.authorizationServer.oauth.clientSecret | string | `nil` |  |
| resourcedirectory.clients.authorizationServer.oauth.http.idleConnTimeout | string | `"30s"` |  |
| resourcedirectory.clients.authorizationServer.oauth.http.maxConnsPerHost | int | `32` |  |
| resourcedirectory.clients.authorizationServer.oauth.http.maxIdleConns | int | `16` |  |
| resourcedirectory.clients.authorizationServer.oauth.http.maxIdleConnsPerHost | int | `16` |  |
| resourcedirectory.clients.authorizationServer.oauth.http.timeout | string | `"10s"` |  |
| resourcedirectory.clients.authorizationServer.oauth.http.tls.caPool | string | `nil` |  |
| resourcedirectory.clients.authorizationServer.oauth.http.tls.certFile | string | `nil` |  |
| resourcedirectory.clients.authorizationServer.oauth.http.tls.keyFile | string | `nil` |  |
| resourcedirectory.clients.authorizationServer.oauth.http.tls.useSystemCAPool | bool | `false` |  |
| resourcedirectory.clients.authorizationServer.oauth.scopes | list | `[]` |  |
| resourcedirectory.clients.authorizationServer.oauth.tokenURL | string | `""` |  |
| resourcedirectory.clients.authorizationServer.oauth.verifyServiceTokenFrequency | string | `"10s"` |  |
| resourcedirectory.clients.authorizationServer.ownerClaim | string | `"sub"` |  |
| resourcedirectory.clients.authorizationServer.pullFrequency | string | `"15s"` |  |
| resourcedirectory.clients.eventBus.goPoolSize | int | `16` |  |
| resourcedirectory.clients.eventBus.nats.jetstream | bool | `false` |  |
| resourcedirectory.clients.eventBus.nats.pendingLimits.bytesLimit | string | `"67108864"` |  |
| resourcedirectory.clients.eventBus.nats.pendingLimits.msgLimit | string | `"524288"` |  |
| resourcedirectory.clients.eventBus.nats.tls.caPool | string | `nil` |  |
| resourcedirectory.clients.eventBus.nats.tls.certFile | string | `nil` |  |
| resourcedirectory.clients.eventBus.nats.tls.keyFile | string | `nil` |  |
| resourcedirectory.clients.eventBus.nats.tls.useSystemCAPool | bool | `false` |  |
| resourcedirectory.clients.eventBus.nats.url | string | `""` |  |
| resourcedirectory.clients.eventStore.cacheExpiration | string | `"20m"` |  |
| resourcedirectory.clients.eventStore.mongoDB.batchSize | int | `128` |  |
| resourcedirectory.clients.eventStore.mongoDB.database | string | `"eventStore"` |  |
| resourcedirectory.clients.eventStore.mongoDB.maxConnIdleTime | string | `"4m0s"` |  |
| resourcedirectory.clients.eventStore.mongoDB.maxPoolSize | int | `16` |  |
| resourcedirectory.clients.eventStore.mongoDB.tls.caPool | string | `nil` |  |
| resourcedirectory.clients.eventStore.mongoDB.tls.certFile | string | `nil` |  |
| resourcedirectory.clients.eventStore.mongoDB.tls.keyFile | string | `nil` |  |
| resourcedirectory.clients.eventStore.mongoDB.tls.useSystemCAPool | bool | `false` |  |
| resourcedirectory.clients.eventStore.mongoDB.uri | string | `""` |  |
| resourcedirectory.config.fileName | string | `"service.yaml"` |  |
| resourcedirectory.config.mountPath | string | `"/config"` |  |
| resourcedirectory.config.volume | string | `"config"` |  |
| resourcedirectory.deploymentAnnotations | object | `{}` |  |
| resourcedirectory.deploymentLabels | object | `{}` |  |
| resourcedirectory.enabled | bool | `true` |  |
| resourcedirectory.extraVolumeMounts | object | `{}` |  |
| resourcedirectory.extraVolumes | object | `{}` |  |
| resourcedirectory.fullnameOverride | string | `nil` |  |
| resourcedirectory.image.imagePullSecrets | object | `{}` |  |
| resourcedirectory.image.pullPolicy | string | `"Always"` |  |
| resourcedirectory.image.registry | string | `nil` |  |
| resourcedirectory.image.repository | string | `"plgd/resource-directory"` |  |
| resourcedirectory.image.tag | string | `nil` |  |
| resourcedirectory.imagePullSecrets | object | `{}` |  |
| resourcedirectory.initContainersTpl | object | `{}` |  |
| resourcedirectory.livenessProbe | object | `{}` |  |
| resourcedirectory.log.debug | bool | `false` |  |
| resourcedirectory.name | string | `"resource-directory"` |  |
| resourcedirectory.nodeSelector | object | `{}` |  |
| resourcedirectory.podAnnotations | object | `{}` |  |
| resourcedirectory.podLabels | object | `{}` |  |
| resourcedirectory.podSecurityContext | object | `{}` |  |
| resourcedirectory.port | int | `9100` |  |
| resourcedirectory.publicConfiguration.authorizationURL | string | `nil` |  |
| resourcedirectory.publicConfiguration.caPool | string | `nil` |  |
| resourcedirectory.publicConfiguration.cloudAuthorizationProvider | string | `nil` |  |
| resourcedirectory.publicConfiguration.cloudID | string | `nil` |  |
| resourcedirectory.publicConfiguration.cloudURL | string | `nil` |  |
| resourcedirectory.publicConfiguration.deviceAuthorization.audience | string | `""` |  |
| resourcedirectory.publicConfiguration.deviceAuthorization.authority | string | `""` |  |
| resourcedirectory.publicConfiguration.deviceAuthorization.clientID | string | `""` |  |
| resourcedirectory.publicConfiguration.deviceAuthorization.http.idleConnTimeout | string | `"30s"` |  |
| resourcedirectory.publicConfiguration.deviceAuthorization.http.maxConnsPerHost | int | `32` |  |
| resourcedirectory.publicConfiguration.deviceAuthorization.http.maxIdleConns | int | `16` |  |
| resourcedirectory.publicConfiguration.deviceAuthorization.http.maxIdleConnsPerHost | int | `16` |  |
| resourcedirectory.publicConfiguration.deviceAuthorization.http.timeout | string | `"10s"` |  |
| resourcedirectory.publicConfiguration.deviceAuthorization.http.tls.caPool | string | `nil` |  |
| resourcedirectory.publicConfiguration.deviceAuthorization.http.tls.certFile | string | `nil` |  |
| resourcedirectory.publicConfiguration.deviceAuthorization.http.tls.keyFile | string | `nil` |  |
| resourcedirectory.publicConfiguration.deviceAuthorization.http.tls.useSystemCAPool | bool | `false` |  |
| resourcedirectory.publicConfiguration.deviceAuthorization.redirectURL | string | `""` |  |
| resourcedirectory.publicConfiguration.deviceAuthorization.scopes | list | `[]` |  |
| resourcedirectory.publicConfiguration.ownerClaim | string | `nil` |  |
| resourcedirectory.publicConfiguration.signingServerAddress | string | `nil` |  |
| resourcedirectory.publicConfiguration.tokenURL | string | `nil` |  |
| resourcedirectory.rbac.enabled | bool | `false` |  |
| resourcedirectory.rbac.roleBindingDefitionTpl | string | `nil` |  |
| resourcedirectory.rbac.serviceAccountName | string | `"resource-directory"` |  |
| resourcedirectory.readinessProbe | object | `{}` |  |
| resourcedirectory.replicas | int | `1` |  |
| resourcedirectory.resources | object | `{}` |  |
| resourcedirectory.restartPolicy | string | `"Always"` |  |
| resourcedirectory.securityContext | object | `{}` |  |
| resourcedirectory.service.annotations | object | `{}` |  |
| resourcedirectory.service.labels | object | `{}` |  |
| resourcedirectory.service.type | string | `"ClusterIP"` |  |
| resourcedirectory.tolerations | object | `{}` |  |

----------------------------------------------
Autogenerated from chart metadata using [helm-docs v1.5.0](https://github.com/norwoodj/helm-docs/releases/v1.5.0)
