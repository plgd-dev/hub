# Helm Chart for plgd hub

## Getting Started

More information are available in our [docs](https://plgd.dev/deployment/k8s/).

### Required variables

```yaml
# -- Global config variables
global:
  # -- Global domain
  domain:
  # -- HubID. Used by coap-gateway. It must be unique
  hubId:
  # -- OAuth owner Claim
  ownerClaim: "sub"
  # -- Optional
  #deviceIdClaim:
  # -- OAuth authority
  authority:
  # -- Optional OAuth audience
  #audience: ""
  # Global OAuth configuration used by multiple services
  oauth:
   # -- List of OAuth client's configurations
   device:
       # -- Name of provider
     - name:
       # -- Client ID
       clientID:
       # -- clientSecret or clientSecretFile
       clientSecret:
       #clientSecretFile:
       # -- Redirect URL. In case you are using mobile app, redirectURL should be in format cloud.plgd.mobile://login-callback
       redirectURL:
       # -- Use in httpgateway.ui.webConfiguration.deviceOAuthClient configuration. Default first item in list
       useInUi: true
   web:
    # -- ClientID used by Web UI
    clientID:
```

## Requirements

| Repository | Name | Version |
|------------|------|---------|
| <https://charts.bitnami.com/bitnami> | mongodb | 15.4.4 |
| <https://nats-io.github.io/k8s/helm/charts/> | nats | 1.1.9 |
| <https://scylla-operator-charts.storage.googleapis.com/stable> | scylla | 1.10.0 |

## Values

| Key | Type | Default | Description |
|-----|------|---------|-------------|
| certificateauthority.affinity | string | `nil` | Affinity definition |
| certificateauthority.apis | object | `{"grpc":{"address":null,"authorization":{"audience":null,"authority":null,"http":{"idleConnTimeout":"30s","maxConnsPerHost":32,"maxIdleConns":16,"maxIdleConnsPerHost":16,"timeout":"10s","tls":{"caPool":null,"certFile":null,"keyFile":null,"useSystemCAPool":true}},"ownerClaim":null},"enforcementPolicy":{"minTime":"5s","permitWithoutStream":true},"keepAlive":{"maxConnectionAge":"0s","maxConnectionAgeGrace":"0s","maxConnectionIdle":"0s","time":"2h","timeout":"20s"},"recvMsgSize":4194304,"sendMsgSize":4194304,"tls":{"caPool":null,"certFile":null,"clientCertificateRequired":false,"keyFile":null}},"http":{"address":null,"idleTimeout":"30s","readHeaderTimeout":"4s","readTimeout":"8s","writeTimeout":"16s"}}` | For complete certificate-authority service configuration see [plgd/certificate-authority](https://github.com/plgd-dev/hub/tree/main/certificate-authority) |
| certificateauthority.ca | object | `{"ca":"ca.crt","cert":"tls.crt","key":"tls.key","secret":{"name":null},"volume":{"mountPath":"/certs/coap-device-ca","name":"coap-device-ca"}}` | CA section |
| certificateauthority.ca.ca | string | `"ca.crt"` | CA file name in case of external CA |
| certificateauthority.ca.cert | string | `"tls.crt"` | Cert file name |
| certificateauthority.ca.key | string | `"tls.key"` | Cert key file name |
| certificateauthority.ca.secret.name | string | `nil` | Name of secret |
| certificateauthority.ca.volume.mountPath | string | `"/certs/coap-device-ca"` | CA certificate mount path |
| certificateauthority.ca.volume.name | string | `"coap-device-ca"` | CA certificate volume name |
| certificateauthority.clients.storage.cleanUpRecords | string | `"0 1 * * *"` | Remove any invalid entries in the cron format. If an empty string is provided, the cleanup function will be disabled. |
| certificateauthority.clients.storage.cqlDB.connectTimeout | string | `"10s"` |  |
| certificateauthority.clients.storage.cqlDB.hosts | list | `[]` |  |
| certificateauthority.clients.storage.cqlDB.keyspace.create | bool | `true` |  |
| certificateauthority.clients.storage.cqlDB.keyspace.name | string | `"plgdhub"` |  |
| certificateauthority.clients.storage.cqlDB.keyspace.replication.class | string | `"SimpleStrategy"` |  |
| certificateauthority.clients.storage.cqlDB.keyspace.replication.replication_factor | int | `1` |  |
| certificateauthority.clients.storage.cqlDB.numConnections | int | `16` |  |
| certificateauthority.clients.storage.cqlDB.port | int | `9142` |  |
| certificateauthority.clients.storage.cqlDB.reconnectionPolicy.constant.interval | string | `"3s"` |  |
| certificateauthority.clients.storage.cqlDB.reconnectionPolicy.constant.maxRetries | int | `3` |  |
| certificateauthority.clients.storage.cqlDB.table | string | `"signedCertificateRecords"` |  |
| certificateauthority.clients.storage.cqlDB.tls.caPool | string | `nil` |  |
| certificateauthority.clients.storage.cqlDB.tls.certFile | string | `nil` |  |
| certificateauthority.clients.storage.cqlDB.tls.keyFile | string | `nil` |  |
| certificateauthority.clients.storage.cqlDB.tls.useSystemCAPool | bool | `false` |  |
| certificateauthority.clients.storage.cqlDB.useHostnameResolution | bool | `true` | Resolve IP address to hostname before validate certificate. If false, the TLS validator will use ip/hostname advertised by the Cassandra node. |
| certificateauthority.clients.storage.mongoDB.bulkWrite.documentLimit | int | `1000` | The maximum number of documents to cache before an immediate write. |
| certificateauthority.clients.storage.mongoDB.bulkWrite.throttleTime | string | `"500ms"` | The amount of time to wait until a record is written to mongodb. Any records collected during the throttle time will also be written. A throttle time of zero writes immediately. If recordLimit is reached, all records are written immediately |
| certificateauthority.clients.storage.mongoDB.bulkWrite.timeout | string | `"1m0s"` | A time limit for write bulk to mongodb. A Timeout of zero means no timeout. |
| certificateauthority.clients.storage.mongoDB.database | string | `"certificateAuthorityService"` |  |
| certificateauthority.clients.storage.mongoDB.maxConnIdleTime | string | `"4m0s"` |  |
| certificateauthority.clients.storage.mongoDB.maxPoolSize | int | `16` |  |
| certificateauthority.clients.storage.mongoDB.tls.caPool | string | `nil` |  |
| certificateauthority.clients.storage.mongoDB.tls.certFile | string | `nil` |  |
| certificateauthority.clients.storage.mongoDB.tls.keyFile | string | `nil` |  |
| certificateauthority.clients.storage.mongoDB.tls.useSystemCAPool | bool | `false` |  |
| certificateauthority.clients.storage.mongoDB.uri | string | `nil` |  |
| certificateauthority.clients.storage.use | string | `"mongoDB"` |  |
| certificateauthority.config | object | `{"fileName":"service.yaml","mountPath":"/config","volume":"config"}` | Service configuration |
| certificateauthority.config.fileName | string | `"service.yaml"` | File name for config file |
| certificateauthority.config.mountPath | string | `"/config"` | Mount path |
| certificateauthority.config.volume | string | `"config"` | Config file volume name |
| certificateauthority.deploymentAnnotations | object | `{}` | Additional annotations for certificate-authority deployment |
| certificateauthority.deploymentLabels | object | `{}` | Additional labels for certificate-authority deployment |
| certificateauthority.domain | string | `nil` | External domain for certificate-authority. Default: api.{{ global.domain }} |
| certificateauthority.enabled | bool | `true` | Enable certificate-authority service |
| certificateauthority.extraContainers | object | `{}` | Extra POD containers |
| certificateauthority.extraVolumeMounts | string | `nil` | Optional extra volume mounts |
| certificateauthority.extraVolumes | string | `nil` | Optional extra volumes |
| certificateauthority.fullnameOverride | string | `nil` | Full name to override |
| certificateauthority.httpPort | int | `9101` |  |
| certificateauthority.hubId | string | `nil` | Hub ID. Overrides the global.hubId |
| certificateauthority.image.imagePullSecrets | string | `nil` | Image pull secrets |
| certificateauthority.image.pullPolicy | string | `"Always"` | Image pull policy |
| certificateauthority.image.registry | string | `"ghcr.io/"` | Image registry |
| certificateauthority.image.repository | string | `"plgd-dev/hub/certificate-authority"` | Image repository |
| certificateauthority.image.tag | string | `nil` | Image tag. |
| certificateauthority.imagePullSecrets | string | `nil` | Image pull secrets |
| certificateauthority.ingress.grpc.annotations | object | `{"cert-manager.io/private-key-rotation-policy":"always","ingress.kubernetes.io/force-ssl-redirect":"true","nginx.ingress.kubernetes.io/backend-protocol":"GRPCS","nginx.ingress.kubernetes.io/enable-cors":"true","nginx.org/grpc-services":"{{ include \"plgd-hub.certificateauthority.fullname\" . }}-grpc"}` | Pre defined map of Ingress annotation |
| certificateauthority.ingress.grpc.customAnnotations | object | `{}` | Custom map of Ingress annotation |
| certificateauthority.ingress.grpc.enabled | bool | `true` | Enable ingress |
| certificateauthority.ingress.grpc.paths | list | `["/certificateauthority.pb.CertificateAuthority"]` | Paths |
| certificateauthority.ingress.grpc.secretName | string | `nil` | Override name of host/tls secret. If not specified, it will be generated |
| certificateauthority.ingress.http.annotations | object | `{"cert-manager.io/private-key-rotation-policy":"always","ingress.kubernetes.io/force-ssl-redirect":"true","nginx.ingress.kubernetes.io/backend-protocol":"HTTPS","nginx.ingress.kubernetes.io/enable-cors":"true","nginx.org/grpc-services":"{{ include \"plgd-hub.certificateauthority.fullname\" . }}-http"}` | Pre defined map of Ingress annotation |
| certificateauthority.ingress.http.customAnnotations | object | `{}` | Custom map of Ingress annotation |
| certificateauthority.ingress.http.enabled | bool | `true` | Enable ingress |
| certificateauthority.ingress.http.paths | list | `["/api/v1/sign","/api/v1/signing"]` | Ingress path |
| certificateauthority.ingress.http.secretName | string | `nil` | Override name of host/tls secret. If not specified, it will be generated |
| certificateauthority.initContainersTpl | string | `nil` | Init containers definition |
| certificateauthority.livenessProbe | string | `nil` | Liveness probe. certificate-authority doesn't have any default liveness probe |
| certificateauthority.log | object | `{"dumpBody":false,"encoderConfig":{"timeEncoder":"rfc3339nano"},"encoding":"json","level":"info","stacktrace":{"enabled":false,"level":"warn"}}` | Log section |
| certificateauthority.log.dumpBody | bool | `false` | Dump grpc messages |
| certificateauthority.log.encoderConfig.timeEncoder | string | `"rfc3339nano"` | Time format for logs. The supported values are: "rfc3339nano", "rfc3339" |
| certificateauthority.log.encoding | string | `"json"` | The supported values are: "json", "console" |
| certificateauthority.log.level | string | `"info"` | Logging enabled from level |
| certificateauthority.log.stacktrace.enabled | bool | `false` | Log stacktrace |
| certificateauthority.log.stacktrace.level | string | `"warn"` | Stacktrace from level |
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
| certificateauthority.service.grpc.annotations | object | `{}` | Annotations for certificate-authority service |
| certificateauthority.service.grpc.crt.extraDnsNames | list | `[]` | Extra DNS names for service certificate |
| certificateauthority.service.grpc.labels | object | `{}` | Labels for certificate-authority service |
| certificateauthority.service.grpc.name | string | `"grpc"` | Name |
| certificateauthority.service.grpc.protocol | string | `"TCP"` | Protocol |
| certificateauthority.service.grpc.targetPort | string | `"grpc"` | Target port |
| certificateauthority.service.grpc.type | string | `"ClusterIP"` | Service type |
| certificateauthority.service.http.annotations | object | `{}` | Annotations for certificate-authority service |
| certificateauthority.service.http.crt.extraDnsNames | list | `[]` | Extra DNS names for service certificate |
| certificateauthority.service.http.labels | object | `{}` | Labels for certificate-authority service |
| certificateauthority.service.http.name | string | `"http"` | Name |
| certificateauthority.service.http.protocol | string | `"TCP"` | Protocol |
| certificateauthority.service.http.targetPort | string | `"http"` | Target port |
| certificateauthority.service.http.type | string | `"ClusterIP"` | Service type |
| certificateauthority.signer | object | `{"caPool":null,"certFile":null,"expiresIn":"87600h","keyFile":null,"validFrom":"now-1h"}` | For complete certificate-authority service configuration see [plgd/certificate-authority](https://github.com/plgd-dev/hub/tree/main/certificate-authority) |
| certificateauthority.tolerations | string | `nil` | Toleration definition |
| certmanager | object | `{"coap":{"cert":{"duration":null,"key":{"algorithm":null,"size":null},"renewBefore":null},"issuer":{"annotations":{},"group":null,"kind":null,"labels":{},"name":null,"spec":null}},"default":{"ca":{"commonName":"plgd-ca","enabled":true,"issuer":{"annotations":{},"enabled":true,"group":null,"kind":"Issuer","labels":{},"name":"ca-issuer","spec":{"selfSigned":{}}},"issuerRef":{"group":null,"kind":null,"name":null},"secret":{"name":"plgd-ca"}},"cert":{"annotations":{},"duration":"8760h0m0s","key":{"algorithm":"ECDSA","size":256},"labels":{},"renewBefore":"360h0m0s"},"issuer":{"annotations":{},"enabled":true,"group":"cert-manager.io","kind":"Issuer","labels":{},"name":"default-issuer","spec":{"selfSigned":{}}}},"enabled":true,"external":{"cert":{"duration":null,"key":{"algorithm":null,"size":null},"renewBefore":null},"issuer":{"annotations":{},"group":null,"kind":null,"labels":{},"name":null,"spec":null}},"internal":{"cert":{"duration":null,"key":{"algorithm":null,"size":null},"renewBefore":null},"issuer":{"annotations":{},"group":null,"kind":null,"labels":{},"name":null,"spec":null}},"storage":{"cert":{"duration":null,"key":{"algorithm":null,"size":null},"renewBefore":null},"issuer":{"annotations":{},"group":null,"kind":null,"labels":{},"name":null,"spec":null}}}` | Cert-manager integration section |
| certmanager.coap.cert.duration | string | `nil` | Certificate duration |
| certmanager.coap.cert.key.algorithm | string | `nil` | Certificate key algorithm |
| certmanager.coap.cert.key.size | string | `nil` | Certificate key size |
| certmanager.coap.cert.renewBefore | string | `nil` | Certificate renew before |
| certmanager.coap.issuer.annotations | object | `{}` | Annotations |
| certmanager.coap.issuer.group | string | `nil` | Group of coap issuer |
| certmanager.coap.issuer.kind | string | `nil` | Kind of coap issuer |
| certmanager.coap.issuer.labels | object | `{}` | Labels |
| certmanager.coap.issuer.name | string | `nil` | Name |
| certmanager.coap.issuer.spec | string | `nil` | cert-manager issuer spec |
| certmanager.default | object | `{"ca":{"commonName":"plgd-ca","enabled":true,"issuer":{"annotations":{},"enabled":true,"group":null,"kind":"Issuer","labels":{},"name":"ca-issuer","spec":{"selfSigned":{}}},"issuerRef":{"group":null,"kind":null,"name":null},"secret":{"name":"plgd-ca"}},"cert":{"annotations":{},"duration":"8760h0m0s","key":{"algorithm":"ECDSA","size":256},"labels":{},"renewBefore":"360h0m0s"},"issuer":{"annotations":{},"enabled":true,"group":"cert-manager.io","kind":"Issuer","labels":{},"name":"default-issuer","spec":{"selfSigned":{}}}}` | Default cert-manager section |
| certmanager.default.ca.commonName | string | `"plgd-ca"` | Common name for CA created as default issuer |
| certmanager.default.ca.issuer.annotations | object | `{}` | Annotation for root issuer |
| certmanager.default.ca.issuer.enabled | bool | `true` | Enable root issuer |
| certmanager.default.ca.issuer.group | string | `nil` | Group of root issuer |
| certmanager.default.ca.issuer.kind | string | `"Issuer"` | Kind of root issuer |
| certmanager.default.ca.issuer.labels | object | `{}` | Labels for root issuer |
| certmanager.default.ca.issuer.name | string | `"ca-issuer"` | Name of root issuer |
| certmanager.default.ca.issuer.spec | object | `{"selfSigned":{}}` | Default issuer specification. |
| certmanager.default.ca.issuerRef.group | string | `nil` | Group of issuer for sign CA |
| certmanager.default.ca.issuerRef.kind | string | `nil` | Kind of CA issuer |
| certmanager.default.ca.issuerRef.name | string | `nil` | Name of issuer for sign CA |
| certmanager.default.ca.secret.name | string | `"plgd-ca"` | Name of secret |
| certmanager.default.cert | object | `{"annotations":{},"duration":"8760h0m0s","key":{"algorithm":"ECDSA","size":256},"labels":{},"renewBefore":"360h0m0s"}` | Default certificate specification |
| certmanager.default.cert.annotations | object | `{}` | Certificate annotations |
| certmanager.default.cert.duration | string | `"8760h0m0s"` | Certificate duration |
| certmanager.default.cert.key | object | `{"algorithm":"ECDSA","size":256}` | Certificate key spec |
| certmanager.default.cert.key.algorithm | string | `"ECDSA"` | Algorithm |
| certmanager.default.cert.key.size | int | `256` | Key size |
| certmanager.default.cert.labels | object | `{}` | Certificate labels |
| certmanager.default.cert.renewBefore | string | `"360h0m0s"` | Certificate renew before |
| certmanager.default.issuer | object | `{"annotations":{},"enabled":true,"group":"cert-manager.io","kind":"Issuer","labels":{},"name":"default-issuer","spec":{"selfSigned":{}}}` | Default cert-manager issuer |
| certmanager.default.issuer.annotations | object | `{}` | Annotation for default issuer |
| certmanager.default.issuer.enabled | bool | `true` | Enable Default issuer |
| certmanager.default.issuer.group | string | `"cert-manager.io"` | Group of default issuer |
| certmanager.default.issuer.kind | string | `"Issuer"` | Kind of default issuer |
| certmanager.default.issuer.labels | object | `{}` | Labels for default issuer |
| certmanager.default.issuer.name | string | `"default-issuer"` | Name of default issuer |
| certmanager.default.issuer.spec | object | `{"selfSigned":{}}` | Default issuer specification. |
| certmanager.enabled | bool | `true` | Enable cert-manager integration |
| certmanager.external.cert.duration | string | `nil` | Certificate duration |
| certmanager.external.cert.key.algorithm | string | `nil` | Certificate key algorithm |
| certmanager.external.cert.key.size | string | `nil` | Certificate key size |
| certmanager.external.cert.renewBefore | string | `nil` | Certificate renew before |
| certmanager.external.issuer.annotations | object | `{}` | Annotations |
| certmanager.external.issuer.group | string | `nil` | Group of external issuer |
| certmanager.external.issuer.kind | string | `nil` | Kind of external issuer |
| certmanager.external.issuer.labels | object | `{}` | Labels |
| certmanager.external.issuer.name | string | `nil` | Name |
| certmanager.external.issuer.spec | string | `nil` | cert-manager issuer spec |
| certmanager.internal.cert.duration | string | `nil` | Certificate duration |
| certmanager.internal.cert.key.algorithm | string | `nil` | Certificate key algorithm |
| certmanager.internal.cert.key.size | string | `nil` | Certificate key size |
| certmanager.internal.cert.renewBefore | string | `nil` | Certificate renew before |
| certmanager.internal.issuer | object | `{"annotations":{},"group":null,"kind":null,"labels":{},"name":null,"spec":null}` | Internal issuer. In case you want to create your own issuer for internal certs |
| certmanager.internal.issuer.annotations | object | `{}` | Annotations |
| certmanager.internal.issuer.group | string | `nil` | Group of internal issuer |
| certmanager.internal.issuer.kind | string | `nil` | Kind of internal issuer |
| certmanager.internal.issuer.labels | object | `{}` | Labels |
| certmanager.internal.issuer.name | string | `nil` | Name |
| certmanager.internal.issuer.spec | string | `nil` | cert-manager issuer spec |
| certmanager.storage.cert.duration | string | `nil` | Certificate duration |
| certmanager.storage.cert.key.algorithm | string | `nil` | Certificate key algorithm |
| certmanager.storage.cert.key.size | string | `nil` | Certificate key size |
| certmanager.storage.cert.renewBefore | string | `nil` | Certificate renew before |
| certmanager.storage.issuer | object | `{"annotations":{},"group":null,"kind":null,"labels":{},"name":null,"spec":null}` | Storage issuer. In case you want to create your own issuer for storage certs (mongodb, scylla). In case if it is not set, the internal or default issuer will be used. |
| certmanager.storage.issuer.annotations | object | `{}` | Annotations |
| certmanager.storage.issuer.group | string | `nil` | Group of internal issuer |
| certmanager.storage.issuer.kind | string | `nil` | Kind of internal issuer |
| certmanager.storage.issuer.labels | object | `{}` | Labels |
| certmanager.storage.issuer.name | string | `nil` | Name |
| certmanager.storage.issuer.spec | string | `nil` | cert-manager issuer spec |
| cluster.dns | string | `"cluster.local"` | Cluster internal DNS prefix |
| coapgateway | object | `{"affinity":{},"apis":{"coap":{"authorization":{"deviceIdClaim":null,"ownerClaim":null,"providers":null},"blockwiseTransfer":{"blockSize":"1024","enabled":true},"externalAddress":"","keepAlive":{"timeout":"20s"},"maxMessageSize":262144,"messagePoolSize":1000,"messageQueueSize":16,"ownerCacheExpiration":"1m","protocols":["tcp"],"requireBatchObserveEnabled":true,"subscriptionBufferSize":1000,"tls":{"caPool":null,"certFile":null,"clientCertificateRequired":true,"disconnectOnExpiredCertificate":false,"enabled":true,"identityPropertiesRequired":true,"keyFile":null}}},"clients":{"certificateAuthority":{"grpc":{"address":"","keepAlive":{"permitWithoutStream":true,"time":"10s","timeout":"20s"},"recvMsgSize":4194304,"sendMsgSize":4194304,"tls":{"caPool":null,"certFile":null,"keyFile":null,"useSystemCAPool":false}}},"eventBus":{"nats":{"pendingLimits":{"bytesLimit":"67108864","msgLimit":"524288"},"tls":{"caPool":null,"certFile":null,"keyFile":null,"useSystemCAPool":false},"url":""}},"identityStore":{"grpc":{"address":"","keepAlive":{"permitWithoutStream":true,"time":"10s","timeout":"20s"},"recvMsgSize":4194304,"sendMsgSize":4194304,"tls":{"caPool":null,"certFile":null,"keyFile":null,"useSystemCAPool":false}},"ownerClaim":null},"resourceAggregate":{"grpc":{"address":"","keepAlive":{"permitWithoutStream":true,"time":"10s","timeout":"20s"},"recvMsgSize":4194304,"sendMsgSize":4194304,"tls":{"caPool":null,"certFile":null,"keyFile":null,"useSystemCAPool":false}}},"resourceDirectory":{"grpc":{"address":"","keepAlive":{"permitWithoutStream":true,"time":"10s","timeout":"20s"},"recvMsgSize":4194304,"sendMsgSize":4194304,"tls":{"caPool":null,"certFile":null,"keyFile":null,"useSystemCAPool":false}}}},"config":{"fileName":"service.yaml","mountPath":"/config","volume":"config"},"deploymentAnnotations":{},"deploymentLabels":{},"deviceTwin":{"maxETagsCountInRequest":8,"useETags":false},"enabled":true,"extraContainers":{},"extraVolumeMounts":{},"extraVolumes":{},"fullnameOverride":null,"hubId":null,"image":{"imagePullSecrets":{},"pullPolicy":"Always","registry":"ghcr.io/","repository":"plgd-dev/hub/coap-gateway","tag":null},"imagePullSecrets":{},"initContainersTpl":{},"livenessProbe":{},"log":{"dumpBody":false,"encoderConfig":{"timeEncoder":"rfc3339nano"},"encoding":"json","level":"info","stacktrace":{"enabled":false,"level":"warn"}},"name":"coap-gateway","nodeSelector":{},"podAnnotations":{},"podLabels":{},"podSecurityContext":{},"port":5684,"rbac":{"enabled":false,"roleBindingDefinitionTpl":null,"serviceAccountName":"coap-gateway"},"readinessProbe":{},"replicas":1,"resources":{},"restartPolicy":"Always","securityContext":{},"service":{"annotations":{},"labels":{},"nodePort":null,"tcp":{"annotations":{},"labels":{},"name":"coaps-tcp","nodePort":null,"protocol":"TCP","targetPort":"coaps-tcp","type":null},"type":"LoadBalancer","udp":{"annotations":{},"labels":{},"name":"coaps-udp","nodePort":null,"protocol":"UDP","targetPort":"coaps-udp","type":null}},"serviceHeartbeat":{"timeToLive":"1m"},"taskQueue":{"goPoolSize":1600,"maxIdleTime":"10m","size":"2097152"},"tolerations":{}}` | CoAP gateway parameters |
| coapgateway.affinity | object | `{}` | Affinity definition |
| coapgateway.apis | object | `{"coap":{"authorization":{"deviceIdClaim":null,"ownerClaim":null,"providers":null},"blockwiseTransfer":{"blockSize":"1024","enabled":true},"externalAddress":"","keepAlive":{"timeout":"20s"},"maxMessageSize":262144,"messagePoolSize":1000,"messageQueueSize":16,"ownerCacheExpiration":"1m","protocols":["tcp"],"requireBatchObserveEnabled":true,"subscriptionBufferSize":1000,"tls":{"caPool":null,"certFile":null,"clientCertificateRequired":true,"disconnectOnExpiredCertificate":false,"enabled":true,"identityPropertiesRequired":true,"keyFile":null}}}` | For complete coap-gateway service configuration see [plgd/coap-gateway](https://github.com/plgd-dev/hub/tree/main/coap-gateway) |
| coapgateway.apis.coap.tls.disconnectOnExpiredCertificate | bool | `false` | After the certificate expires, the connection will be disconnected |
| coapgateway.clients | object | `{"certificateAuthority":{"grpc":{"address":"","keepAlive":{"permitWithoutStream":true,"time":"10s","timeout":"20s"},"recvMsgSize":4194304,"sendMsgSize":4194304,"tls":{"caPool":null,"certFile":null,"keyFile":null,"useSystemCAPool":false}}},"eventBus":{"nats":{"pendingLimits":{"bytesLimit":"67108864","msgLimit":"524288"},"tls":{"caPool":null,"certFile":null,"keyFile":null,"useSystemCAPool":false},"url":""}},"identityStore":{"grpc":{"address":"","keepAlive":{"permitWithoutStream":true,"time":"10s","timeout":"20s"},"recvMsgSize":4194304,"sendMsgSize":4194304,"tls":{"caPool":null,"certFile":null,"keyFile":null,"useSystemCAPool":false}},"ownerClaim":null},"resourceAggregate":{"grpc":{"address":"","keepAlive":{"permitWithoutStream":true,"time":"10s","timeout":"20s"},"recvMsgSize":4194304,"sendMsgSize":4194304,"tls":{"caPool":null,"certFile":null,"keyFile":null,"useSystemCAPool":false}}},"resourceDirectory":{"grpc":{"address":"","keepAlive":{"permitWithoutStream":true,"time":"10s","timeout":"20s"},"recvMsgSize":4194304,"sendMsgSize":4194304,"tls":{"caPool":null,"certFile":null,"keyFile":null,"useSystemCAPool":false}}}}` | For complete coap-gateway service configuration see [plgd/coap-gateway](https://github.com/plgd-dev/hub/tree/main/coap-gateway) |
| coapgateway.config.fileName | string | `"service.yaml"` | Service configuration file name |
| coapgateway.config.mountPath | string | `"/config"` | Configuration mount path |
| coapgateway.config.volume | string | `"config"` | Volume name |
| coapgateway.deploymentAnnotations | object | `{}` | Additional annotations for coap-gateway deployment |
| coapgateway.deploymentLabels | object | `{}` | Additional labels for coap-gateway deployment |
| coapgateway.enabled | bool | `true` | Enable coap-gateway service |
| coapgateway.extraContainers | object | `{}` | Extra POD containers |
| coapgateway.extraVolumeMounts | object | `{}` | Optional extra volume mounts |
| coapgateway.extraVolumes | object | `{}` | Optional extra volumes |
| coapgateway.fullnameOverride | string | `nil` | Full name to override |
| coapgateway.hubId | string | `nil` | Hub ID. Overrides the global.hubId |
| coapgateway.image.imagePullSecrets | object | `{}` | Image pull secrets |
| coapgateway.image.pullPolicy | string | `"Always"` | Image pull policy |
| coapgateway.image.registry | string | `"ghcr.io/"` | Image registry |
| coapgateway.image.repository | string | `"plgd-dev/hub/coap-gateway"` | Image repository |
| coapgateway.image.tag | string | `nil` | Image tag |
| coapgateway.imagePullSecrets | object | `{}` | Image pull secrets |
| coapgateway.initContainersTpl | object | `{}` | Init containers definition |
| coapgateway.livenessProbe | object | `{}` | Liveness probe. coap-gateway doesn't have any default liveness probe |
| coapgateway.log | object | `{"dumpBody":false,"encoderConfig":{"timeEncoder":"rfc3339nano"},"encoding":"json","level":"info","stacktrace":{"enabled":false,"level":"warn"}}` | Log section |
| coapgateway.log.dumpBody | bool | `false` | Dump coap messages |
| coapgateway.log.encoderConfig.timeEncoder | string | `"rfc3339nano"` | Time format for logs. The supported values are: "rfc3339nano", "rfc3339" |
| coapgateway.log.encoding | string | `"json"` | The supported values are: "json", "console" |
| coapgateway.log.level | string | `"info"` | Logging enabled from level |
| coapgateway.log.stacktrace.enabled | bool | `false` | Log stacktrace |
| coapgateway.log.stacktrace.level | string | `"warn"` | Stacktrace from level |
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
| coapgateway.service.annotations | object | `{}` | Default annotations for coap-gateway services |
| coapgateway.service.labels | object | `{}` | Default labels for coap-gateway services |
| coapgateway.service.nodePort | string | `nil` | Use nodePort, if specified, for one of the protocols. If both protocols are enabled, nodePort needs to be configured directly in the service to mutually different ports. |
| coapgateway.service.tcp | object | `{"annotations":{},"labels":{},"name":"coaps-tcp","nodePort":null,"protocol":"TCP","targetPort":"coaps-tcp","type":null}` | TCP service |
| coapgateway.service.tcp.annotations | object | `{}` | Annotations for coap-gateway service |
| coapgateway.service.tcp.labels | object | `{}` | Labels for coap-gateway service |
| coapgateway.service.tcp.name | string | `"coaps-tcp"` | Name |
| coapgateway.service.tcp.nodePort | string | `nil` | Use nodePort if specified, must to be different as is in udp |
| coapgateway.service.tcp.protocol | string | `"TCP"` | Protocol |
| coapgateway.service.tcp.targetPort | string | `"coaps-tcp"` | Target port |
| coapgateway.service.tcp.type | string | `nil` | Service type |
| coapgateway.service.type | string | `"LoadBalancer"` | Service type |
| coapgateway.service.udp | object | `{"annotations":{},"labels":{},"name":"coaps-udp","nodePort":null,"protocol":"UDP","targetPort":"coaps-udp","type":null}` | UDP service |
| coapgateway.service.udp.annotations | object | `{}` | Annotations for coap-gateway service |
| coapgateway.service.udp.labels | object | `{}` | Labels for coap-gateway service |
| coapgateway.service.udp.name | string | `"coaps-udp"` | Name |
| coapgateway.service.udp.nodePort | string | `nil` | Use nodePort if specified. Must to be different as is in tcp |
| coapgateway.service.udp.protocol | string | `"UDP"` | Protocol |
| coapgateway.service.udp.targetPort | string | `"coaps-udp"` | Target port |
| coapgateway.service.udp.type | string | `nil` | Service type |
| coapgateway.serviceHeartbeat | object | `{"timeToLive":"1m"}` | service heartbeat section |
| coapgateway.serviceHeartbeat.timeToLive | string | `"1m"` | Specifies validity of the presence record created by the gateway. Must be greater than 1s. |
| coapgateway.taskQueue | object | `{"goPoolSize":1600,"maxIdleTime":"10m","size":"2097152"}` | For complete coap-gateway service configuration see [plgd/coap-gateway](https://github.com/plgd-dev/hub/tree/main/coap-gateway) |
| coapgateway.tolerations | object | `{}` | Toleration definition |
| extraCAPool | object | `{"authorization":{"configMapName":null,"enabled":"{{ include \"plgd-hub.extraCAPoolAuthorizationEnabled\" . }}","key":"{{ include \"plgd-hub.oldExtraCAPoolAuthorizationFileName\" . }}","mountPath":"/certs/extra/authorization","name":"authorization-ca-pool","secretName":"{{ include \"plgd-hub.oldExtraCAPoolAuthorizationSecretName\" . }}"},"coap":{"configMapName":null,"enabled":"{{ include \"plgd-hub.extraCAPoolCoapEnabled\" . }}","key":"ca.crt","mountPath":"/certs/extra/coap","name":"coap-ca-pool","secretName":"coap-ca-pool"},"internal":{"configMapName":null,"enabled":"{{ include \"plgd-hub.extraCAPoolInternalEnabled\" . }}","key":"ca.crt","mountPath":"/certs/extra/internal","name":"internal-ca-pool","secretName":"internal-ca-pool"},"storage":{"configMapName":null,"enabled":"{{ include \"plgd-hub.extraCAPoolStorageEnabled\" . }}","key":"ca.crt","mountPath":"/certs/extra/storage","name":"storage-ca-pool","secretName":"storage-ca-pool"}}` | Configuration parameters for extraCAPool used by services and clients |
| extraCAPool.authorization | object | `{"configMapName":null,"enabled":"{{ include \"plgd-hub.extraCAPoolAuthorizationEnabled\" . }}","key":"{{ include \"plgd-hub.oldExtraCAPoolAuthorizationFileName\" . }}","mountPath":"/certs/extra/authorization","name":"authorization-ca-pool","secretName":"{{ include \"plgd-hub.oldExtraCAPoolAuthorizationSecretName\" . }}"}` | Authorization CAPool section to verify the OAuth service certificate. |
| extraCAPool.authorization.enabled | string | `"{{ include \"plgd-hub.extraCAPoolAuthorizationEnabled\" . }}"` | Enable extra authorization ca pool |
| extraCAPool.authorization.mountPath | string | `"/certs/extra/authorization"` | Mount path for custom auth ca pool |
| extraCAPool.authorization.name | string | `"authorization-ca-pool"` | Volume and Mount name |
| extraCAPool.coap | object | `{"configMapName":null,"enabled":"{{ include \"plgd-hub.extraCAPoolCoapEnabled\" . }}","key":"ca.crt","mountPath":"/certs/extra/coap","name":"coap-ca-pool","secretName":"coap-ca-pool"}` | CoAP CAPool section to verify device certificate by coap-gateway |
| extraCAPool.coap.enabled | string | `"{{ include \"plgd-hub.extraCAPoolCoapEnabled\" . }}"` | Enable extra coap ca pool |
| extraCAPool.coap.mountPath | string | `"/certs/extra/coap"` | Mount path for custom coap ca pool |
| extraCAPool.coap.name | string | `"coap-ca-pool"` | Volume and Mount name |
| extraCAPool.internal | object | `{"configMapName":null,"enabled":"{{ include \"plgd-hub.extraCAPoolInternalEnabled\" . }}","key":"ca.crt","mountPath":"/certs/extra/internal","name":"internal-ca-pool","secretName":"internal-ca-pool"}` | Internal CAPool section to verify internal and storage services certificates by plgd services |
| extraCAPool.internal.enabled | string | `"{{ include \"plgd-hub.extraCAPoolInternalEnabled\" . }}"` | Enable extra internal ca pool |
| extraCAPool.internal.mountPath | string | `"/certs/extra/internal"` | Mount path for custom internal ca pool |
| extraCAPool.internal.name | string | `"internal-ca-pool"` | Volume and Mount name |
| extraCAPool.storage | object | `{"configMapName":null,"enabled":"{{ include \"plgd-hub.extraCAPoolStorageEnabled\" . }}","key":"ca.crt","mountPath":"/certs/extra/storage","name":"storage-ca-pool","secretName":"storage-ca-pool"}` | Storage CAPool section to verify internal and storage services certificates by storage services |
| extraCAPool.storage.enabled | string | `"{{ include \"plgd-hub.extraCAPoolStorageEnabled\" . }}"` | Enable extra storage ca pool |
| extraCAPool.storage.mountPath | string | `"/certs/extra/storage"` | Mount path for custom storage ca pool |
| extraCAPool.storage.name | string | `"storage-ca-pool"` | Volume and Mount name |
| extraDeploy | string | `nil` | Extra deploy. Resolved as template |
| global | object | `{"audience":"","authority":null,"authorization":{"audience":"{{ include \"plgd-hub.globalAudience\" . }}","endpoints":[{"authority":"{{ include \"plgd-hub.globalAuthority\" . }}","http":{"idleConnTimeout":"30s","maxConnsPerHost":32,"maxIdleConns":16,"maxIdleConnsPerHost":16,"timeout":"10s","tls":{"caPool":null,"certFile":null,"keyFile":null,"useSystemCAPool":true}}},{"authority":"{{ include \"plgd-hub.m2mOAuthServerAuthority\" . }}","http":{"idleConnTimeout":"30s","maxConnsPerHost":32,"maxIdleConns":16,"maxIdleConnsPerHost":16,"timeout":"10s","tls":{"caPool":null,"certFile":null,"keyFile":null,"useSystemCAPool":true}}}],"tokenTrustVerification":{"cacheExpiration":"30s"}},"defaultCommandTimeToLive":"10s","deviceIdClaim":null,"domain":null,"enableWildCartCert":true,"extraCAPool":{"authorization":"{{ include \"plgd-hub.oldGlobalAuthorizationCAPool\" . }}","coap":null,"internal":null,"storage":null},"hubId":null,"image":{"tag":null},"m2mOAuthServer":{"privateKey":""},"mongoUri":"","nats":{"leadResourceType":{"enabled":false,"filter":"","regexFilter":[],"useUUID":false}},"oauth":{"device":[],"web":{"clientID":null,"scopes":["openid"]}},"openTelemetryExporter":{"address":null,"enabled":false,"keepAlive":{"permitWithoutStream":true,"time":"10s","timeout":"20s"},"tls":{"caPool":null,"certFile":null,"keyFile":null,"useSystemCAPool":false}},"ownerClaim":"sub","standby":false,"useDatabase":"mongoDB"}` | Global config variables |
| global.audience | string | `""` | OAuth audience |
| global.authority | string | `nil` | OAuth authority |
| global.authorization | object | `{"audience":"{{ include \"plgd-hub.globalAudience\" . }}","endpoints":[{"authority":"{{ include \"plgd-hub.globalAuthority\" . }}","http":{"idleConnTimeout":"30s","maxConnsPerHost":32,"maxIdleConns":16,"maxIdleConnsPerHost":16,"timeout":"10s","tls":{"caPool":null,"certFile":null,"keyFile":null,"useSystemCAPool":true}}},{"authority":"{{ include \"plgd-hub.m2mOAuthServerAuthority\" . }}","http":{"idleConnTimeout":"30s","maxConnsPerHost":32,"maxIdleConns":16,"maxIdleConnsPerHost":16,"timeout":"10s","tls":{"caPool":null,"certFile":null,"keyFile":null,"useSystemCAPool":true}}}],"tokenTrustVerification":{"cacheExpiration":"30s"}}` | Default OAuth authorization for all services |
| global.authorization.endpoints[0] | object | `{"authority":"{{ include \"plgd-hub.globalAuthority\" . }}","http":{"idleConnTimeout":"30s","maxConnsPerHost":32,"maxIdleConns":16,"maxIdleConnsPerHost":16,"timeout":"10s","tls":{"caPool":null,"certFile":null,"keyFile":null,"useSystemCAPool":true}}}` | Authorization endpoint to Customer OAuth server |
| global.defaultCommandTimeToLive | string | `"10s"` | Global Default command time to live for resource-aggregate and resource-directory |
| global.deviceIdClaim | string | `nil` | Device ID claim |
| global.domain | string | `nil` | Global domain |
| global.enableWildCartCert | bool | `true` | Enable *.{{ global.domain }} for all external domain |
| global.extraCAPool | object | `{"authorization":"{{ include \"plgd-hub.oldGlobalAuthorizationCAPool\" . }}","coap":null,"internal":null,"storage":null}` | Custom CA certificates |
| global.extraCAPool.authorization | string | `"{{ include \"plgd-hub.oldGlobalAuthorizationCAPool\" . }}"` | Custom CA certificate for authorization endpoint in PEM format |
| global.extraCAPool.coap | string | `nil` | Custom CA certificate for coap endpoints in PEM format |
| global.extraCAPool.internal | string | `nil` | Custom CA certificate for internal endpoints in PEM format |
| global.extraCAPool.storage | string | `nil` | Custom CA certificate for storage(database) endpoints in PEM format |
| global.hubId | string | `nil` | hubId. Used by coapgateway, resourceaggregate, resourcedirectory, indentitystore, certificateauthority. It must be unique |
| global.image | object | `{"tag":null}` | Set image.tag for all services |
| global.m2mOAuthServer | object | `{"privateKey":""}` | M2M OAuth server |
| global.m2mOAuthServer.privateKey | string | `""` | private key to sign JWT m2m tokens |
| global.mongoUri | string | `""` | MongoDB URI |
| global.nats | object | `{"leadResourceType":{"enabled":false,"filter":"","regexFilter":[],"useUUID":false}}` | NATS publisher and subscriber configuration |
| global.nats.leadResourceType | object | `{"enabled":false,"filter":"","regexFilter":[],"useUUID":false}` | Lead resource type configuration |
| global.oauth | object | `{"device":[],"web":{"clientID":null,"scopes":["openid"]}}` | Global OAuth configuration used by multiple services |
| global.openTelemetryExporter | object | `{"address":null,"enabled":false,"keepAlive":{"permitWithoutStream":true,"time":"10s","timeout":"20s"},"tls":{"caPool":null,"certFile":null,"keyFile":null,"useSystemCAPool":false}}` | Global Open Telemetry exporter configuration |
| global.openTelemetryExporter.address | string | `nil` | The gRPC collector to which the exporter is going to send data |
| global.openTelemetryExporter.enabled | bool | `false` | Enable OTLP gRPC exporter |
| global.openTelemetryExporter.keepAlive | object | `{"permitWithoutStream":true,"time":"10s","timeout":"20s"}` | Expoter keep alive configuration |
| global.openTelemetryExporter.tls | object | `{"caPool":null,"certFile":null,"keyFile":null,"useSystemCAPool":false}` | Expoter TLS configuration |
| global.ownerClaim | string | `"sub"` | OAuth owner Claim |
| global.standby | bool | `false` | Sets cloud to standby mode |
| global.useDatabase | string | `"mongoDB"` | Use database. Supported values: "mongoDB", "cqlDB" |
| grpcgateway.affinity | object | `{}` | Affinity definition |
| grpcgateway.apis | object | `{"grpc":{"address":null,"authorization":{"audience":"","authority":"","http":{"idleConnTimeout":"30s","maxConnsPerHost":32,"maxIdleConns":16,"maxIdleConnsPerHost":16,"timeout":"10s","tls":{"caPool":null,"certFile":null,"keyFile":null,"useSystemCAPool":true}}},"enforcementPolicy":{"minTime":"5s","permitWithoutStream":true},"keepAlive":{"maxConnectionAge":"0s","maxConnectionAgeGrace":"0s","maxConnectionIdle":"0s","time":"2h","timeout":"20s"},"ownerCacheExpiration":"1m","recvMsgSize":4194304,"sendMsgSize":4194304,"subscriptionBufferSize":1000,"tls":{"caPool":null,"certFile":null,"clientCertificateRequired":false,"keyFile":null}}}` | For complete grpc-gateway service configuration see [plgd/grpc-gateway](https://github.com/plgd-dev/hub/tree/main/grpc-gateway) |
| grpcgateway.clients | object | `{"certificateAuthority":{"grpc":{"address":"","keepAlive":{"permitWithoutStream":true,"time":"10s","timeout":"20s"},"recvMsgSize":4194304,"sendMsgSize":4194304,"tls":{"caPool":null,"certFile":null,"keyFile":null,"useSystemCAPool":false}}},"eventBus":{"goPoolSize":16,"nats":{"pendingLimits":{"bytesLimit":"67108864","msgLimit":524288},"tls":{"caPool":null,"certFile":null,"keyFile":null,"useSystemCAPool":false},"url":null}},"identityStore":{"grpc":{"address":"","keepAlive":{"permitWithoutStream":true,"time":"10s","timeout":"20s"},"recvMsgSize":4194304,"sendMsgSize":4194304,"tls":{"caPool":null,"certFile":null,"keyFile":null,"useSystemCAPool":false}}},"resourceAggregate":{"grpc":{"address":"","keepAlive":{"permitWithoutStream":true,"time":"10s","timeout":"20s"},"recvMsgSize":4194304,"sendMsgSize":4194304,"tls":{"caPool":null,"certFile":null,"keyFile":null,"useSystemCAPool":false}}},"resourceDirectory":{"grpc":{"address":"","keepAlive":{"permitWithoutStream":true,"time":"10s","timeout":"20s"},"recvMsgSize":4194304,"sendMsgSize":4194304,"tls":{"caPool":null,"certFile":null,"keyFile":null,"useSystemCAPool":false}}}}` | For complete grpc-gateway service configuration see [plgd/grpc-gateway](https://github.com/plgd-dev/hub/tree/main/grpc-gateway) |
| grpcgateway.config | object | `{"fileName":"service.yaml","mountPath":"/config","volume":"config"}` | Service yaml configuration section |
| grpcgateway.config.fileName | string | `"service.yaml"` | Service configuration file name |
| grpcgateway.config.mountPath | string | `"/config"` | Service configuration mount path |
| grpcgateway.config.volume | string | `"config"` | Service configuration volume name |
| grpcgateway.deploymentAnnotations | object | `{}` | Additional annotations for grpc-gateway deployment |
| grpcgateway.deploymentLabels | object | `{}` | Additional labels for grpc-gateway deployment |
| grpcgateway.domain | string | `nil` | External domain for grpc-gateway. Default: api.{{ global.domain }} |
| grpcgateway.enabled | bool | `true` | Enable grpc-gateway service |
| grpcgateway.extraContainers | object | `{}` | Extra POD containers |
| grpcgateway.extraVolumeMounts | object | `{}` | Optional extra volume mounts |
| grpcgateway.extraVolumes | object | `{}` | Optional extra volumes |
| grpcgateway.fullnameOverride | string | `nil` | Full name to override |
| grpcgateway.image.imagePullSecrets | object | `{}` | Image pull secrets |
| grpcgateway.image.pullPolicy | string | `"Always"` | Image pull policy |
| grpcgateway.image.registry | string | `"ghcr.io/"` | Image registry |
| grpcgateway.image.repository | string | `"plgd-dev/hub/grpc-gateway"` | Image repository |
| grpcgateway.image.tag | string | `nil` | Image tag. |
| grpcgateway.imagePullSecrets | object | `{}` | Image pull secrets |
| grpcgateway.ingress.annotations | object | `{"cert-manager.io/private-key-rotation-policy":"always","ingress.kubernetes.io/force-ssl-redirect":"true","nginx.ingress.kubernetes.io/backend-protocol":"GRPCS","nginx.ingress.kubernetes.io/enable-cors":"true","nginx.org/grpc-services":"{{ include \"plgd-hub.grpcgateway.fullname\" . }}"}` | Ingress annotations |
| grpcgateway.ingress.customAnnotations | object | `{}` | Custom map of Ingress annotation |
| grpcgateway.ingress.enabled | bool | `true` | Enable ingress |
| grpcgateway.ingress.paths[0] | string | `"/grpcgateway.pb.GrpcGateway"` |  |
| grpcgateway.ingress.secretName | string | `nil` | Override name of host/tls secret. If not specified, it will be generated |
| grpcgateway.initContainersTpl | object | `{}` | Init containers definition |
| grpcgateway.livenessProbe | object | `{}` | Liveness probe. grpc-gateway doesn't have any default liveness probe |
| grpcgateway.log.dumpBody | bool | `false` | Dump grpc messages |
| grpcgateway.log.encoderConfig.timeEncoder | string | `"rfc3339nano"` | Time format for logs. The supported values are: "rfc3339nano", "rfc3339" |
| grpcgateway.log.encoding | string | `"json"` | The supported values are: "json", "console" |
| grpcgateway.log.level | string | `"info"` | Logging enabled from level |
| grpcgateway.log.stacktrace.enabled | bool | `false` | Log stacktrace |
| grpcgateway.log.stacktrace.level | string | `"warn"` | Stacktrace from level |
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
| grpcgateway.service.crt.extraDnsNames | list | `[]` | Extra DNS names for service certificate |
| grpcgateway.service.labels | object | `{}` | Labels for grpc-gateway service |
| grpcgateway.service.name | string | `"grpc"` | Name |
| grpcgateway.service.protocol | string | `"TCP"` | Protocol |
| grpcgateway.service.targetPort | string | `"grpc"` | Target port |
| grpcgateway.service.type | string | `"ClusterIP"` | Service type |
| grpcgateway.tolerations | object | `{}` | Toleration definition |
| grpcreflection.affinity | object | `{}` | Affinity definition |
| grpcreflection.apis | object | `{"grpc":{"address":null,"enforcementPolicy":{"minTime":"5s","permitWithoutStream":true},"keepAlive":{"maxConnectionAge":"0s","maxConnectionAgeGrace":"0s","maxConnectionIdle":"0s","time":"2h","timeout":"20s"},"ownerCacheExpiration":"1m","recvMsgSize":4194304,"sendMsgSize":4194304,"subscriptionBufferSize":1000,"tls":{"caPool":null,"certFile":null,"clientCertificateRequired":false,"keyFile":null}}}` | For complete grpc-reflection service configuration see [plgd/grpc-reflection](https://github.com/plgd-dev/hub/tree/main/grpc-reflection) |
| grpcreflection.config | object | `{"fileName":"service.yaml","mountPath":"/config","volume":"config"}` | Service yaml configuration section |
| grpcreflection.config.fileName | string | `"service.yaml"` | Service configuration file name |
| grpcreflection.config.mountPath | string | `"/config"` | Service configuration mount path |
| grpcreflection.config.volume | string | `"config"` | Service configuration volume name |
| grpcreflection.deploymentAnnotations | object | `{}` | Additional annotations for grpc-reflection deployment |
| grpcreflection.deploymentLabels | object | `{}` | Additional labels for grpc-reflection deployment |
| grpcreflection.enabled | bool | `true` | Enable grpc-reflection service |
| grpcreflection.extraContainers | object | `{}` | Extra POD containers |
| grpcreflection.extraVolumeMounts | object | `{}` | Optional extra volume mounts |
| grpcreflection.extraVolumes | object | `{}` | Optional extra volumes |
| grpcreflection.fullnameOverride | string | `nil` | Full name to override |
| grpcreflection.image.imagePullSecrets | object | `{}` | Image pull secrets |
| grpcreflection.image.pullPolicy | string | `"Always"` | Image pull policy |
| grpcreflection.image.registry | string | `"ghcr.io/"` | Image registry |
| grpcreflection.image.repository | string | `"plgd-dev/hub/grpc-reflection"` | Image repository |
| grpcreflection.image.tag | string | `nil` | Image tag. |
| grpcreflection.imagePullSecrets | object | `{}` | Image pull secrets |
| grpcreflection.ingress.annotations | object | `{"cert-manager.io/private-key-rotation-policy":"always","ingress.kubernetes.io/force-ssl-redirect":"true","nginx.ingress.kubernetes.io/backend-protocol":"GRPCS","nginx.ingress.kubernetes.io/enable-cors":"true","nginx.org/grpc-services":"{{ include \"plgd-hub.grpcreflection.fullname\" . }}"}` | Ingress annotations |
| grpcreflection.ingress.customAnnotations | object | `{}` | Custom map of Ingress annotation |
| grpcreflection.ingress.enabled | bool | `true` | Enable ingress |
| grpcreflection.ingress.paths[0] | string | `"/grpc.reflection.v1alpha.ServerReflection"` |  |
| grpcreflection.ingress.paths[1] | string | `"/grpc.reflection.v1.ServerReflection"` |  |
| grpcreflection.ingress.secretName | string | `nil` | Override name of host/tls secret. If not specified, it will be generated |
| grpcreflection.initContainersTpl | object | `{}` | Init containers definition |
| grpcreflection.livenessProbe | object | `{}` | Liveness probe. grpc-reflection doesn't have any default liveness probe |
| grpcreflection.log.dumpBody | bool | `false` | Dump grpc messages |
| grpcreflection.log.encoderConfig.timeEncoder | string | `"rfc3339nano"` | Time format for logs. The supported values are: "rfc3339nano", "rfc3339" |
| grpcreflection.log.encoding | string | `"json"` | The supported values are: "json", "console" |
| grpcreflection.log.level | string | `"info"` | Logging enabled from level |
| grpcreflection.log.stacktrace.enabled | bool | `false` | Log stacktrace |
| grpcreflection.log.stacktrace.level | string | `"warn"` | Stacktrace from level |
| grpcreflection.name | string | `"grpc-reflection"` | Name of component. Used in label selectors |
| grpcreflection.nodeSelector | object | `{}` | Node selector |
| grpcreflection.podAnnotations | object | `{}` | Annotations for grpc-reflection pod |
| grpcreflection.podLabels | object | `{}` | Labels for grpc-reflection pod |
| grpcreflection.podSecurityContext | object | `{}` | Pod security context |
| grpcreflection.port | int | `9100` | Service and POD port |
| grpcreflection.rbac | object | `{"enabled":false,"roleBindingDefitionTpl":null,"serviceAccountName":"grpc-reflection"}` | RBAC configuration |
| grpcreflection.rbac.roleBindingDefitionTpl | string | `nil` | Template definition for Role/binding etc.. |
| grpcreflection.rbac.serviceAccountName | string | `"grpc-reflection"` | Name of grpc-reflection SA |
| grpcreflection.readinessProbe | object | `{}` | Readiness probe. grpc-reflection doesn't have aby default readiness probe |
| grpcreflection.replicas | int | `1` | Number of replicas |
| grpcreflection.resources | object | `{}` | Resources limit |
| grpcreflection.restartPolicy | string | `"Always"` | Restart policy for pod |
| grpcreflection.securityContext | object | `{}` | Security context for pod |
| grpcreflection.service.annotations | object | `{}` | Annotations for grpc-reflection service |
| grpcreflection.service.crt.extraDnsNames | list | `[]` | Extra DNS names for service certificate |
| grpcreflection.service.labels | object | `{}` | Labels for grpc-reflection service |
| grpcreflection.service.name | string | `"grpc"` | Name |
| grpcreflection.service.protocol | string | `"TCP"` | Protocol |
| grpcreflection.service.targetPort | string | `"grpc"` | Target port |
| grpcreflection.service.type | string | `"ClusterIP"` | Service type |
| grpcreflection.tolerations | object | `{}` | Toleration definition |
| httpgateway.affinity | object | `{}` | Affinity definition |
| httpgateway.apiDomain | string | `nil` | Domain for http-gateway API. Default: api.{{ global.domain }} |
| httpgateway.apis | object | `{"http":{"address":null,"authorization":{"audience":null,"authority":null,"http":{"idleConnTimeout":"30s","maxConnsPerHost":32,"maxIdleConns":16,"maxIdleConnsPerHost":16,"timeout":"10s","tls":{"caPool":null,"certFile":null,"keyFile":null,"useSystemCAPool":true}}},"idleTimeout":"30s","readHeaderTimeout":"4s","readTimeout":"8s","tls":{"caPool":null,"certFile":null,"clientCertificateRequired":false,"keyFile":null},"webSocket":{"pingFrequency":"10s","streamBodyLimit":262144},"writeTimeout":"16s"}}` | For complete http-gateway service configuration see [plgd/http-gateway](https://github.com/plgd-dev/hub/tree/main/http-gateway) |
| httpgateway.clients | object | `{"grpcGateway":{"grpc":{"address":"","keepAlive":{"permitWithoutStream":true,"time":"10s","timeout":"20s"},"recvMsgSize":4194304,"sendMsgSize":4194304,"tls":{"caPool":null,"certFile":null,"keyFile":null,"useSystemCAPool":false}}}}` | For complete http-gateway service configuration see [plgd/http-gateway](https://github.com/plgd-dev/hub/tree/main/http-gateway) |
| httpgateway.config | object | `{"fileName":"service.yaml","mountPath":"/config","volume":"config"}` | Http-gateway service yaml config section |
| httpgateway.config.fileName | string | `"service.yaml"` | Name of configuration file |
| httpgateway.config.mountPath | string | `"/config"` | Mount path |
| httpgateway.config.volume | string | `"config"` | Volume for configuration file |
| httpgateway.deploymentAnnotations | object | `{}` | Additional annotations for http-gateway deployment |
| httpgateway.deploymentLabels | object | `{}` | Additional labels for http-gateway deployment |
| httpgateway.enabled | bool | `true` | Enable http-gateway service |
| httpgateway.extraContainers | object | `{}` | Extra POD containers |
| httpgateway.extraVolumeMounts | object | `{}` | Optional extra volume mounts |
| httpgateway.extraVolumes | object | `{}` | Optional extra volumes |
| httpgateway.fullnameOverride | string | `nil` | Full name to override |
| httpgateway.image.imagePullSecrets | object | `{}` | Image pull secrets |
| httpgateway.image.pullPolicy | string | `"Always"` | Image pull policy |
| httpgateway.image.registry | string | `"ghcr.io/"` | Image registry |
| httpgateway.image.repository | string | `"plgd-dev/hub/http-gateway"` | Image repository |
| httpgateway.image.tag | string | `nil` | Image tag. |
| httpgateway.imagePullSecrets | object | `{}` | Image pull secrets |
| httpgateway.ingress.api | object | `{"annotations":{"cert-manager.io/private-key-rotation-policy":"always","ingress.kubernetes.io/force-ssl-redirect":"true","nginx.ingress.kubernetes.io/backend-protocol":"HTTPS","nginx.ingress.kubernetes.io/enable-cors":"true","nginx.org/grpc-services":"{{ include \"plgd-hub.httpgateway.fullname\" . }}"},"customAnnotations":{},"enabled":true,"paths":["/api","/.well-known/hub-configuration","/.well-known/configuration"],"secretName":null}` | API ingress |
| httpgateway.ingress.api.annotations | object | `{"cert-manager.io/private-key-rotation-policy":"always","ingress.kubernetes.io/force-ssl-redirect":"true","nginx.ingress.kubernetes.io/backend-protocol":"HTTPS","nginx.ingress.kubernetes.io/enable-cors":"true","nginx.org/grpc-services":"{{ include \"plgd-hub.httpgateway.fullname\" . }}"}` | Pre defined map of Ingress annotation |
| httpgateway.ingress.api.customAnnotations | object | `{}` | Custom map of Ingress annotation |
| httpgateway.ingress.api.enabled | bool | `true` | Enable ingress |
| httpgateway.ingress.api.paths | list | `["/api","/.well-known/hub-configuration","/.well-known/configuration"]` | Ingress path |
| httpgateway.ingress.api.secretName | string | `nil` | Override name of host/tls secret. If not specified, it will be generated |
| httpgateway.ingress.ui | object | `{"annotations":{"cert-manager.io/private-key-rotation-policy":"always","ingress.kubernetes.io/force-ssl-redirect":"true","nginx.ingress.kubernetes.io/backend-protocol":"HTTPS","nginx.ingress.kubernetes.io/enable-cors":"true"},"customAnnotations":{},"enabled":true,"paths":["/"],"secretName":null}` | UI ingress |
| httpgateway.ingress.ui.annotations | object | `{"cert-manager.io/private-key-rotation-policy":"always","ingress.kubernetes.io/force-ssl-redirect":"true","nginx.ingress.kubernetes.io/backend-protocol":"HTTPS","nginx.ingress.kubernetes.io/enable-cors":"true"}` | Pre defined map of Ingress annotation |
| httpgateway.ingress.ui.customAnnotations | object | `{}` | Custom map of Ingress annotation |
| httpgateway.ingress.ui.enabled | bool | `true` | Enable ingress |
| httpgateway.ingress.ui.paths | list | `["/"]` | Ingress path |
| httpgateway.ingress.ui.secretName | string | `nil` | Override name of host/tls secret. If not specified, it will be generated |
| httpgateway.initContainersTpl | object | `{}` | Init containers definition. Render as template |
| httpgateway.livenessProbe | object | `{}` | Liveness probe. http-gateway doesn't have any default liveness probe |
| httpgateway.log.encoderConfig.timeEncoder | string | `"rfc3339nano"` | Time format for logs. The supported values are: "rfc3339nano", "rfc3339" |
| httpgateway.log.encoding | string | `"json"` | The supported values are: "json", "console" |
| httpgateway.log.level | string | `"info"` | Logging enabled from level |
| httpgateway.log.stacktrace.enabled | bool | `false` | Log stacktrace |
| httpgateway.log.stacktrace.level | string | `"warn"` | Stacktrace from level |
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
| httpgateway.service.crt.extraDnsNames | list | `[]` | Extra DNS names for service certificate |
| httpgateway.service.labels | object | `{}` | Labels for http-gateway service |
| httpgateway.service.name | string | `"http"` | Name |
| httpgateway.service.protocol | string | `"TCP"` | Protocol |
| httpgateway.service.targetPort | string | `"http"` | Target port |
| httpgateway.service.type | string | `"ClusterIP"` |  |
| httpgateway.tolerations | object | `{}` | Toleration definition |
| httpgateway.ui | object | `{"directory":"/usr/local/var/www","enabled":true,"theme":"","webConfiguration":{"deviceOAuthClient":{"audience":null,"authority":"","clientID":null,"providerName":null,"scopes":[]},"deviceProvisioningService":"","httpGatewayAddress":"","m2mOAuthClient":{"audience":null,"authority":"","clientAssertionType":null,"clientID":null,"grantType":null,"scopes":[]},"snippetService":"","visibility":{"mainSidebar":{"apiTokens":false,"certificates":true,"chatRoom":true,"configuration":true,"dashboard":false,"deviceFirmwareUpdate":false,"deviceLogs":false,"deviceProvisioning":true,"devices":true,"docs":true,"integrations":false,"pendingCommands":true,"remoteClients":true,"schemaHub":false,"snippetService":true}},"webOAuthClient":{"audience":"","authority":"","clientID":"","scopes":[]}}}` | For complete http-gateway service configuration see [plgd/http-gateway](https://github.com/plgd-dev/hub/tree/main/http-gateway) |
| httpgateway.uiDomain | string | `nil` | Domain for UI Default: {{ global.domain }} |
| identitystore.affinity | object | `{}` | Affinity definition |
| identitystore.apis | object | `{"grpc":{"address":null,"authorization":{"audience":null,"authority":null,"http":{"idleConnTimeout":"30s","maxConnsPerHost":32,"maxIdleConns":16,"maxIdleConnsPerHost":16,"timeout":"10s","tls":{"caPool":null,"certFile":null,"keyFile":null,"useSystemCAPool":true}},"ownerClaim":null},"enforcementPolicy":{"minTime":"5s","permitWithoutStream":true},"keepAlive":{"maxConnectionAge":"0s","maxConnectionAgeGrace":"0s","maxConnectionIdle":"0s","time":"2h","timeout":"20s"},"recvMsgSize":4194304,"sendMsgSize":4194304,"tls":{"caPool":null,"certFile":null,"clientCertificateRequired":true,"keyFile":null}}}` | For complete identity service configuration see [plgd/identity](https://github.com/plgd-dev/hub/tree/main/identity) |
| identitystore.clients | object | `{"eventBus":{"nats":{"flusherTimeout":"30s","jetstream":false,"tls":{"useSystemCAPool":false},"url":""}},"storage":{"cqlDB":{"connectTimeout":"10s","hosts":[],"keyspace":{"create":true,"name":"plgdhub","replication":{"class":"SimpleStrategy","replication_factor":1}},"numConnections":16,"port":9142,"reconnectionPolicy":{"constant":{"interval":"3s","maxRetries":3}},"table":"deviceOwners","tls":{"caPool":null,"certFile":null,"keyFile":null,"useSystemCAPool":false},"useHostnameResolution":true},"mongoDB":{"database":"ownersDevices","maxConnIdleTime":"4m0s","maxPoolSize":16,"tls":{"caPool":null,"certFile":null,"keyFile":null,"useSystemCAPool":false},"uri":null},"use":"mongoDB"}}` | For complete identity service configuration see [plgd/authorization](https://github.com/plgd-dev/hub/tree/main/identity) |
| identitystore.clients.storage.cqlDB.useHostnameResolution | bool | `true` | Resolve IP address to hostname before validate certificate. If false, the TLS validator will use ip/hostname advertised by the Cassandra node. |
| identitystore.config | object | `{"fileName":"service.yaml","mountPath":"/config","volume":"config"}` | yaml configuration |
| identitystore.config.fileName | string | `"service.yaml"` | File name |
| identitystore.config.mountPath | string | `"/config"` | Service configuration mount path |
| identitystore.config.volume | string | `"config"` | Volume name |
| identitystore.deploymentAnnotations | object | `{}` | Additional annotations for identity deployment |
| identitystore.deploymentLabels | object | `{}` | Additional labels for identity deployment |
| identitystore.enabled | bool | `true` | Enable identity service |
| identitystore.extraContainers | object | `{}` | Extra POD containers |
| identitystore.extraVolumeMounts | object | `{}` | Extra volume mounts |
| identitystore.extraVolumes | object | `{}` | Extra volumes |
| identitystore.fullnameOverride | string | `nil` | Full name to override |
| identitystore.hubId | string | `nil` | Hub ID. Overrides the global.hubId |
| identitystore.image | object | `{"imagePullSecrets":{},"pullPolicy":"Always","registry":"ghcr.io/","repository":"plgd-dev/hub/identity-store","tag":null}` | Identity service image section |
| identitystore.image.imagePullSecrets | object | `{}` | Image pull secrets |
| identitystore.image.pullPolicy | string | `"Always"` | Image pull policy |
| identitystore.image.registry | string | `"ghcr.io/"` | Image registry |
| identitystore.image.repository | string | `"plgd-dev/hub/identity-store"` | Image repository |
| identitystore.image.tag | string | `nil` | Image tag. |
| identitystore.imagePullSecrets | object | `{}` | Image pull secrets |
| identitystore.initContainersTpl | object | `{}` | Init containers definition. Resolved as template |
| identitystore.livenessProbe | object | `{}` | Liveness probe. Identity doesn't have any default liveness probe |
| identitystore.log | object | `{"dumpBody":false,"encoderConfig":{"timeEncoder":"rfc3339nano"},"encoding":"json","level":"info","stacktrace":{"enabled":false,"level":"warn"}}` | Log section |
| identitystore.log.dumpBody | bool | `false` | Dump grpc messages |
| identitystore.log.encoderConfig.timeEncoder | string | `"rfc3339nano"` | Time format for logs. The supported values are: "rfc3339nano", "rfc3339" |
| identitystore.log.encoding | string | `"json"` | The supported values are: "json", "console" |
| identitystore.log.level | string | `"info"` | Logging enabled from level |
| identitystore.log.stacktrace.enabled | bool | `false` | Log stacktrace |
| identitystore.log.stacktrace.level | string | `"warn"` | Stacktrace from level |
| identitystore.name | string | `"identity-store"` | Name of component. Used in label selectors |
| identitystore.nodeSelector | object | `{}` | Node selector |
| identitystore.podAnnotations | object | `{}` | Annotations for identity pod |
| identitystore.podLabels | object | `{}` | Labels for identity pod |
| identitystore.podSecurityContext | object | `{}` | Pod security context |
| identitystore.port | int | `9100` | Service and POD port |
| identitystore.rbac | object | `{"enabled":false,"roleBindingDefitionTpl":null,"serviceAccountName":"identity-store"}` | RBAC configuration |
| identitystore.rbac.enabled | bool | `false` | Enable RBAC setup |
| identitystore.rbac.roleBindingDefitionTpl | string | `nil` | Template definition for Role/binding etc.. Resolved as template |
| identitystore.rbac.serviceAccountName | string | `"identity-store"` | Name of identity SA |
| identitystore.readinessProbe | object | `{}` | Readiness probe. Identity doesn't have aby default readiness probe |
| identitystore.replicas | int | `1` | Number of replicas |
| identitystore.resources | object | `{}` | Resources limit |
| identitystore.restartPolicy | string | `"Always"` | Restart policy for pod |
| identitystore.securityContext | object | `{}` | Security context for pod |
| identitystore.service | object | `{"annotations":{},"crt":{"extraDnsNames":[]},"labels":{},"name":"grpc","protocol":"TCP","targetPort":"grpc","type":"ClusterIP"}` | Service configuration |
| identitystore.service.annotations | object | `{}` | Service annotations |
| identitystore.service.crt.extraDnsNames | list | `[]` | Extra DNS names for service certificate |
| identitystore.service.labels | object | `{}` | Service labels |
| identitystore.service.name | string | `"grpc"` | Name |
| identitystore.service.protocol | string | `"TCP"` | Protocol |
| identitystore.service.targetPort | string | `"grpc"` | Target port |
| identitystore.service.type | string | `"ClusterIP"` | Service type |
| identitystore.tolerations | object | `{}` | Toleration definition |
| m2moauthserver.affinity | object | `{}` | Affinity definition |
| m2moauthserver.apis | object | `{"grpc":{"address":"","authorization":{"audience":null,"authority":null,"http":{"idleConnTimeout":"30s","maxConnsPerHost":32,"maxIdleConns":16,"maxIdleConnsPerHost":16,"timeout":"10s","tls":{"caPool":null,"certFile":null,"keyFile":null,"useSystemCAPool":false}},"ownerClaim":null},"enforcementPolicy":{"minTime":"5s","permitWithoutStream":true},"keepAlive":{"maxConnectionAge":"0s","maxConnectionAgeGrace":"0s","maxConnectionIdle":"0s","time":"2h","timeout":"20s"},"recvMsgSize":4194304,"sendMsgSize":4194304,"tls":{"caPool":null,"certFile":null,"clientCertificateRequired":false,"keyFile":null}},"http":{"address":null,"idleTimeout":"30s","readHeaderTimeout":"4s","readTimeout":"8s","writeTimeout":"16s"}}` | For complete m2m-oauth-server service configuration see [plgd/oauth-server](https://github.com/plgd-dev/hub/tree/main/test/oauth-server) |
| m2moauthserver.clients.storage.cleanUpDeletedTokens | string | `"0 * * * *"` |  |
| m2moauthserver.clients.storage.mongoDB.database | string | `"m2mOAuthServer"` |  |
| m2moauthserver.clients.storage.mongoDB.maxConnIdleTime | string | `"4m0s"` |  |
| m2moauthserver.clients.storage.mongoDB.maxPoolSize | int | `16` |  |
| m2moauthserver.clients.storage.mongoDB.tls.caPool | string | `nil` |  |
| m2moauthserver.clients.storage.mongoDB.tls.certFile | string | `nil` |  |
| m2moauthserver.clients.storage.mongoDB.tls.keyFile | string | `nil` |  |
| m2moauthserver.clients.storage.mongoDB.tls.useSystemCAPool | bool | `false` |  |
| m2moauthserver.clients.storage.mongoDB.uri | string | `nil` |  |
| m2moauthserver.clients.storage.use | string | `"mongoDB"` |  |
| m2moauthserver.config | object | `{"fileName":"service.yaml","mountPath":"/config","volume":"config"}` | m2m-oauth-server service yaml config section |
| m2moauthserver.config.fileName | string | `"service.yaml"` | Name of configuration file |
| m2moauthserver.config.mountPath | string | `"/config"` | Mount path |
| m2moauthserver.config.volume | string | `"config"` | Volume for configuration file |
| m2moauthserver.deploymentAnnotations | object | `{}` | Additional annotations for m2m-oauth-server deployment |
| m2moauthserver.deploymentLabels | object | `{}` | Additional labels for m2m-oauth-server deployment |
| m2moauthserver.domain | string | `nil` | Domain for oauth. Default {{ global.domain }} |
| m2moauthserver.enabled | bool | `true` | Enable m2m-oauth-server service |
| m2moauthserver.extraVolumeMounts | object | `{}` | Optional extra volume mounts |
| m2moauthserver.extraVolumes | object | `{}` | Optional extra volumes |
| m2moauthserver.fullnameOverride | string | `nil` | Full name to override |
| m2moauthserver.httpPort | int | `9101` |  |
| m2moauthserver.image.imagePullSecrets | object | `{}` | Image pull secrets |
| m2moauthserver.image.pullPolicy | string | `"Always"` | Image pull policy |
| m2moauthserver.image.registry | string | `"ghcr.io/"` | Image registry |
| m2moauthserver.image.repository | string | `"plgd-dev/hub/m2m-oauth-server"` | Image repository |
| m2moauthserver.image.tag | string | `nil` | Image tag. |
| m2moauthserver.imagePullSecrets | object | `{}` | Image pull secrets |
| m2moauthserver.ingress.grpc.annotations | object | `{"cert-manager.io/private-key-rotation-policy":"always","ingress.kubernetes.io/force-ssl-redirect":"true","nginx.ingress.kubernetes.io/backend-protocol":"GRPCS","nginx.ingress.kubernetes.io/enable-cors":"true","nginx.org/grpc-services":"{{ include \"plgd-hub.m2moauthserver.fullname\" . }}-grpc"}` | Pre defined map of Ingress annotation |
| m2moauthserver.ingress.grpc.customAnnotations | object | `{}` | Custom map of Ingress annotation |
| m2moauthserver.ingress.grpc.enabled | bool | `true` | Enable ingress |
| m2moauthserver.ingress.grpc.paths | list | `["/m2moauthserver.pb.M2MOAuthService"]` | Paths |
| m2moauthserver.ingress.grpc.secretName | string | `nil` | Override name of host/tls secret. If not specified, it will be generated |
| m2moauthserver.ingress.http.allowHeaders | string | `"Authortity,Method,Path,Scheme,Accept,Accept-Encoding,Accept-Language,Content-Type,auth0-client,Origin,Refer,Sec-Fetch-Dest,Sec-Fetch-Mode,Sec-Fetch-Site,Authorization,DNT,User-Agent,X-Requested-With,If-Modified-Since,Cache-Control,Content-Type,Range"` |  |
| m2moauthserver.ingress.http.annotations | object | `{"cert-manager.io/private-key-rotation-policy":"always","ingress.kubernetes.io/force-ssl-redirect":"true","nginx.ingress.kubernetes.io/backend-protocol":"HTTPS","nginx.ingress.kubernetes.io/enable-cors":"true"}` | Pre defined map of Ingress annotation |
| m2moauthserver.ingress.http.customAnnotations | object | `{}` | Custom map of Ingress annotation |
| m2moauthserver.ingress.http.enabled | bool | `true` | Enable ingress |
| m2moauthserver.ingress.http.paths | list | `["/m2m-oauth-server"]` | Ingress path |
| m2moauthserver.ingress.http.secretName | string | `nil` | Override name of host/tls secret. If not specified, it will be generated |
| m2moauthserver.livenessProbe | object | `{}` | Liveness probe. m2m-oauth-server doesn't have any default liveness probe |
| m2moauthserver.log.encoderConfig.timeEncoder | string | `"rfc3339nano"` | Time format for logs. The supported values are: "rfc3339nano", "rfc3339" |
| m2moauthserver.log.encoding | string | `"json"` | The supported values are: "json", "console" |
| m2moauthserver.log.level | string | `"info"` | Logging enabled from level |
| m2moauthserver.log.stacktrace.enabled | bool | `false` | Log stacktrace |
| m2moauthserver.log.stacktrace.level | string | `"warn"` | Stacktrace from level |
| m2moauthserver.name | string | `"m2m-oauth-server"` | Name of component. Used in label selectors |
| m2moauthserver.nodeSelector | object | `{}` | Node selector |
| m2moauthserver.oauthSigner.clients[0].accessTokenLifetime | string | `"0s"` |  |
| m2moauthserver.oauthSigner.clients[0].allowedAudiences | list | `[]` |  |
| m2moauthserver.oauthSigner.clients[0].allowedGrantTypes[0] | string | `"client_credentials"` |  |
| m2moauthserver.oauthSigner.clients[0].allowedScopes | list | `[]` |  |
| m2moauthserver.oauthSigner.clients[0].id | string | `"jwt-private-key"` |  |
| m2moauthserver.oauthSigner.clients[0].jwtPrivateKey.authorization.audience | string | `nil` |  |
| m2moauthserver.oauthSigner.clients[0].jwtPrivateKey.authorization.endpoints | string | `nil` |  |
| m2moauthserver.oauthSigner.clients[0].jwtPrivateKey.enabled | bool | `true` |  |
| m2moauthserver.oauthSigner.deviceIDClaim | string | `nil` |  |
| m2moauthserver.oauthSigner.domain | string | `nil` |  |
| m2moauthserver.oauthSigner.ownerClaim | string | `nil` |  |
| m2moauthserver.oauthSigner.privateKeyFile | string | `nil` |  |
| m2moauthserver.podAnnotations | object | `{}` | Annotations for m2m-oauth-server pod |
| m2moauthserver.podLabels | object | `{}` | Labels for http-gateway pod |
| m2moauthserver.podSecurityContext | object | `{}` | Pod security context |
| m2moauthserver.port | int | `9100` | Port for service and POD |
| m2moauthserver.privateKey.enabled | bool | `false` | Set deployment to use secret for private key |
| m2moauthserver.privateKey.fileName | string | `"private.key"` | Name of private key file |
| m2moauthserver.privateKey.mountPath | string | `"/secrets/keys"` | Mount path |
| m2moauthserver.privateKey.secretName | string | `"m2m-private-key"` | Name of secret |
| m2moauthserver.privateKey.volume | string | `"private-key"` | Volume name |
| m2moauthserver.readinessProbe | object | `{}` | Readiness probe. m2m-oauth-server doesn't have aby default readiness probe |
| m2moauthserver.replicas | int | `1` | Number of replicas |
| m2moauthserver.resources | object | `{}` | Resources limit |
| m2moauthserver.restartPolicy | string | `"Always"` | Restart policy for pod |
| m2moauthserver.securityContext | object | `{}` | RBAC configuration |
| m2moauthserver.service.grpc.annotations | object | `{}` | Annotations for m2m-oauth-server |
| m2moauthserver.service.grpc.crt.extraDnsNames | list | `[]` | Extra DNS names for service certificate |
| m2moauthserver.service.grpc.labels | object | `{}` | Labels for m2m-oauth-server |
| m2moauthserver.service.grpc.name | string | `"grpc"` | Name |
| m2moauthserver.service.grpc.protocol | string | `"TCP"` | Protocol |
| m2moauthserver.service.grpc.targetPort | string | `"grpc"` | Target port |
| m2moauthserver.service.grpc.type | string | `"ClusterIP"` | Service type |
| m2moauthserver.service.http.annotations | object | `{}` | Annotations for m2m-oauth-server |
| m2moauthserver.service.http.crt.extraDnsNames | list | `[]` | Extra DNS names for service certificate |
| m2moauthserver.service.http.labels | object | `{}` | Labels for m2m-oauth-server |
| m2moauthserver.service.http.name | string | `"http"` | Name |
| m2moauthserver.service.http.protocol | string | `"TCP"` | Protocol |
| m2moauthserver.service.http.targetPort | string | `"http"` | Target port |
| m2moauthserver.service.http.type | string | `"ClusterIP"` | Service type |
| m2moauthserver.tolerations | object | `{}` | Toleration definition |
| mockoauthserver.affinity | object | `{}` | Affinity definition |
| mockoauthserver.apis | object | `{"http":{"address":null,"idleTimeout":"30s","readHeaderTimeout":"4s","readTimeout":"8s","tls":{"caPool":null,"certFile":null,"clientCertificateRequired":false,"keyFile":null},"writeTimeout":"16s"}}` | For complete mock-oauth-server service configuration see [plgd/oauth-server](https://github.com/plgd-dev/hub/tree/main/test/oauth-server) |
| mockoauthserver.config | object | `{"fileName":"service.yaml","mountPath":"/config","volume":"config"}` | mock-oauth-server service yaml config section |
| mockoauthserver.config.fileName | string | `"service.yaml"` | Name of configuration file |
| mockoauthserver.config.mountPath | string | `"/config"` | Mount path |
| mockoauthserver.config.volume | string | `"config"` | Volume for configuration file |
| mockoauthserver.deploymentAnnotations | object | `{}` | Additional annotations for mock-oauth-server deployment |
| mockoauthserver.deploymentLabels | object | `{}` | Additional labels for mock-oauth-server deployment |
| mockoauthserver.domain | string | `nil` | Domain for   apiDomain: Default: auth.{{ global.domain }} |
| mockoauthserver.enabled | bool | `false` | Enable mock-oauth-server service |
| mockoauthserver.extraVolumeMounts | object | `{}` | Optional extra volume mounts |
| mockoauthserver.extraVolumes | object | `{}` | Optional extra volumes |
| mockoauthserver.fullnameOverride | string | `nil` | Full name to override |
| mockoauthserver.image.imagePullSecrets | object | `{}` | Image pull secrets |
| mockoauthserver.image.pullPolicy | string | `"Always"` | Image pull policy |
| mockoauthserver.image.registry | string | `"ghcr.io/"` | Image registry |
| mockoauthserver.image.repository | string | `"plgd-dev/hub/mock-oauth-server"` | Image repository |
| mockoauthserver.image.tag | string | `nil` | Image tag. |
| mockoauthserver.imagePullSecrets | object | `{}` | Image pull secrets |
| mockoauthserver.ingress.allowHeaders | string | `"Authortity,Method,Path,Scheme,Accept,Accept-Encoding,Accept-Language,Content-Type,auth0-client,Origin,Refer,Sec-Fetch-Dest,Sec-Fetch-Mode,Sec-Fetch-Site,Authorization,DNT,User-Agent,X-Requested-With,If-Modified-Since,Cache-Control,Content-Type,Range"` |  |
| mockoauthserver.ingress.annotations | object | `{"cert-manager.io/private-key-rotation-policy":"always","ingress.kubernetes.io/force-ssl-redirect":"true","nginx.ingress.kubernetes.io/backend-protocol":"HTTPS","nginx.ingress.kubernetes.io/configuration-snippet":"more_set_headers \"Host $host\";\nmore_set_headers \"X-Forwarded-Host $host\";\nmore_set_headers \"X-Forwarded-Proto $scheme\";\nset $cors \"true\";\nif ($request_method = 'OPTIONS') {\n  set $cors \"${cors}options\";\n}\nif ($cors = \"trueoptions\") {\n  add_header 'Access-Control-Allow-Origin' \"$http_origin\";\n  add_header 'Access-Control-Allow-Credentials' 'true';\n  add_header 'Access-Control-Allow-Methods' 'GET, PUT, POST, DELETE, PATCH, OPTIONS';\n  add_header 'Access-Control-Allow-Headers' '{{ .Values.mockoauthserver.ingress.allowHeaders }}';\n  add_header 'Access-Control-Expose-Headers' 'Content-Length,Content-Range';\n  add_header 'Access-Control-Max-Age' 1728000;\n  add_header 'Content-Type' 'text/plain charset=UTF-8';\n  add_header 'Content-Length' 0;\n  return 204;\n}\nif ($request_method = 'POST') {\nadd_header 'Access-Control-Allow-Credentials' 'true';\n}\nif ($request_method = 'PUT') {\nadd_header 'Access-Control-Allow-Credentials' 'true';\n}\nif ($request_method = 'GET') {\n    add_header 'Access-Control-Allow-Credentials' 'true';\n}\n","nginx.ingress.kubernetes.io/enable-cors":"true"}` | Pre defined map of Ingress annotation |
| mockoauthserver.ingress.customAnnotations | object | `{}` | Custom map of Ingress annotation |
| mockoauthserver.ingress.enabled | bool | `true` | Enable ingress |
| mockoauthserver.ingress.paths | list | `["/authorize","/oauth/token","/.well-known/jwks.json","/.well-known/openid-configuration","/v2/logout","/authorize/userinfo"]` | Ingress path |
| mockoauthserver.livenessProbe | object | `{}` | Liveness probe. mock-oauth-server doesn't have any default liveness probe |
| mockoauthserver.log.encoderConfig.timeEncoder | string | `"rfc3339nano"` | Time format for logs. The supported values are: "rfc3339nano", "rfc3339" |
| mockoauthserver.log.encoding | string | `"json"` | The supported values are: "json", "console" |
| mockoauthserver.log.level | string | `"info"` | Logging enabled from level |
| mockoauthserver.log.stacktrace.enabled | bool | `false` | Log stacktrace |
| mockoauthserver.log.stacktrace.level | string | `"warn"` | Stacktrace from level |
| mockoauthserver.name | string | `"mock-oauth-server"` | Name of component. Used in label selectors |
| mockoauthserver.nodeSelector | object | `{}` | Node selector |
| mockoauthserver.oauthSigner.accessTokenKeyFile | string | `"/keys/accessToken.key"` |  |
| mockoauthserver.oauthSigner.clients.accessTokenLifetime | string | `"0s"` |  |
| mockoauthserver.oauthSigner.clients.authorizationCodeLifetime | string | `"10m"` |  |
| mockoauthserver.oauthSigner.clients.codeRestrictionLifetime | string | `"0s"` |  |
| mockoauthserver.oauthSigner.clients.id | string | `"test"` |  |
| mockoauthserver.oauthSigner.clients.refreshTokenRestrictionLifetime | string | `"0s"` |  |
| mockoauthserver.oauthSigner.domain | string | `nil` |  |
| mockoauthserver.oauthSigner.idTokenKeyFile | string | `"/keys/idToken.key"` |  |
| mockoauthserver.oauth[0].clientID | string | `"test"` |  |
| mockoauthserver.oauth[0].clientSecret | string | `"test"` |  |
| mockoauthserver.oauth[0].name | string | `"plgd.mobile"` |  |
| mockoauthserver.oauth[0].redirectURL | string | `"cloud.plgd.mobile://login-callback"` |  |
| mockoauthserver.oauth[1].clientID | string | `"test"` |  |
| mockoauthserver.oauth[1].clientSecret | string | `"test"` |  |
| mockoauthserver.oauth[1].name | string | `"plgd.web"` |  |
| mockoauthserver.oauth[1].redirectURL | string | `"{{ printf \"https://%s\" ( include \"plgd-hub.mockoauthserver.ingressDomain\" . ) }}/devices"` |  |
| mockoauthserver.oauth[1].useInUi | bool | `true` |  |
| mockoauthserver.podAnnotations | object | `{}` | Annotations for mock-oauth-server pod |
| mockoauthserver.podLabels | object | `{}` | Labels for http-gateway pod |
| mockoauthserver.podSecurityContext | object | `{}` | Pod security context |
| mockoauthserver.port | int | `9100` | Port for service and POD |
| mockoauthserver.readinessProbe | object | `{}` | Readiness probe. mock-oauth-server doesn't have aby default readiness probe |
| mockoauthserver.replicas | int | `1` | Number of replicas |
| mockoauthserver.resources | object | `{}` | Resources limit |
| mockoauthserver.restartPolicy | string | `"Always"` | Restart policy for pod |
| mockoauthserver.securityContext | object | `{}` |  |
| mockoauthserver.service.annotations | object | `{}` | Annotations for mock-oauth-server service |
| mockoauthserver.service.crt.extraDnsNames | list | `[]` | Extra DNS names for service certificate |
| mockoauthserver.service.labels | object | `{}` | Labels for mock-oauth-server service |
| mockoauthserver.service.name | string | `"http"` | Name |
| mockoauthserver.service.protocol | string | `"TCP"` | Protocol |
| mockoauthserver.service.targetPort | string | `"http"` | Target port |
| mockoauthserver.service.type | string | `"ClusterIP"` |  |
| mockoauthserver.tolerations | object | `{}` | Toleration definition |
| mongodb | object | `{"arbiter":{"enabled":false},"architecture":"replicaset","auth":{"enabled":false},"customLivenessProbe":{"exec":{"command":["/bin/bash","-c","/certs/livenessProbe.sh"]},"failureThreshold":6,"initialDelaySeconds":30,"periodSeconds":20,"successThreshold":1,"timeoutSeconds":10},"customReadinessProbe":{"exec":{"command":["bash","-ec","TLS_OPTIONS='--tls --tlsCertificateKeyFile=/certs/cert.pem --tlsCAFile=/certs/ca.pem'\nmongosh $TLS_OPTIONS --eval 'db.hello().isWritablePrimary || db.hello().secondary' | grep -q 'true'\n"]},"failureThreshold":6,"initialDelaySeconds":10,"periodSeconds":20,"successThreshold":1,"timeoutSeconds":10},"enabled":true,"extraEnvVars":[{"name":"MONGODB_EXTRA_FLAGS","value":"--tlsMode=requireTLS --tlsCertificateKeyFile=/certs/cert.pem --tlsCAFile=/certs/ca.pem"},{"name":"MONGODB_CLIENT_EXTRA_FLAGS","value":"--tls --tlsCertificateKeyFile=/certs/cert.pem --tlsCAFile=/certs/ca.pem"}],"extraVolumeMounts":[{"mountPath":"/certs","name":"mongodb-crt"},{"mountPath":"/certs-original","name":"mongodb-cm-crt"},{"mountPath":"/certs/extra/storage","name":"mongodb-extra-ca-pool"}],"extraVolumes":[{"emptyDir":{},"name":"mongodb-crt"},{"name":"mongodb-cm-crt","secret":{"secretName":"mongodb-cm-crt"}},{"name":"mongodb-extra-ca-pool","secret":{"secretName":"mongodb-extra-ca-pool"}}],"fullnameOverride":"mongodb","image":{"debug":true},"initContainers":[{"command":["sh","-c","/bin/bash <<'EOF'\n#!/bin/bash\ncp /usr/local/bin/mongodb-admin-tool /certs\necho '\n#!/bin/bash\nset -e\nINIT=0\nwhile [[ $# -gt 0 ]]; do\n  case $1 in\n    --init)\n      INIT=1\n      shift\n      ;;\n  esac\ndone\n\nCERT_CRT=/certs-original/tls.crt\nCERT_SHA256=/certs/cert.sha256.$(sha256sum ${CERT_CRT} | cut -d \" \" -f 1)\nCA=/certs-original/ca.crt\nEXTRA_CA=/dev/null\nif [ -f /certs/extra/storage/ca.crt ]; then\n  EXTRA_CA=/certs/extra/storage/ca.crt\nfi\nCA_SHA256=$(cat ${CA} ${EXTRA_CA} | sha256sum | cut -d \" \" -f 1)\nCA_FILE_SHA256=/certs/ca.sha256.$CA_SHA256\nROTATE_CERTIFICATES=0\nif [ ! -f ${CERT_SHA256} ]; then\n  rm -f /certs/cert.sha256.*\n  cat ${CERT_CRT} > /certs/cert.pem\n  cat /certs-original/tls.key >> /certs/cert.pem\n  touch ${CERT_SHA256}\n  ROTATE_CERTIFICATES=1\nfi\n\nif [ ! -f ${CA_FILE_SHA256} ]; then\n  rm -f /certs/ca.sha256.*\n  cat ${CA} ${EXTRA_CA} > /certs/ca.pem\n  touch ${CA_FILE_SHA256}\n  ROTATE_CERTIFICATES=1\nfi\n\nif [ \"${INIT}\" == \"1\" ]; then\n  exit 0\nfi\n\nif [ \"${ROTATE_CERTIFICATES}\" == \"1\" ]; then\n  echo \"Rotating certificates\"\n  /certs/mongodb-admin-tool --tls --directConnection --tlsCertificateKeyFile=/certs/cert.pem --tlsCAFile=/certs/ca.pem --eval db.adminCommand\\(\"{rotateCertificates: 1, message: \\\"Rotating certificates\\\"}\"\\)\nelse\n  echo \"Ping database\"\n  /certs/mongodb-admin-tool --tls --directConnection --tlsCertificateKeyFile=/certs/cert.pem --tlsCAFile=/certs/ca.pem --eval db.adminCommand\\(\\\"ping\\\"\\)\nfi\n' > /certs/livenessProbe.sh\nchmod uga+x /certs/livenessProbe.sh\n/certs/livenessProbe.sh --init\nEOF\n"],"image":"ghcr.io/plgd-dev/hub/mongodb-admin-tool:vnext","imagePullPolicy":"Always","name":"mongo-binary","securityContext":{"runAsGroup":1001,"runAsUser":1001},"volumeMounts":[{"mountPath":"/certs","name":"mongodb-crt"},{"mountPath":"/certs-original","name":"mongodb-cm-crt"},{"mountPath":"/certs/extra/storage","name":"mongodb-extra-ca-pool"}]}],"livenessProbe":{"enabled":false},"persistence":{"enabled":true},"readinessProbe":{"enabled":false},"replicaCount":3,"replicaSetName":"rs0","standbyTool":{"affinity":{},"clients":{"storage":{"mongoDB":{"timeout":"30s","tls":{"caPool":null,"certFile":null,"keyFile":null,"useSystemCAPool":false}}}},"config":{"fileName":"config.yaml","mountPath":"/config","volume":"config"},"enabled":false,"extraVolumeMounts":{},"extraVolumes":{},"fullnameOverride":null,"image":{"imagePullSecrets":{},"pullPolicy":"Always","registry":"ghcr.io/","repository":"plgd-dev/hub/mongodb-standby-tool","tag":null},"jobAnnotations":{},"jobLabels":{},"log":{"dumpBody":false,"encoderConfig":{"timeEncoder":"rfc3339nano"},"encoding":"json","level":"info","stacktrace":{"enabled":false,"level":"warn"}},"mode":"standby","name":"mongodb-standby-tool","nodeSelector":{},"podAnnotations":{},"podSecurityContext":{},"rbac":{"enabled":false,"roleBindingDefitionTpl":null,"serviceAccountName":"mongodb-standby-tool"},"replicaSet":{"forceUpdate":false,"maxWaitsForReady":30,"secondary":{"priority":10,"votes":1},"standby":{"delays":"10m","members":[]}},"resources":{},"securityContext":{},"tolerations":{}},"tls":{"enabled":false}}` | External mongodb-replica dependency setup |
| mongodb.standbyTool.affinity | object | `{}` | Affinity definition |
| mongodb.standbyTool.clients.storage.mongoDB.timeout | string | `"30s"` | Timeout for connection to MongoDB and read/write operations |
| mongodb.standbyTool.clients.storage.mongoDB.tls.caPool | string | `nil` | Path to the CA certificate file |
| mongodb.standbyTool.clients.storage.mongoDB.tls.certFile | string | `nil` | The certFile and keyFile are the paths to the TLS certificate pair files |
| mongodb.standbyTool.clients.storage.mongoDB.tls.keyFile | string | `nil` | The keyFile are the paths to the TLS certificate pair files |
| mongodb.standbyTool.clients.storage.mongoDB.tls.useSystemCAPool | bool | `false` | Path to the CA certificate file |
| mongodb.standbyTool.config | object | `{"fileName":"config.yaml","mountPath":"/config","volume":"config"}` | Job configuration |
| mongodb.standbyTool.config.fileName | string | `"config.yaml"` | Job configuration file |
| mongodb.standbyTool.config.mountPath | string | `"/config"` | Configuration mount path |
| mongodb.standbyTool.config.volume | string | `"config"` | Job configuration volume name |
| mongodb.standbyTool.enabled | bool | `false` | Create standby job |
| mongodb.standbyTool.extraVolumeMounts | object | `{}` | Optional extra volume mounts |
| mongodb.standbyTool.extraVolumes | object | `{}` | Optional extra volumes |
| mongodb.standbyTool.fullnameOverride | string | `nil` | Full name to override |
| mongodb.standbyTool.image.imagePullSecrets | object | `{}` | Image pull secrets |
| mongodb.standbyTool.image.pullPolicy | string | `"Always"` | Image pull policy |
| mongodb.standbyTool.image.registry | string | `"ghcr.io/"` | Image registry |
| mongodb.standbyTool.image.repository | string | `"plgd-dev/hub/mongodb-standby-tool"` | Image repository |
| mongodb.standbyTool.image.tag | string | `nil` | Image tag. |
| mongodb.standbyTool.jobAnnotations | object | `{}` | Additional annotations for mongodb-standby job |
| mongodb.standbyTool.jobLabels | object | `{}` | Additional labels for mongodb-standby job |
| mongodb.standbyTool.log | object | `{"dumpBody":false,"encoderConfig":{"timeEncoder":"rfc3339nano"},"encoding":"json","level":"info","stacktrace":{"enabled":false,"level":"warn"}}` | Log section |
| mongodb.standbyTool.log.dumpBody | bool | `false` | Dump grpc messages |
| mongodb.standbyTool.log.encoderConfig.timeEncoder | string | `"rfc3339nano"` | Time format for logs. The supported values are: "rfc3339nano", "rfc3339" |
| mongodb.standbyTool.log.encoding | string | `"json"` | The supported values are: "json", "console" |
| mongodb.standbyTool.log.level | string | `"info"` | Logging enabled from level |
| mongodb.standbyTool.log.stacktrace.enabled | bool | `false` | Log stacktrace |
| mongodb.standbyTool.log.stacktrace.level | string | `"warn"` | Stacktrace from level |
| mongodb.standbyTool.mode | string | `"standby"` | Mode of standby job. Supported values: "active", "standby" |
| mongodb.standbyTool.name | string | `"mongodb-standby-tool"` | Name of component. Used in label selectors |
| mongodb.standbyTool.nodeSelector | object | `{}` | Node selector |
| mongodb.standbyTool.podAnnotations | object | `{}` | Annotations for mongodb-standby pod |
| mongodb.standbyTool.podSecurityContext | object | `{}` | Pod security context |
| mongodb.standbyTool.rbac.roleBindingDefitionTpl | string | `nil` | template definition for Role/binding etc.. |
| mongodb.standbyTool.rbac.serviceAccountName | string | `"mongodb-standby-tool"` | Name of mongodb-standby SA |
| mongodb.standbyTool.replicaSet | object | `{"forceUpdate":false,"maxWaitsForReady":30,"secondary":{"priority":10,"votes":1},"standby":{"delays":"10m","members":[]}}` | Standby members of replica set |
| mongodb.standbyTool.replicaSet.forceUpdate | bool | `false` | Update the replica set configuration with force flag |
| mongodb.standbyTool.replicaSet.maxWaitsForReady | int | `30` | Set the maximum number of waits for becomes members ready. |
| mongodb.standbyTool.resources | object | `{}` | Resources limit |
| mongodb.standbyTool.securityContext | object | `{}` | Security context for pod |
| mongodb.standbyTool.tolerations | object | `{}` | Toleration definition |
| nats | object | `{"config":{"nats":{"tls":{"enabled":true,"merge":{"verify":true},"secretName":"nats-service-crt"}}},"enabled":true,"monitor":{"enabled":false},"natsBox":{"enabled":false},"tlsCA":{"enabled":true,"secretName":"nats-service-crt"}}` | External nats dependency setup |
| resourceaggregate.affinity | object | `{}` | Affinity definition |
| resourceaggregate.apis | object | `{"grpc":{"address":null,"authorization":{"audience":null,"authority":null,"http":{"idleConnTimeout":"30s","maxConnsPerHost":32,"maxIdleConns":16,"maxIdleConnsPerHost":16,"timeout":"10s","tls":{"caPool":null,"certFile":null,"keyFile":null,"useSystemCAPool":true}},"ownerClaim":null},"enforcementPolicy":{"minTime":"5s","permitWithoutStream":true},"keepAlive":{"maxConnectionAge":"0s","maxConnectionAgeGrace":"0s","maxConnectionIdle":"0s","time":"2h","timeout":"20s"},"ownerCacheExpiration":"1m","recvMsgSize":4194304,"sendMsgSize":4194304,"tls":{"caPool":null,"certFile":null,"clientCertificateRequired":true,"keyFile":null}}}` | For complete resource-aggregate service configuration see [plgd/resource-aggregate](https://github.com/plgd-dev/hub/tree/main/resource-aggregate) |
| resourceaggregate.clients | object | `{"eventBus":{"nats":{"flusherTimeout":"30s","jetstream":false,"pendingLimits":{"bytesLimit":"67108864","msgLimit":524288},"tls":{"caPool":null,"certFile":null,"keyFile":null,"useSystemCAPool":false},"url":null}},"eventStore":{"cqlDB":{"connectTimeout":"10s","hosts":[],"keyspace":{"create":true,"name":"plgdhub","replication":{"class":"SimpleStrategy","replication_factor":1}},"numConnections":16,"port":9142,"reconnectionPolicy":{"constant":{"interval":"3s","maxRetries":3}},"table":"events","tls":{"caPool":null,"certFile":null,"keyFile":null,"useSystemCAPool":false},"useHostnameResolution":true},"defaultCommandTimeToLive":null,"mongoDB":{"batchSize":128,"database":"eventStore","maxConnIdleTime":"4m0s","maxPoolSize":16,"tls":{"caPool":null,"certFile":null,"keyFile":null,"useSystemCAPool":false},"uri":null},"occMaxRetry":8,"use":"mongoDB"},"identityStore":{"grpc":{"address":null,"keepAlive":{"permitWithoutStream":true,"time":"10s","timeout":"20s"},"recvMsgSize":4194304,"sendMsgSize":4194304,"tls":{"caPool":null,"certFile":null,"keyFile":null,"useSystemCAPool":false}}}}` | For complete resource-aggregate service configuration see [plgd/resource-aggregate](https://github.com/plgd-dev/hub/tree/main/resource-aggregate) |
| resourceaggregate.clients.eventStore.cqlDB.useHostnameResolution | bool | `true` | Resolve IP address to hostname before validate certificate. If false, the TLS validator will use ip/hostname advertised by the Cassandra node. |
| resourceaggregate.config | object | `{"fileName":"service.yaml","mountPath":"/config","volume":"config"}` | Service configuration |
| resourceaggregate.config.fileName | string | `"service.yaml"` | Service configuration file name |
| resourceaggregate.config.mountPath | string | `"/config"` | Configuration mount path |
| resourceaggregate.config.volume | string | `"config"` | Volume name |
| resourceaggregate.deploymentAnnotations | object | `{}` | Additional annotations for resource-aggregate deployment |
| resourceaggregate.deploymentLabels | object | `{}` | Additional labels for resource-aggregate deployment |
| resourceaggregate.enabled | bool | `true` | Enable resource-aggregate service |
| resourceaggregate.extraContainers | object | `{}` | Extra POD containers |
| resourceaggregate.extraVolumeMounts | object | `{}` | Optional extra volume mounts |
| resourceaggregate.extraVolumes | object | `{}` | Optional extra volumes |
| resourceaggregate.fullnameOverride | string | `nil` | Full name to override |
| resourceaggregate.hubId | string | `nil` | Hub ID. Overrides the global.hubId |
| resourceaggregate.image.imagePullSecrets | object | `{}` | Image pull secrets |
| resourceaggregate.image.pullPolicy | string | `"Always"` | Image pull policy |
| resourceaggregate.image.registry | string | `"ghcr.io/"` | Image registry |
| resourceaggregate.image.repository | string | `"plgd-dev/hub/resource-aggregate"` | Image repository |
| resourceaggregate.image.tag | string | `nil` | Image tag. |
| resourceaggregate.imagePullSecrets | object | `{}` | Image pull secrets |
| resourceaggregate.initContainersTpl | object | `{}` | Init containers definition. Resolved as template |
| resourceaggregate.livenessProbe | object | `{}` | Liveness probe. resource-aggregate doesn't have any default liveness probe |
| resourceaggregate.log | object | `{"dumpBody":false,"encoderConfig":{"timeEncoder":"rfc3339nano"},"encoding":"json","level":"info","stacktrace":{"enabled":false,"level":"warn"}}` | Log section |
| resourceaggregate.log.dumpBody | bool | `false` | Dump grpc messages |
| resourceaggregate.log.encoderConfig.timeEncoder | string | `"rfc3339nano"` | Time format for logs. The supported values are: "rfc3339nano", "rfc3339" |
| resourceaggregate.log.encoding | string | `"json"` | The supported values are: "json", "console" |
| resourceaggregate.log.level | string | `"info"` | Logging enabled from level |
| resourceaggregate.log.stacktrace.enabled | bool | `false` | Log stacktrace |
| resourceaggregate.log.stacktrace.level | string | `"warn"` | Stacktrace from level |
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
| resourceaggregate.service.crt.extraDnsNames | list | `[]` | Extra DNS names for service certificate |
| resourceaggregate.service.labels | object | `{}` | Labels for resource-aggregate service |
| resourceaggregate.service.name | string | `"grpc"` | Name |
| resourceaggregate.service.protocol | string | `"TCP"` | Protocol |
| resourceaggregate.service.targetPort | string | `"grpc"` | Target port |
| resourceaggregate.service.type | string | `"ClusterIP"` | Service type |
| resourceaggregate.tolerations | object | `{}` | Toleration definition |
| resourcedirectory.affinity | object | `{}` | Affinity definition |
| resourcedirectory.apis | object | `{"grpc":{"address":null,"authorization":{"audience":null,"authority":null,"http":{"idleConnTimeout":"30s","maxConnsPerHost":32,"maxIdleConns":16,"maxIdleConnsPerHost":16,"timeout":"10s","tls":{"caPool":null,"certFile":null,"keyFile":null,"useSystemCAPool":true}},"ownerClaim":null},"enforcementPolicy":{"minTime":"5s","permitWithoutStream":true},"keepAlive":{"maxConnectionAge":"0s","maxConnectionAgeGrace":"0s","maxConnectionIdle":"0s","time":"2h","timeout":"20s"},"ownerCacheExpiration":"1m","recvMsgSize":4194304,"sendMsgSize":4194304,"tls":{"caPool":null,"certFile":null,"clientCertificateRequired":true,"keyFile":null}}}` | For complete resource-directory service configuration see [plgd/resource-directory](https://github.com/plgd-dev/hub/tree/main/resource-directory) |
| resourcedirectory.clients | object | `{"eventBus":{"goPoolSize":16,"nats":{"pendingLimits":{"bytesLimit":"67108864","msgLimit":"524288"},"tls":{"caPool":null,"certFile":null,"keyFile":null,"useSystemCAPool":false},"url":""}},"eventStore":{"cacheExpiration":"20m","cqlDB":{"connectTimeout":"10s","hosts":[],"keyspace":{"create":true,"name":"plgdhub","replication":{"class":"SimpleStrategy","replication_factor":1}},"numConnections":16,"port":9142,"reconnectionPolicy":{"constant":{"interval":"3s","maxRetries":3}},"table":"events","tls":{"caPool":null,"certFile":null,"keyFile":null,"useSystemCAPool":false},"useHostnameResolution":true},"mongoDB":{"batchSize":128,"database":"eventStore","maxConnIdleTime":"4m0s","maxPoolSize":16,"tls":{"caPool":null,"certFile":null,"keyFile":null,"useSystemCAPool":false},"uri":""},"use":"mongoDB"},"identityStore":{"cacheExpiration":"1m","grpc":{"address":"","keepAlive":{"permitWithoutStream":true,"time":"10s","timeout":"20s"},"recvMsgSize":4194304,"sendMsgSize":4194304,"tls":{"caPool":null,"certFile":null,"keyFile":null,"useSystemCAPool":false}},"oauth":{"audience":"","clientID":null,"clientSecret":null,"http":{"idleConnTimeout":"30s","maxConnsPerHost":32,"maxIdleConns":16,"maxIdleConnsPerHost":16,"timeout":"10s","tls":{"caPool":null,"certFile":null,"keyFile":null,"useSystemCAPool":false}},"scopes":[],"tokenURL":"","verifyServiceTokenFrequency":"10s"},"ownerClaim":null,"pullFrequency":"15s"}}` | For complete resource-directory service configuration see [plgd/resource-directory](https://github.com/plgd-dev/hub/tree/main/resource-directory) |
| resourcedirectory.clients.eventStore.cqlDB.useHostnameResolution | bool | `true` | Resolve IP address to hostname before validate certificate. If false, the TLS validator will use ip/hostname advertised by the Cassandra node. |
| resourcedirectory.config | object | `{"fileName":"service.yaml","mountPath":"/config","volume":"config"}` | Service configuration |
| resourcedirectory.config.fileName | string | `"service.yaml"` | Service configuration file |
| resourcedirectory.config.mountPath | string | `"/config"` | Configuration mount path |
| resourcedirectory.config.volume | string | `"config"` | Service configuration volume name |
| resourcedirectory.deploymentAnnotations | object | `{}` | Additional annotations for resource-directory deployment |
| resourcedirectory.deploymentLabels | object | `{}` | Additional labels for resource-directory deployment |
| resourcedirectory.enabled | bool | `true` | Enable resource-directory service |
| resourcedirectory.extraContainers | object | `{}` | Extra POD containers |
| resourcedirectory.extraVolumeMounts | object | `{}` | Optional extra volume mounts |
| resourcedirectory.extraVolumes | object | `{}` | Optional extra volumes |
| resourcedirectory.fullnameOverride | string | `nil` | Full name to override |
| resourcedirectory.hubId | string | `nil` | Hub ID. Overrides the global.hubId |
| resourcedirectory.image.command | string | `nil` | Container command |
| resourcedirectory.image.imagePullSecrets | object | `{}` | Image pull secrets |
| resourcedirectory.image.pullPolicy | string | `"Always"` | Image pull policy |
| resourcedirectory.image.registry | string | `"ghcr.io/"` | Image registry |
| resourcedirectory.image.repository | string | `"plgd-dev/hub/resource-directory"` | Image repository |
| resourcedirectory.image.tag | string | `nil` | Image tag. |
| resourcedirectory.initContainersTpl | object | `{}` | Init containers definition. Resolved as template |
| resourcedirectory.livenessProbe | object | `{}` | Liveness probe. resource-directory doesn't have any default liveness probe |
| resourcedirectory.log | object | `{"dumpBody":false,"encoderConfig":{"timeEncoder":"rfc3339nano"},"encoding":"json","level":"info","stacktrace":{"enabled":false,"level":"warn"}}` | Log section |
| resourcedirectory.log.dumpBody | bool | `false` | Dump grpc messages |
| resourcedirectory.log.encoderConfig.timeEncoder | string | `"rfc3339nano"` | Time format for logs. The supported values are: "rfc3339nano", "rfc3339" |
| resourcedirectory.log.encoding | string | `"json"` | The supported values are: "json", "console" |
| resourcedirectory.log.level | string | `"info"` | Logging enabled from level |
| resourcedirectory.log.stacktrace.enabled | bool | `false` | Log stacktrace |
| resourcedirectory.log.stacktrace.level | string | `"warn"` | Stacktrace from level |
| resourcedirectory.name | string | `"resource-directory"` | Name of component. Used in label selectors |
| resourcedirectory.nodeSelector | object | `{}` | Node selector |
| resourcedirectory.podAnnotations | object | `{}` | Annotations for resource-directory pod |
| resourcedirectory.podLabels | object | `{}` | Labels for resource-directory pod |
| resourcedirectory.podSecurityContext | object | `{}` | Pod security context |
| resourcedirectory.port | int | `9100` | Service and POD port |
| resourcedirectory.publicConfiguration | object | `{"authority":null,"caPool":null,"certificateAuthority":null,"coapGateway":null,"defaultCommandTimeToLive":null,"deviceIdClaim":null,"ownerClaim":null}` | For complete resource-directory service configuration see [plgd/resource-directory](https://github.com/plgd-dev/hub/tree/main/resource-directory) |
| resourcedirectory.rbac | object | `{"enabled":false,"roleBindingDefitionTpl":null,"serviceAccountName":"resource-directory"}` | RBAC configuration |
| resourcedirectory.rbac.roleBindingDefitionTpl | string | `nil` | template definition for Role/binding etc.. |
| resourcedirectory.rbac.serviceAccountName | string | `"resource-directory"` | Name of resource-directory SA |
| resourcedirectory.readinessProbe | object | `{}` | Readiness probe. resource-directory doesn't have aby default readiness probe |
| resourcedirectory.replicas | int | `1` | Number of replicas |
| resourcedirectory.resources | object | `{}` | Resources limit |
| resourcedirectory.restartPolicy | string | `"Always"` | Restart policy for pod |
| resourcedirectory.securityContext | object | `{}` | Security context for pod |
| resourcedirectory.service.annotations | object | `{}` | Annotations for resource-directory service |
| resourcedirectory.service.crt.extraDnsNames | list | `[]` | Extra DNS names for service certificate |
| resourcedirectory.service.labels | object | `{}` | Labels for resource-directory service |
| resourcedirectory.service.name | string | `"grpc"` | Name |
| resourcedirectory.service.protocol | string | `"TCP"` | Protocol |
| resourcedirectory.service.targetPort | string | `"grpc"` | Target port |
| resourcedirectory.service.type | string | `"ClusterIP"` | resource-directory service type |
| resourcedirectory.tolerations | object | `{}` | Toleration definition |
| scylla.datacenter | string | `"dc-1"` |  |
| scylla.enabled | bool | `false` | Enable scylla service. Required scylla operator: <https://github.com/scylladb/scylla-operator/blob/master/docs/source/generic.md#deploy-scylla-operator> |
| scylla.racks[0].members | int | `3` |  |
| scylla.racks[0].name | string | `"dc-1a"` |  |
| scylla.racks[0].resources.limits.cpu | int | `1` |  |
| scylla.racks[0].resources.limits.memory | string | `"4Gi"` |  |
| scylla.racks[0].resources.requests.cpu | int | `1` |  |
| scylla.racks[0].resources.requests.memory | string | `"4Gi"` |  |
| scylla.racks[0].scyllaConfig | string | `"scylla-cfg"` |  |
| scylla.racks[0].storage.capacity | string | `"10Gi"` |  |
| scylla.racks[0].volumeMounts[0].mountPath | string | `"/certs"` |  |
| scylla.racks[0].volumeMounts[0].name | string | `"scylla-certs-volume"` |  |
| scylla.racks[0].volumes[0].name | string | `"scylla-certs-volume"` |  |
| scylla.racks[0].volumes[0].secret.secretName | string | `"scylla-dc-1a-crt"` |  |
| scylla.scyllaImage.tag | string | `"5.2.9"` |  |
| scylla.sysctls[0] | string | `"fs.aio-max-nr=2097152"` |  |
| snippetservice.affinity | string | `nil` | Affinity definition |
| snippetservice.apis | object | `{"grpc":{"address":null,"authorization":{"audience":null,"authority":null,"http":{"idleConnTimeout":"30s","maxConnsPerHost":32,"maxIdleConns":16,"maxIdleConnsPerHost":16,"timeout":"10s","tls":{"caPool":null,"certFile":null,"keyFile":null,"useSystemCAPool":true}},"ownerClaim":null},"enforcementPolicy":{"minTime":"5s","permitWithoutStream":true},"keepAlive":{"maxConnectionAge":"0s","maxConnectionAgeGrace":"0s","maxConnectionIdle":"0s","time":"2h","timeout":"20s"},"recvMsgSize":4194304,"sendMsgSize":4194304,"tls":{"caPool":null,"certFile":null,"clientCertificateRequired":false,"keyFile":null}},"http":{"address":null,"idleTimeout":"30s","readHeaderTimeout":"4s","readTimeout":"8s","writeTimeout":"16s"}}` | For complete snippet-service configuration see [plgd/snippet-service](https://github.com/plgd-dev/hub/tree/main/snippet-service) |
| snippetservice.clients.eventBus.nats.pendingLimits.bytesLimit | string | `"67108864"` |  |
| snippetservice.clients.eventBus.nats.pendingLimits.msgLimit | string | `"524288"` |  |
| snippetservice.clients.eventBus.nats.tls.caPool | string | `nil` |  |
| snippetservice.clients.eventBus.nats.tls.certFile | string | `nil` |  |
| snippetservice.clients.eventBus.nats.tls.keyFile | string | `nil` |  |
| snippetservice.clients.eventBus.nats.tls.useSystemCAPool | bool | `false` |  |
| snippetservice.clients.eventBus.nats.url | string | `""` |  |
| snippetservice.clients.eventBus.subscriptionID | string | `"snippet-service"` |  |
| snippetservice.clients.resourceAggregate.grpc.address | string | `""` |  |
| snippetservice.clients.resourceAggregate.grpc.keepAlive.permitWithoutStream | bool | `true` |  |
| snippetservice.clients.resourceAggregate.grpc.keepAlive.time | string | `"10s"` |  |
| snippetservice.clients.resourceAggregate.grpc.keepAlive.timeout | string | `"20s"` |  |
| snippetservice.clients.resourceAggregate.grpc.recvMsgSize | int | `4194304` |  |
| snippetservice.clients.resourceAggregate.grpc.sendMsgSize | int | `4194304` |  |
| snippetservice.clients.resourceAggregate.grpc.tls.caPool | string | `nil` |  |
| snippetservice.clients.resourceAggregate.grpc.tls.certFile | string | `nil` |  |
| snippetservice.clients.resourceAggregate.grpc.tls.keyFile | string | `nil` |  |
| snippetservice.clients.resourceAggregate.grpc.tls.useSystemCAPool | bool | `false` |  |
| snippetservice.clients.storage.cleanUpExpiredUpdates | string | `"0 * * * *"` |  |
| snippetservice.clients.storage.mongoDB.database | string | `"snippetService"` |  |
| snippetservice.clients.storage.mongoDB.maxConnIdleTime | string | `"4m0s"` |  |
| snippetservice.clients.storage.mongoDB.maxPoolSize | int | `16` |  |
| snippetservice.clients.storage.mongoDB.tls.caPool | string | `nil` |  |
| snippetservice.clients.storage.mongoDB.tls.certFile | string | `nil` |  |
| snippetservice.clients.storage.mongoDB.tls.keyFile | string | `nil` |  |
| snippetservice.clients.storage.mongoDB.tls.useSystemCAPool | bool | `false` |  |
| snippetservice.clients.storage.mongoDB.uri | string | `nil` |  |
| snippetservice.clients.storage.use | string | `"mongoDB"` |  |
| snippetservice.config | object | `{"fileName":"service.yaml","mountPath":"/config","volume":"config"}` | Service configuration |
| snippetservice.config.fileName | string | `"service.yaml"` | File name for config file |
| snippetservice.config.mountPath | string | `"/config"` | Mount path |
| snippetservice.config.volume | string | `"config"` | Config file volume name |
| snippetservice.deploymentAnnotations | object | `{}` | Additional annotations for snippet-service deployment |
| snippetservice.deploymentLabels | object | `{}` | Additional labels for snippet-service deployment |
| snippetservice.domain | string | `nil` | External domain for snippet-service. Default: api.{{ global.domain }} |
| snippetservice.enabled | bool | `true` | Enable snippet-service |
| snippetservice.extraContainers | object | `{}` | Extra POD containers |
| snippetservice.extraVolumeMounts | string | `nil` | Optional extra volume mounts |
| snippetservice.extraVolumes | string | `nil` | Optional extra volumes |
| snippetservice.fullnameOverride | string | `nil` | Full name to override |
| snippetservice.httpPort | int | `9101` |  |
| snippetservice.hubId | string | `nil` | Hub ID. Overrides the global.hubId |
| snippetservice.image.imagePullSecrets | string | `nil` | Image pull secrets |
| snippetservice.image.pullPolicy | string | `"Always"` | Image pull policy |
| snippetservice.image.registry | string | `"ghcr.io/"` | Image registry |
| snippetservice.image.repository | string | `"plgd-dev/hub/snippet-service"` | Image repository |
| snippetservice.image.tag | string | `nil` | Image tag. |
| snippetservice.imagePullSecrets | string | `nil` | Image pull secrets |
| snippetservice.ingress.grpc.annotations | object | `{"cert-manager.io/private-key-rotation-policy":"always","ingress.kubernetes.io/force-ssl-redirect":"true","nginx.ingress.kubernetes.io/backend-protocol":"GRPCS","nginx.ingress.kubernetes.io/enable-cors":"true","nginx.org/grpc-services":"{{ include \"plgd-hub.snippetservice.fullname\" . }}-grpc"}` | Pre defined map of Ingress annotation |
| snippetservice.ingress.grpc.customAnnotations | object | `{}` | Custom map of Ingress annotation |
| snippetservice.ingress.grpc.enabled | bool | `true` | Enable ingress |
| snippetservice.ingress.grpc.paths | list | `["/snippetservice.pb.SnippetService"]` | Paths |
| snippetservice.ingress.grpc.secretName | string | `nil` | Override name of host/tls secret. If not specified, it will be generated |
| snippetservice.ingress.http.annotations | object | `{"cert-manager.io/private-key-rotation-policy":"always","ingress.kubernetes.io/force-ssl-redirect":"true","nginx.ingress.kubernetes.io/backend-protocol":"HTTPS","nginx.ingress.kubernetes.io/enable-cors":"true","nginx.org/grpc-services":"{{ include \"plgd-hub.snippetservice.fullname\" . }}-http"}` | Pre defined map of Ingress annotation |
| snippetservice.ingress.http.customAnnotations | object | `{}` | Custom map of Ingress annotation |
| snippetservice.ingress.http.enabled | bool | `true` | Enable ingress |
| snippetservice.ingress.http.paths | list | `["/snippet-service"]` | Ingress path |
| snippetservice.ingress.http.secretName | string | `nil` | Override name of host/tls secret. If not specified, it will be generated |
| snippetservice.initContainersTpl | string | `nil` | Init containers definition |
| snippetservice.livenessProbe | string | `nil` | Liveness probe. snippet-service doesn't have any default liveness probe |
| snippetservice.log | object | `{"dumpBody":false,"encoderConfig":{"timeEncoder":"rfc3339nano"},"encoding":"json","level":"info","stacktrace":{"enabled":false,"level":"warn"}}` | Log section |
| snippetservice.log.dumpBody | bool | `false` | Dump grpc messages |
| snippetservice.log.encoderConfig.timeEncoder | string | `"rfc3339nano"` | Time format for logs. The supported values are: "rfc3339nano", "rfc3339" |
| snippetservice.log.encoding | string | `"json"` | The supported values are: "json", "console" |
| snippetservice.log.level | string | `"info"` | Logging enabled from level |
| snippetservice.log.stacktrace.enabled | bool | `false` | Log stacktrace |
| snippetservice.log.stacktrace.level | string | `"warn"` | Stacktrace from level |
| snippetservice.name | string | `"snippet-service"` | Name of component. Used in label selectors |
| snippetservice.nodeSelector | string | `nil` | Node selector |
| snippetservice.podAnnotations | object | `{}` | Annotations for snippet-service pod |
| snippetservice.podLabels | object | `{}` | Labels for snippet-service pod |
| snippetservice.podSecurityContext | object | `{}` | Pod security context |
| snippetservice.port | int | `9100` | Service and POD port |
| snippetservice.rbac | object | `{"enabled":false,"roleBindingDefitionTpl":null,"serviceAccountName":"snippet-service"}` | RBAC configuration |
| snippetservice.rbac.enabled | bool | `false` | Enable RBAC |
| snippetservice.rbac.roleBindingDefitionTpl | string | `nil` | Template definition for Role/binding etc.. |
| snippetservice.rbac.serviceAccountName | string | `"snippet-service"` | Name of snippet-service SA |
| snippetservice.readinessProbe | string | `nil` | Readiness probe. snippet-service doesn't have aby default readiness probe |
| snippetservice.replicas | int | `1` | Number of replicas |
| snippetservice.resources | string | `nil` | Resources limit |
| snippetservice.restartPolicy | string | `"Always"` | Restart policy for pod |
| snippetservice.securityContext | string | `nil` | Security context for pod |
| snippetservice.service.grpc.annotations | object | `{}` | Annotations for snippet-service |
| snippetservice.service.grpc.crt.extraDnsNames | list | `[]` | Extra DNS names for service certificate |
| snippetservice.service.grpc.labels | object | `{}` | Labels for snippet-service |
| snippetservice.service.grpc.name | string | `"grpc"` | Name |
| snippetservice.service.grpc.protocol | string | `"TCP"` | Protocol |
| snippetservice.service.grpc.targetPort | string | `"grpc"` | Target port |
| snippetservice.service.grpc.type | string | `"ClusterIP"` | Service type |
| snippetservice.service.http.annotations | object | `{}` | Annotations for snippet-service |
| snippetservice.service.http.crt.extraDnsNames | list | `[]` | Extra DNS names for service certificate |
| snippetservice.service.http.labels | object | `{}` | Labels for snippet-service |
| snippetservice.service.http.name | string | `"http"` | Name |
| snippetservice.service.http.protocol | string | `"TCP"` | Protocol |
| snippetservice.service.http.targetPort | string | `"http"` | Target port |
| snippetservice.service.http.type | string | `"ClusterIP"` | Service type |
| snippetservice.tolerations | string | `nil` | Toleration definition |

----------------------------------------------
Autogenerated from chart metadata using [helm-docs v1.14.2](https://github.com/norwoodj/helm-docs/releases/v1.14.2)
