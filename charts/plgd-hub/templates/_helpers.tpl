{{/*
Expand the name of the chart.
*/}}
{{- define "plgd-hub.name" -}}
{{- default .Chart.Name .Values.nameOverride | trunc 63 | trimSuffix "-" }}
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

{{- define "plgd-hub.certificateConfig" }}
  {{- $ := index . 0 }}
  {{- $certDefinition := index . 1 }}
  {{- $certPath := index . 2 }}
  {{- if $certDefinition.caPool }}
  caPool:{{- printf " " }}{{- printf "%s" $certDefinition.caPool | quote }}
  {{- else if $.Values.certmanager.enabled }}
  caPool:{{- printf " " }}{{- printf "%s/ca.crt" $certPath | quote  }}
  {{- end }}
  {{- if $certDefinition.keyFile }}
  keyFile:{{- printf " " }}{{- printf "%s" $certDefinition.keyFile | quote }}
  {{- else if $.Values.certmanager.enabled }}
  keyFile:{{- printf " " }}{{- printf "%s/tls.key" $certPath  | quote  }}
  {{- end }}
  {{- if $certDefinition.certFile }}
  certFile:{{- printf " " }}{{- printf "%s" $certDefinition.certFile | quote }}
  {{- else if $.Values.certmanager.enabled }}
  certFile:{{- printf " " }}{{- printf "%s/tls.crt" $certPath | quote }}
  {{- end }}
{{- end }}

{{- define "plgd-hub.authorizationConfig" }}
  {{- $ := index . 0 }}
  {{- $authoriztion := index . 1 }}
  {{- $prefix := index . 2 }}
  ownerClaim:{{ printf " " }}{{ required (printf "%s.apis.grpc.authorization.ownerClaim or global.ownerClaim is required " $prefix) ( $authoriztion.ownerClaim | default $.Values.global.ownerClaim ) | quote }}
  {{- if not $.Values.mockoauthserver.enabled }}
  authority:{{ printf " " }}{{ required (printf "%s.apis.grpc.authorization.authority or global.authority is required " $prefix) ( $authoriztion.authority | default $.Values.global.authority ) | quote }}
  audience:{{ printf " " }}{{ ( $authoriztion.audience | default $.Values.global.audience ) | quote }}
  {{- else }}
  authority:{{ printf " " }}{{ include "plgd-hub.mockoauthserver.uri" $ }}
  audience:{{ printf " " }}{{ printf "" | quote }}
  {{- end }}
{{- end }}

{{- define "plgd-hub.baseAthorizationConfig" }}
  {{- $ := index . 0 }}
  {{- $authoriztion := index . 1 }}
  {{- $prefix := index . 2 }}
  {{- if not $.Values.mockoauthserver.enabled }}
  authority:{{ printf " " }}{{ required (printf "%s.apis.grpc.authorization.authority or global.authority is required " $prefix) ( $authoriztion.authority | default $.Values.global.authority ) | quote }}
  audience:{{ printf " " }}{{ ( $authoriztion.audience | default $.Values.global.audience ) | quote }}
  {{- else }}
  authority:{{ printf " " }}{{ include "plgd-hub.mockoauthserver.uri" $ }}
  audience:{{ printf " " }}{{ printf "" | quote }}
  {{- end }}
{{- end }}



{{- define "plgd-hub.createInternalCertByCm" }}
    {{- $natsTls := .Values.coapgateway.clients.eventBus.nats.tls.certFile }}
    {{- $authClientTls := .Values.coapgateway.clients.identityStore.grpc.tls.certFile }}
    {{- $raClientTls := .Values.coapgateway.clients.resourceAggregate.grpc.tls.certFile }}
    {{- $rdClientTls := .Values.coapgateway.clients.resourceDirectory.grpc.tls.certFile }}
    {{- if and $natsTls $authClientTls $raClientTls $rdClientTls }}
    {{- printf "false" }}
    {{- else }}
    {{- printf "true" }}
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
  {{- printf "mongodb://mongodb-0.mongodb-headless.%s.svc.%s:27017" $.Release.Namespace $.Values.cluster.dns }}
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
        {{- printf "false" }}
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