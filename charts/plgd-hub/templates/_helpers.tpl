{{/*
Expand the name of the chart.
*/}}
{{- define "plgd-hub.name" -}}
{{- default .Chart.Name .Values.nameOverride | replace "+" "_" | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Create a default fully qualified app name.
We truncate at 63 chars because some Kubernetes name fields are limited to this (by the DNS naming spec).
If release name contains chart name it will be used as a full name.
*/}}
{{- define "plgd-hub.fullname" -}}
{{- if .Values.fullnameOverride }}
{{- .Values.fullnameOverride | trunc 63 | trimSuffix "-" }}
{{- else }}
{{- $name := default .Chart.Name .Values.nameOverride }}
{{- if contains $name .Release.Name }}
{{- .Release.Name | trunc 63 | trimSuffix "-" }}
{{- else }}
{{- printf "%s-%s" .Release.Name $name | trunc 63 | trimSuffix "-" }}
{{- end }}
{{- end }}
{{- end }}

{{- define "plgd-hub.identitystore.fullname" -}}
{{- if .Values.identitystore.fullnameOverride }}
{{- .Values.identitystore.fullnameOverride | trunc 63 | trimSuffix "-" }}
{{- else }}
{{- $name := default .Chart.Name .Values.nameOverride }}
{{- if contains $name .Release.Name }}
{{- .Values.identitystore.name | trunc 63 | trimSuffix "-" }}
{{- else }}
{{- printf "%s-%s-%s" .Release.Name $name .Values.identitystore.name | trunc 63 | trimSuffix "-" }}
{{- end }}
{{- end }}
{{- end }}


{{- define "plgd-hub.coapgateway.fullname" -}}
{{- if .Values.coapgateway.fullnameOverride }}
{{- .Values.coapgateway.fullnameOverride | trunc 63 | trimSuffix "-" }}
{{- else }}
{{- $name := default .Chart.Name .Values.nameOverride }}
{{- if contains $name .Release.Name }}
{{- .Values.coapgateway.name | trunc 63 | trimSuffix "-" }}
{{- else }}
{{- printf "%s-%s-%s" .Release.Name $name .Values.coapgateway.name | trunc 63 | trimSuffix "-" }}
{{- end }}
{{- end }}
{{- end }}

{{- define "plgd-hub.resourceaggregate.fullname" -}}
{{- if .Values.resourceaggregate.fullnameOverride }}
{{- .Values.resourceaggregate.fullnameOverride | trunc 63 | trimSuffix "-" }}
{{- else }}
{{- $name := default .Chart.Name .Values.nameOverride }}
{{- if contains $name .Release.Name }}
{{- .Values.resourceaggregate.name | trunc 63 | trimSuffix "-" }}
{{- else }}
{{- printf "%s-%s-%s" .Release.Name $name .Values.resourceaggregate.name | trunc 63 | trimSuffix "-" }}
{{- end }}
{{- end }}
{{- end }}

{{- define "plgd-hub.resourcedirectory.fullname" -}}
{{- if .Values.resourcedirectory.fullnameOverride }}
{{- .Values.resourcedirectory.fullnameOverride | trunc 63 | trimSuffix "-" }}
{{- else }}
{{- $name := default .Chart.Name .Values.nameOverride }}
{{- if contains $name .Release.Name }}
{{- .Values.resourcedirectory.name | trunc 63 | trimSuffix "-" }}
{{- else }}
{{- printf "%s-%s-%s" .Release.Name $name .Values.resourcedirectory.name | trunc 63 | trimSuffix "-" }}
{{- end }}
{{- end }}
{{- end }}


{{- define "plgd-hub.certificateConfigWithExtraCAPool" }}
  {{- $ := index . 0 }}
  {{- $certDefinition := index . 1 }}
  {{- $certPath := index . 2 }}
  {{- $useCAPool := index . 3 }}
  {{- $caPool := "" }}
  {{- if $certDefinition.caPool }}
  {{- $caPool = printf "%s" $certDefinition.caPool | quote }}
  {{- else if $.Values.certmanager.enabled }}
  {{- $caPool = printf "%s/ca.crt" $certPath | quote }}
  {{- end }}
  {{- $extraCAPool := include "plgd-hub.extraCAPoolConfig" (list $ $useCAPool) }}
  caPool:
    - {{ $caPool }}
    {{- if $extraCAPool }}
    {{ $extraCAPool }}
    {{- end }}
  {{- if $certDefinition.keyFile }}
  keyFile: {{ printf "%s" $certDefinition.keyFile | quote }}
  {{- else if $.Values.certmanager.enabled }}
  keyFile: {{ printf "%s/tls.key" $certPath  | quote  }}
  {{- end }}
  {{- if $certDefinition.certFile }}
  certFile: {{ printf "%s" $certDefinition.certFile | quote }}
  {{- else if $.Values.certmanager.enabled }}
  certFile: {{ printf "%s/tls.crt" $certPath | quote }}
  {{- end }}
{{- end }}

{{- define "plgd-hub.internalCertificateConfig" }}
{{- $ := index . 0 }}
{{- $certDefinition := index . 1 }}
{{- $certPath := index . 2 }}
{{- include "plgd-hub.certificateConfigWithExtraCAPool" (list $ $certDefinition $certPath $.Values.extraCAPool.internal) }}
{{- end }}

{{- define "plgd-hub.externalCertificateConfig" }}
{{- $ := index . 0 }}
{{- $certDefinition := index . 1 }}
{{- $certPath := index . 2 }}
{{- include "plgd-hub.certificateConfigWithExtraCAPool" (list $ $certDefinition $certPath $.Values.extraCAPool.external) }}
{{- end }}

{{- define "plgd-hub.internalCRLConfig" }}
{{- $ := index . 0 }}
{{- $certDefinition := index . 1 }}
{{- $certPath := index . 2 }}
{{- include "plgd-hub.certificateConfigWithExtraCAPool" (list $ $certDefinition $certPath $.Values.extraCAPool.internal) }}
{{- end }}

{{- define "plgd-hub.coapCertificateConfig" }}
{{- $ := index . 0 }}
{{- $certDefinition := index . 1 }}
{{- $certPath := index . 2 }}
{{- include "plgd-hub.certificateConfigWithExtraCAPool" (list $ $certDefinition $certPath $.Values.extraCAPool.coap) }}
{{- end }}

{{- define "plgd-hub.storageCertificateConfig" }}
{{- $ := index . 0 }}
{{- $certDefinition := index . 1 }}
{{- $certPath := index . 2 }}
{{- include "plgd-hub.certificateConfigWithExtraCAPool" (list $ $certDefinition $certPath $.Values.extraCAPool.storage) }}
{{- end }}

{{- define "plgd-hub.certificateConfig" }}
  {{- $ := index . 0 }}
  {{- $certDefinition := index . 1 }}
  {{- $certPath := index . 2 }}
  {{- if $certDefinition.caPool }}
  caPool: {{ printf "%s" $certDefinition.caPool | quote }}
  {{- else if $.Values.certmanager.enabled }}
  caPool: {{ printf "%s/ca.crt" $certPath | quote  }}
  {{- end }}
  {{- if $certDefinition.keyFile }}
  keyFile: {{ printf "%s" $certDefinition.keyFile | quote }}
  {{- else if $.Values.certmanager.enabled }}
  keyFile: {{ printf "%s/tls.key" $certPath  | quote  }}
  {{- end }}
  {{- if $certDefinition.certFile }}
  certFile: {{ printf "%s" $certDefinition.certFile | quote }}
  {{- else if $.Values.certmanager.enabled }}
  certFile: {{ printf "%s/tls.crt" $certPath | quote }}
  {{- end }}
{{- end }}

{{- define "plgd-hub.authorizationCaCertificateConfig" }}
{{- $ := index . 0 }}
{{- $certDefinition := index . 1 }}
{{- $certPath := index . 2 }}
{{- include "plgd-hub.certificateConfigWithExtraCAPool" (list $ $certDefinition $certPath $.Values.extraCAPool.authorization) }}
{{- end }}

{{- define "plgd-hub.httpConfig" }}
{{- $ := index . 0 }}
{{- $http := index . 1 }}
{{- $certPath := index . 2 }}
maxIdleConns: {{ $http.maxIdleConns }}
maxConnsPerHost:  {{ $http.maxConnsPerHost }}
maxIdleConnsPerHost:  {{ $http.maxIdleConnsPerHost }}
idleConnTimeout:  {{ $http.idleConnTimeout }}
timeout: {{ $http.timeout }}
tls:
  {{- $httpTls := $http.tls }}
  {{- include "plgd-hub.authorizationCaCertificateConfig" (list $ $httpTls $certPath ) }}
  useSystemCAPool: {{ $http.tls.useSystemCAPool }}
  {{- if $httpTls.crl }}
  {{- include "plgd-hub.crlAuthorizationConfig" (list $ $httpTls.crl ) | indent 2 }}
  {{- end }}
{{- end }}

{{- define "plgd-hub.CRLConfig" }}
{{- $ := index . 0 }}
{{- $crl := index . 1 }}
{{- if include "plgd-hub.resolveTemplateString" (list $ $crl.enabled) -}}
enabled: true
http:
  maxIdleConns: {{ $crl.http.maxIdleConns }}
  maxConnsPerHost:  {{ $crl.http.maxConnsPerHost }}
  maxIdleConnsPerHost:  {{ $crl.http.maxIdleConnsPerHost }}
  idleConnTimeout:  {{ $crl.http.idleConnTimeout }}
  timeout: {{ $crl.http.timeout }}
  tls:
    {{- $caPoolKey := include "plgd-hub.resolveTemplateString" (list $ $crl.http.tls.caPoolKey) }}
    {{- if $caPoolKey }}
    caPool: {{ printf "%s/%s" $crl.mountPath $caPoolKey | quote }}
    {{- end }}
    {{- $keyKey := include "plgd-hub.resolveTemplateString" (list $ $crl.http.tls.keyKey) }}
    {{- if $keyKey }}
    keyFile: {{ printf "%s/%s" $crl.mountPath $keyKey| quote }}
    {{- end }}
    {{- $crtKey := include "plgd-hub.resolveTemplateString" (list $ $crl.http.tls.crtKey) }}
    {{- if $crtKey }}
    certFile: {{ printf "%s/%s" $crl.mountPath $crtKey | quote }}
    {{- end }}
    useSystemCAPool: {{ $crl.http.tls.useSystemCAPool }}
{{- else }}
enabled: false
{{- end }}
{{- end }}

{{- define "plgd-hub.crlAuthorizationConfig" }}
{{- $ := index . 0 }}
{{- $crl := index . 1 }}
{{- $crlCfg := dict }}
{{- if $crl.enabled }}
{{- $crlCfg = $crl }}
{{- else if include "plgd-hub.crlAuthorizationEnabled" $ }}
{{- $crlCfg = include "plgd-hub.CRLConfig" (list $ $.Values.crl.authorization) | fromYaml }}
{{- end }}
crl:
{{- if $crlCfg.enabled }}
{{ $crlCfg | toYaml | indent 2 }}
{{- else  }}
  enabled: false
{{- end }}
{{- end }}

{{- define "plgd-hub.CRLConfigFromCertificateAuthority" }}
{{- $ := . }}
enabled: {{and $.Values.certificateauthority.enabled $.Values.certificateauthority.signer.crl.enabled }}
http:
  maxIdleConns: {{ default 16 $.Values.crl.coap.http.maxIdleConns }}
  maxConnsPerHost: {{ default 32 $.Values.crl.coap.http.maxConnsPerHost }}
  maxIdleConnsPerHost: {{ default 16 $.Values.crl.coap.http.maxIdleConnsPerHost }}
  idleConnTimeout: {{ default "30s" $.Values.crl.coap.http.idleConnTimeout }}
  timeout: {{ default "10s" $.Values.crl.coap.http.timeout }}
  tls:
    {{- $coap := $.Values.coapgateway }}
    {{- $certPath := "/certs/client" }}
    {{- $caClientTls := $coap.clients.certificateAuthority.grpc.tls }}
    {{- include "plgd-hub.externalCertificateConfig" (list $ $caClientTls $certPath) | indent 2 }}
    useSystemCAPool: true
{{- end }}

{{- define "plgd-hub.crlCoapConfig" }}
{{- $ := index . 0 }}
{{- $crl := index . 1 }}
{{- $crlCfg := dict }}
{{- if $crl.enabled }}
{{- $crlCfg = $crl }}
{{- else if include "plgd-hub.crlCoapEnabled" $ }}
{{- $crlCfg = include "plgd-hub.CRLConfig" (list $ $.Values.crl.coap) | fromYaml }}
{{- end }}
{{- if and (not $crlCfg.enabled) $.Values.certificateauthority.enabled $.Values.certificateauthority.signer.crl.enabled }}
{{- $crlCfg = include "plgd-hub.CRLConfigFromCertificateAuthority" $ | fromYaml }}
{{- end }}
crl:
{{- if $crlCfg.enabled }}
{{ $crlCfg | toYaml | indent 2 }}
{{- else  }}
  enabled: false
{{- end }}
{{- end }}

{{- define "plgd-hub.crlInternalConfig" }}
{{- $ := index . 0 }}
{{- $crl := index . 1 }}
{{- $crlCfg := dict }}
{{- if $crl.enabled }}
{{- $crlCfg = $crl }}
{{- else if include "plgd-hub.crlInternalEnabled" $ }}
{{- $crlCfg = include "plgd-hub.CRLConfig" (list $ $.Values.crl.internal) | fromYaml }}
{{- end }}
crl:
{{- if $crlCfg.enabled }}
{{ $crlCfg | toYaml | indent 2 }}
{{- else  }}
  enabled: false
{{- end }}
{{- end }}

{{- define "plgd-hub.authorizationFilterEndpoints" }}
  {{- $ := index . 0 }}
  {{- $endpoints := index . 1 }}
  {{- $result := list}}
  {{- range $endpoints }}
  {{- $authority := include "plgd-hub.resolveTemplateString" (list $ .authority) }}
  {{- if $authority }}
  {{- $result = append $result . }}
  {{- end }}
  {{- end }}
  {{- dict "Values" $result | toYaml }}
{{- end }}

{{- define "plgd-hub.basicAuthorizationConfig" }}
  {{- $ := index . 0 }}
  {{- $authorization := index . 1 }}
  {{- $prefix := index . 2 }}
  {{- $certPath := index . 3 }}
  {{- $endpoints := list}}
  {{- $audience := ""}}
  {{- if $authorization }}
  {{- if $authorization.audience }}
  {{- $audience = $authorization.audience }}
  {{- end }}
  {{- if $authorization.endpoints }}
  {{- if gt (len $authorization.endpoints) 0 }}
  {{- $endpoints = $authorization.endpoints }}
  {{- end }}
  {{- end }}
  {{- end }}
  {{- if not $audience }}
  {{- $audience = $.Values.global.audience }}
  {{- end }}
  {{- if eq (len $endpoints) 0 }}
  {{- $endpoints = $.Values.global.authorization.endpoints }}
  {{- end }}
  {{- $mapEndpoints := include "plgd-hub.authorizationFilterEndpoints" (list $ $endpoints) | fromYaml }}
  {{- if eq (len $mapEndpoints.Values) 0 }}
  {{- fail (printf "%s.endpoints or global.authorization.endpoints is required" $prefix) }}
  {{- end }}
  audience: {{ include "plgd-hub.resolveTemplateString" (list $ $audience) }}
  endpoints:
    {{- range $mapEndpoints.Values }}
    {{- $authority := include "plgd-hub.resolveTemplateString" (list $ .authority) }}
    {{- if $authority }}
    - authority: {{ include "plgd-hub.resolveTemplateString" (list $ .authority) }}
      http:
        {{- include "plgd-hub.httpConfig" (list $ .http $certPath ) | indent 8 }}
    {{- end }}
    {{- end }}
  tokenTrustVerification:
    {{- $tokenTrustVerification := $authorization.tokenTrustVerification }}
    {{- if not $tokenTrustVerification }}
    {{- $tokenTrustVerification = $.Values.global.authorization.tokenTrustVerification }}
    {{- end }}
    {{- $cacheExpiration := "30s" }}
    {{- if $tokenTrustVerification }}
    {{- if $tokenTrustVerification.cacheExpiration }}
    {{- $cacheExpiration = $tokenTrustVerification.cacheExpiration }}
    {{- end }}
    {{- end }}
    cacheExpiration: {{ $cacheExpiration }}
{{- end }}

{{- define "plgd-hub.authorizationConfig" }}
  {{- $ := index . 0 }}
  {{- $authorization := index . 1 }}
  {{- $prefix := index . 2 }}
  {{- $certPath := index . 3 }}
  ownerClaim: {{ required (printf "%s.authorization.ownerClaim or global.ownerClaim is required " $prefix) ( $authorization.ownerClaim | default $.Values.global.ownerClaim ) | quote }}
  {{- include "plgd-hub.basicAuthorizationConfig" (list $ $authorization $prefix $certPath) }}
{{- end }}

{{- define "plgd-hub.createInternalCertByCm" }}
    {{- $natsTls := .Values.coapgateway.clients.eventBus.nats.tls.certFile }}
    {{- $authClientTls := .Values.coapgateway.clients.identityStore.grpc.tls.certFile }}
    {{- $raClientTls := .Values.coapgateway.clients.resourceAggregate.grpc.tls.certFile }}
    {{- $rdClientTls := .Values.coapgateway.clients.resourceDirectory.grpc.tls.certFile }}
    {{- if and $natsTls $authClientTls $raClientTls $rdClientTls }}
    {{- printf "" -}}
    {{- else }}
    {{- printf "true" -}}
    {{- end }}
{{- end }}

{{- define "plgd-hub.certmanager.coapIssuerName" -}}
    {{- if .Values.certmanager.coap.issuer.name }}
        {{- printf "%s" .Values.certmanager.coap.issuer.name }}
    {{- end }}
    {{- printf "%s" .Values.certmanager.default.issuer.name }}
{{- end }}

{{- define "plgd-hub.certmanager.coapIssuerKind" -}}
    {{- if .Values.certmanager.coap.issuer.kind }}
        {{- printf "%s" .Values.certmanager.coap.issuer.kind }}
    {{- end }}
    {{- printf "%s" .Values.certmanager.default.issuer.kind }}
{{- end }}


{{- define "plgd-hub.certmanager.internalIssuerName" -}}
    {{- if .Values.certmanager.internal.issuer.name }}
        {{- printf "%s" .Values.certmanager.internal.issuer.name }}
    {{- end }}
    {{- printf "%s" .Values.certmanager.default.issuer.name }}
{{- end }}

{{- define "plgd-hub.certmanager.internalIssuerKind" -}}
    {{- if .Values.certmanager.internal.issuer.kind }}
        {{- printf "%s" .Values.certmanager.internal.issuer.kind }}
    {{- end }}
    {{- printf "%s" .Values.certmanager.default.issuer.kind }}
{{- end }}

{{- define "plgd-hub.chart" -}}
{{- printf "%s-%s" .Chart.Name .Chart.Version | replace "+" "_" | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Common labels
*/}}
{{- define "plgd-hub.labels" -}}
helm.sh/chart: {{ include "plgd-hub.chart" . }}
{{ include "plgd-hub.selectorLabels" . }}
{{- if .Chart.AppVersion }}
app.kubernetes.io/version: {{ .Chart.AppVersion | quote }}
{{- end }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
{{- end }}

{{/*
Selector labels
*/}}
{{- define "plgd-hub.selectorLabels" -}}
app.kubernetes.io/name: {{ include "plgd-hub.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
{{- end }}

{{- define "plgd-hub.coapgateway.selectorLabels" -}}
app.kubernetes.io/name: {{ .Values.coapgateway.name }}
app.kubernetes.io/instance: {{ .Release.Name }}
{{- end }}

{{- define "plgd-hub.resourceaggregate.selectorLabels" -}}
app.kubernetes.io/name: {{ .Values.resourceaggregate.name }}
app.kubernetes.io/instance: {{ .Release.Name }}
{{- end }}

{{- define "plgd-hub.natsUri" }}
  {{- $ := index . 0 }}
  {{- $natsUri := index . 1 }}
  {{- if $.Values.global.natsUri }}
  {{- printf "%s" $.Values.global.natsUri }}
  {{- else if $natsUri }}
  {{- printf "%s" $natsUri }}
  {{- else }}
  {{- printf "nats://%s-nats.%s.svc.%s:4222" $.Release.Name $.Release.Namespace $.Values.cluster.dns }}
  {{- end }}
{{- end }}

{{- define "plgd-hub.mongoDBUri" }}
  {{- $ := index . 0 }}
  {{- $mongoUri := index . 1 }}
  {{- if $.Values.global.mongoUri }}
  {{- printf "%s" $.Values.global.mongoUri }}
  {{- else if $mongoUri }}
  {{- printf "%s" $mongoUri }}
  {{- else }}
  {{- printf "mongodb://mongodb-headless.%s.svc.%s:27017/?replicaSet=%s" $.Release.Namespace $.Values.cluster.dns $.Values.mongodb.replicaSetName }}
  {{- end }}
{{- end }}

{{- define "plgd-hub.cqlDBHosts" }}
  {{- $ := index . 0 }}
  {{- $cqlDBHosts := index . 1 }}
  {{- if $.Values.global.cqlDBHosts }}
  {{- range $.Values.global.cqlDBHosts }}
  - {{- printf " %s" . }}
  {{- end }}
  {{- else if $cqlDBHosts }}
  {{- range $cqlDBHosts }}
  - {{- printf " %s" . }}
  {{- end }}
  {{- else }}
  - {{ printf "%s-scylla-client.%s.svc.%s" $.Release.Name $.Release.Namespace $.Values.cluster.dns }}
  {{- end }}
{{- end }}

{{- define "plgd-hub.identityStoreAddress" }}
  {{- $ := index . 0 }}
  {{- $address := index . 1 }}
  {{- if $address }}
  {{- printf "%s" $address }}
  {{- else }}
  {{- $authorizationServer := include "plgd-hub.identitystore.fullname" $ }}
  {{- printf "%s.%s.svc.%s:%v" $authorizationServer $.Release.Namespace $.Values.cluster.dns $.Values.identitystore.port }}
  {{- end }}
{{- end }}

{{- define "plgd-hub.resourceDirectoryAddress" }}
  {{- $ := index . 0 }}
  {{- $address := index . 1 }}
  {{- if $address }}
  {{- printf "%s" $address }}
  {{- else }}
  {{- $rdServer := include "plgd-hub.resourcedirectory.fullname" $ }}
  {{- printf "%s.%s.svc.%s:%v" $rdServer $.Release.Namespace $.Values.cluster.dns $.Values.resourcedirectory.port }}
  {{- end }}
{{- end }}

{{- define "plgd-hub.resourceAggregateAddress" }}
  {{- $ := index . 0 }}
  {{- $address := index . 1 }}
  {{- if $address }}
  {{- printf "%s" $address }}
  {{- else }}
  {{- $raServer := include "plgd-hub.resourceaggregate.fullname" $ }}
  {{- printf "%s.%s.svc.%s:%v" $raServer $.Release.Namespace $.Values.cluster.dns $.Values.resourcedirectory.port }}
  {{- end }}
{{- end }}

{{- define "plgd-hub.certificateAuthorityAddress" }}
  {{- $ := index . 0 }}
  {{- $address := index . 1 }}
  {{- if $address }}
  {{- printf "%s" $address }}
  {{- else }}
  {{- $raServer := include "plgd-hub.certificateauthority.fullname" $ }}
  {{- printf "%s-grpc.%s.svc.%s:%v" $raServer $.Release.Namespace $.Values.cluster.dns $.Values.certificateauthority.port }}
  {{- end }}
{{- end }}

{{- define "plgd-hub.grpcGatewayAddress" }}
  {{- $ := index . 0 }}
  {{- $address := index . 1 }}
  {{- if $address }}
  {{- printf "%s" $address }}
  {{- else }}
  {{- $raServer := include "plgd-hub.grpcgateway.fullname" $ }}
  {{- printf "%s.%s.svc.%s:%v" $raServer $.Release.Namespace $.Values.cluster.dns $.Values.grpcgateway.port }}
  {{- end }}
{{- end }}

{{- define  "plgd-hub.globalDomain" }}
  {{- $ := index . 0 }}
  {{- $address := index . 1 }}
  {{- $subDomainPrefix := index . 2 }}
  {{- if $address }}
  {{- printf "%s" $address }}
  {{- else }}
  {{- printf "%s%s" $subDomainPrefix $.Values.global.domain }}
  {{- end }}
{{- end }}

{{- define  "plgd-hub.oauthSecretFile" }}
  {{- $ := index . 0 }}
  {{- $provider := index . 1 }}
  {{- if $provider.clientSecret }}
  {{- printf "/secrets/%s/client-secret" $provider.name }}
  {{- else if $provider.clientSecretFile }}
  {{- printf "%s" $provider.clientSecretFile }}
  {{- else }}
  {{ required "clientSecret or clientSecretFile for oauth provider is required " ( $provider.clientSecret | default $provider.clientSecretFile ) }}
  {{- end }}
{{- end }}

{{- define "plgd-hub.enableDefaultIssuer" }}
    {{- if and .Values.certmanager.enabled .Values.certmanager.default.issuer.enabled }}
        {{- $nameInternal := .Values.certmanager.internal.issuer.name }}
        {{- $kindInternal := .Values.certmanager.internal.issuer.kind }}
        {{- $specInternal := .Values.certmanager.internal.issuer.spec }}

        {{- $nameCoap := .Values.certmanager.coap.issuer.name }}
        {{- $kindCoap := .Values.certmanager.coap.issuer.kind }}
        {{- $specCoap := .Values.certmanager.coap.issuer.spec }}

        {{- $nameExternal := .Values.certmanager.external.issuer.name }}
        {{- $kindExternal := .Values.certmanager.external.issuer.kind }}
        {{- $specExternal := .Values.certmanager.external.issuer.spec }}

        {{- $internalIssuer := or ( and $nameInternal $kindInternal ) $specInternal }}
        {{- $coapIssuer := or ( and $nameCoap $kindCoap ) $specCoap }}
        {{- $externalIssuer := or ( and $nameExternal $kindExternal ) $specExternal }}
        {{- printf "%t" ( not ( and $internalIssuer $coapIssuer $externalIssuer )) }}
    {{- else }}
        {{- printf "" -}}
    {{- end }}
{{- end }}

{{- define "plgd-hub.wildCardCertDomain" -}}
    {{- printf "*.%s" .Values.global.domain }}
{{- end }}

{{- define "plgd-hub.wildCardCertName" -}}
  {{- $fullName := include "plgd-hub.fullname" . -}}
  {{- printf "%s-wildcard-crt" $fullName -}}
{{- end }}

{{- define "plgd-hub.tplvalues.render" -}}
    {{- if typeIs "string" .value }}
        {{- tpl .value .context }}
    {{- else }}
        {{- tpl (.value | toYaml) .context }}
    {{- end }}
{{- end -}}

{{- define "plgd-hub.openTelemetryExporterConfig" -}}
{{- $ := index . 0 }}
{{- $certPath := index . 1 }}
{{- $cfg := $.Values.global.openTelemetryExporter -}}
openTelemetryCollector:
  grpc:
    enabled: {{ $cfg.enabled }}
    address: {{ $cfg.address | quote }}
    keepAlive:
      time: {{ $cfg.keepAlive.time }}
      timeout: {{ $cfg.keepAlive.timeout }}
      permitWithoutStream: {{ $cfg.keepAlive.permitWithoutStream }}
    tls:
      {{- include "plgd-hub.certificateConfig" (list $ $cfg.tls $certPath ) | indent 4 }}
      useSystemCAPool: {{ $cfg.tls.useSystemCAPool }}
{{- end -}}

{{- define "plgd-hub.useDatabase" }}
  {{- $ := index . 0 }}
  {{- $useDatabase := index . 1 }}
  {{- if $.Values.global.useDatabase }}
  {{- printf "%s" $.Values.global.useDatabase }}
  {{- else if $useDatabase }}
  {{- printf "%s" $useDatabase }}
  {{- else }}
  {{- printf "mongoDB" }}
  {{- end }}
{{- end }}

{{- define "plgd-hub.replicas" -}}
  {{- $ := index . 0 -}}
  {{- $useReplicas := index . 1 }}
  {{- if $.Values.global.standby -}}
  0
  {{- else -}}
  {{- $useReplicas -}}
  {{- end -}}
{{- end -}}

{{- define "plgd-hub.extraCAPoolAuthorizationEnabled" -}}
{{- $ := . }}
{{- if include "plgd-hub.resolveTemplateString" (list . $.Values.global.extraCAPool.authorization) -}}
true
{{- else -}}
{{- printf "" }}
{{- end -}}
{{- end -}}

{{- define "plgd-hub.extraCAPoolInternalEnabled" -}}
{{- $ := . }}
{{- if include "plgd-hub.resolveTemplateString" (list . $.Values.global.extraCAPool.internal) -}}
true
{{- else -}}
{{- printf "" }}
{{- end -}}
{{- end -}}

{{- define "plgd-hub.extraCAPoolStorageEnabled" -}}
{{- $ := . }}
{{- if include "plgd-hub.resolveTemplateString" (list . $.Values.global.extraCAPool.storage) -}}
true
{{- else -}}
{{- printf "" }}
{{- end -}}
{{- end -}}

{{- define "plgd-hub.extraCAPoolCoapEnabled" -}}
{{- $ := . }}
{{- if include "plgd-hub.resolveTemplateString" (list . $.Values.global.extraCAPool.coap) -}}
true
{{- else -}}
{{- printf "" }}
{{- end -}}
{{- end -}}

{{- define "plgd-hub.crlAuthorizationEnabled" -}}
{{- $ := . }}
{{- if include "plgd-hub.resolveTemplateString" (list . $.Values.global.crl.authorization.caPool) -}}
true
{{- else -}}
{{- printf "" }}
{{- end -}}
{{- end -}}

{{- define "plgd-hub.crlHttpTlsCaPoolKey" -}}
{{- $ := index . 0 -}}
{{- $crt := index . 1 }}
{{- if include "plgd-hub.resolveTemplateString" (list $ $crt) -}}
ca.crt
{{- end }}
{{- end -}}

{{- define "plgd-hub.crlHttpTlsCrtKey" -}}
{{- $ := index . 0 -}}
{{- $crt := index . 1 }}
{{- if include "plgd-hub.resolveTemplateString" (list $ $crt) -}}
tls.crt
{{- end }}
{{- end -}}

{{- define "plgd-hub.crlHttpTlsKeyKey" -}}
{{- $ := index . 0 -}}
{{- $crt := index . 1 }}
{{- if include "plgd-hub.resolveTemplateString" (list $ $crt) -}}
tls.key
{{- end }}
{{- end -}}

{{- define "plgd-hub.crlInternalEnabled" -}}
{{- $ := . }}
{{- if include "plgd-hub.resolveTemplateString" (list . $.Values.global.crl.internal.caPool) -}}
true
{{- else -}}
{{- printf "" }}
{{- end -}}
{{- end -}}

{{- define "plgd-hub.crlCoapEnabled" -}}
{{- $ := . }}
{{- if include "plgd-hub.resolveTemplateString" (list . $.Values.global.crl.coap.caPool) -}}
true
{{- else -}}
{{- printf "" }}
{{- end -}}
{{- end -}}

{{- define "plgd-hub.extraCAPoolVolume" -}}
{{- $ := index . 0 -}}
{{- $useCAPool := index . 1 }}
{{- with $useCAPool -}}
{{- $enabled := include "plgd-hub.resolveTemplateString" (list $ .enabled) -}}
{{- if and $enabled (or .configMapName .secretName) -}}
- name: {{ .name | quote }}
{{- if .configMapName }}
  configMap:
    name: {{ include "plgd-hub.resolveTemplateString" (list $ .configMapName) }}
{{- else if .secretName }}
  secret:
    secretName: {{ include "plgd-hub.resolveTemplateString" (list $ .secretName) }}
{{- end -}}
{{- end -}}
{{- end -}}
{{- end -}}

{{- define "plgd-hub.extraCAPoolMount" -}}
{{- $ := index . 0 -}}
{{- $useCAPool := index . 1 -}}
{{- with $useCAPool -}}
{{- $enabled := include "plgd-hub.resolveTemplateString" (list $ .enabled) -}}
{{- if and $enabled (or .configMapName .secretName) -}}
- name: {{ .name | quote }}
  mountPath: {{ .mountPath | quote }}
{{- end -}}
{{- end -}}
{{- end -}}

{{- define "plgd-hub.extraCAPoolConfig" }}
{{- $ := index . 0 -}}
{{- $useCAPool := index . 1 }}
{{- with $useCAPool -}}
{{- $enabled := include "plgd-hub.resolveTemplateString" (list $ .enabled) -}}
{{- if and $enabled (or .configMapName .secretName) -}}
{{- if .key -}}
- {{ printf "%s/%s" .mountPath (include "plgd-hub.resolveTemplateString" (list $ .key) ) | quote }}
{{- end -}}
{{- if .http }}
{{- if .http.tls }}
{{- if .http.tls.caPoolKey -}}
- {{ printf "%s/%s" .mountPath (include "plgd-hub.resolveTemplateString" (list $ .http.tls.caPoolKey) ) | quote }}
{{- end -}}
{{- end -}}
{{- end -}}
{{- end -}}
{{- end -}}
{{- end -}}


{{- define "plgd-hub.crlVolume" -}}
{{- include "plgd-hub.extraCAPoolVolume" . }}
{{- end -}}

{{- define "plgd-hub.crlMount" -}}
{{- include "plgd-hub.extraCAPoolMount" . }}
{{- end -}}

{{- define "plgd-hub.isTemplateString" }}
{{- $ret := "" }}
{{- if typeIs "string" . -}}
{{- if and (hasPrefix "{{" .) (hasSuffix "}}" .) -}}
{{- $ret = "true" }}
{{- end }}
{{- end }}
{{- printf $ret }}
{{- end }}

{{- define "plgd-hub.resolveTemplateString" }}
{{- $ := index . 0 -}}
{{- $string := index . 1 }}
{{- $ret := "" }}
{{- if include "plgd-hub.isTemplateString" $string -}}
{{- $ret = tpl $string $ -}}
{{- else -}}
{{- $ret = $string -}}
{{- end }}
{{- if $ret }}
{{- $ret }}
{{- else }}
{{- printf "" }}
{{- end }}
{{- end }}

{{- define "plgd-hub.oldExtraCAPoolAuthorizationFileName" }}
{{- $ := . -}}
{{- $fileName := "ca.crt" -}}
{{- if $.Values.extraAuthorizationCAPool -}}
{{- if $.Values.extraAuthorizationCAPool.fileName -}}
{{- $fileName = $.Values.extraAuthorizationCAPool.fileName -}}
{{- end -}}
{{- end -}}
{{- printf "%s" $fileName }}
{{- end -}}

{{- define "plgd-hub.oldExtraCAPoolAuthorizationSecretName" }}
{{- $ := . -}}
{{- $secretName := "authorization-ca-pool" -}}
{{- if $.Values.extraAuthorizationCAPool -}}
{{- if $.Values.extraAuthorizationCAPool.name -}}
{{- $secretName = $.Values.extraAuthorizationCAPool.name -}}
{{- end -}}
{{- end -}}
{{- printf "%s" $secretName }}
{{- end -}}

{{- define "plgd-hub.oldGlobalAuthorizationCAPool" }}
{{- $ := . -}}
{{- $ca := "" -}}
{{- if $.Values.global.authorizationCAPool -}}
{{- $ca = $.Values.global.authorizationCAPool -}}
{{- end -}}
{{- printf "%s" $ca }}
{{- end -}}

{{- define "plgd-hub.globalAudience" }}
{{- $ := . -}}
{{- $ca := "" -}}
{{- if $.Values.global.audience -}}
{{- $ca = $.Values.global.audience -}}
{{- end -}}
{{- printf "%s" $ca }}
{{- end -}}

{{- define "plgd-hub.globalAuthority" }}
{{- $ := . -}}
{{- $authority := $.Values.global.authority | default "" -}}
{{- if not $authority }}
{{- if $.Values.mockoauthserver.enabled }}
{{- $authority = include "plgd-hub.mockoauthserver.uri" $ }}
{{- end }}
{{- end -}}
{{- printf "%s" $authority }}
{{- end -}}

{{- define "plgd-hub.m2mOAuthServerAuthority" }}
{{- $ := . -}}
{{- $ca := "" -}}
{{- if include "plgd-hub.m2moauthserver.enabled" $ -}}
{{- $ca = include "plgd-hub.m2moauthserver.uri" $ }}
{{- end -}}
{{- printf "%s" $ca }}
{{- end -}}

{{- define  "plgd-hub.image" -}}
{{- $ := index . 0 -}}
{{- $service := index . 1 }}
{{- $registryName := $service.image.registry | default "" -}}
{{- $repositoryName := $service.image.repository -}}
{{- $tag := $service.image.tag | default $.Values.global.image.tag | default $.Chart.AppVersion | toString -}}
{{- printf "%s%s:%s" $registryName $repositoryName  $tag -}}
{{- end -}}