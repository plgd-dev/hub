{{/*
Expand the name of the chart.
*/}}
{{- define "plgd-cloud.name" -}}
{{- default .Chart.Name .Values.nameOverride | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Create a default fully qualified app name.
We truncate at 63 chars because some Kubernetes name fields are limited to this (by the DNS naming spec).
If release name contains chart name it will be used as a full name.
*/}}
{{- define "plgd-cloud.fullname" -}}
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

{{- define "plgd-cloud.authorization.fullname" -}}
{{- if .Values.authorization.fullnameOverride }}
{{- .Values.authorization.fullnameOverride | trunc 63 | trimSuffix "-" }}
{{- else }}
{{- $name := default .Chart.Name .Values.nameOverride }}
{{- if contains $name .Release.Name }}
{{- .Values.authorization.name | trunc 63 | trimSuffix "-" }}
{{- else }}
{{- printf "%s-%s-%s" .Release.Name $name .Values.authorization.name | trunc 63 | trimSuffix "-" }}
{{- end }}
{{- end }}
{{- end }}


{{- define "plgd-cloud.coapgateway.fullname" -}}
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

{{- define "plgd-cloud.resourceaggregate.fullname" -}}
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

{{- define "plgd-cloud.resourcedirectory.fullname" -}}
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

{{- define "plgd-cloud.certificateConfig" }}
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

{{- define "plgd-cloud.createInternalCertByCm" }}
    {{- $natsTls := .Values.coapgateway.clients.eventBus.nats.tls.certFile }}
    {{- $authClientTls := .Values.coapgateway.clients.authorizationServer.grpc.tls.certFile }}
    {{- $raClientTls := .Values.coapgateway.clients.resourceAggregate.grpc.tls.certFile }}
    {{- $rdClientTls := .Values.coapgateway.clients.resourceDirectory.grpc.tls.certFile }}
    {{- if and $natsTls $authClientTls $raClientTls $rdClientTls }}
    {{- printf "false" }}
    {{- else }}
    {{- printf "true" }}
    {{- end }}
{{- end }}

{{- define "plgd-cloud.certmanager.coapIssuerName" -}}
    {{- if .Values.certmanager.coap.issuer.name }}
        {{- printf "%s" .Values.certmanager.coap.issuer.name }}
    {{- end }}
    {{- printf "%s" .Values.certmanager.default.issuer.name }}
{{- end }}

{{- define "plgd-cloud.certmanager.coapIssuerKind" -}}
    {{- if .Values.certmanager.coap.issuer.kind }}
        {{- printf "%s" .Values.certmanager.coap.issuer.kind }}
    {{- end }}
    {{- printf "%s" .Values.certmanager.default.issuer.kind }}
{{- end }}


{{- define "plgd-cloud.certmanager.internalIssuerName" -}}
    {{- if .Values.certmanager.internal.issuer.name }}
        {{- printf "%s" .Values.certmanager.internal.issuer.name }}
    {{- end }}
    {{- printf "%s" .Values.certmanager.default.issuer.name }}
{{- end }}

{{- define "plgd-cloud.certmanager.internalIssuerKind" -}}
    {{- if .Values.certmanager.internal.issuer.kind }}
        {{- printf "%s" .Values.certmanager.internal.issuer.kind }}
    {{- end }}
    {{- printf "%s" .Values.certmanager.default.issuer.kind }}
{{- end }}

{{- define "plgd-cloud.chart" -}}
{{- printf "%s-%s" .Chart.Name .Chart.Version | replace "+" "_" | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Common labels
*/}}
{{- define "plgd-cloud.labels" -}}
helm.sh/chart: {{ include "plgd-cloud.chart" . }}
{{ include "plgd-cloud.selectorLabels" . }}
{{- if .Chart.AppVersion }}
app.kubernetes.io/version: {{ .Chart.AppVersion | quote }}
{{- end }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
{{- end }}

{{/*
Selector labels
*/}}
{{- define "plgd-cloud.selectorLabels" -}}
app.kubernetes.io/name: {{ include "plgd-cloud.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
{{- end }}

{{- define "plgd-cloud.coapgateway.selectorLabels" -}}
app.kubernetes.io/name: {{ .Values.coapgateway.name }}
app.kubernetes.io/instance: {{ .Release.Name }}
{{- end }}

{{- define "plgd-cloud.resourceaggregate.selectorLabels" -}}
app.kubernetes.io/name: {{ .Values.resourceaggregate.name }}
app.kubernetes.io/instance: {{ .Release.Name }}
{{- end }}

{{- define "plgd-cloud.natsUri" }}
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

{{- define "plgd-cloud.mongoDBUri" }}
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

{{- define "plgd-cloud.authorizationAddress" }}
  {{- $ := index . 0 }}
  {{- $address := index . 1 }}
  {{- if $address }}
  {{- printf "%s" $address }}
  {{- else }}
  {{- $authorizationServer := include "plgd-cloud.authorization.fullname" $ }}
  {{- printf "%s.%s.svc.%s:%v" $authorizationServer $.Release.Namespace $.Values.cluster.dns $.Values.authorization.port }}
  {{- end }}
{{- end }}

{{- define "plgd-cloud.resourceDirectoryAddress" }}
  {{- $ := index . 0 }}
  {{- $address := index . 1 }}
  {{- if $address }}
  {{- printf "%s" $address }}
  {{- else }}
  {{- $rdServer := include "plgd-cloud.resourcedirectory.fullname" $ }}
  {{- printf "%s.%s.svc.%s:%v" $rdServer $.Release.Namespace $.Values.cluster.dns $.Values.resourcedirectory.port }}
  {{- end }}
{{- end }}

{{- define "plgd-cloud.resourceAggregateAddress" }}
  {{- $ := index . 0 }}
  {{- $address := index . 1 }}
  {{- if $address }}
  {{- printf "%s" $address }}
  {{- else }}
  {{- $raServer := include "plgd-cloud.resourceaggregate.fullname" $ }}
  {{- printf "%s.%s.svc.%s:%v" $raServer $.Release.Namespace $.Values.cluster.dns $.Values.resourcedirectory.port }}
  {{- end }}
{{- end }}

{{- define "plgd-cloud.grpcGatewayAddress" }}
  {{- $ := index . 0 }}
  {{- $address := index . 1 }}
  {{- if $address }}
  {{- printf "%s" $address }}
  {{- else }}
  {{- $raServer := include "plgd-cloud.grpcgateway.fullname" $ }}
  {{- printf "%s.%s.svc.%s:%v" $raServer $.Release.Namespace $.Values.cluster.dns $.Values.grpcgateway.port }}
  {{- end }}
{{- end }}

{{- define  "plgd-cloud.globalDomain" }}
  {{- $ := index . 0 }}
  {{- $address := index . 1 }}
  {{- $subDomainPrefix := index . 2 }}
  {{- if $address }}
  {{- printf "%s" $address }}
  {{- else }}
  {{- printf "%s%s" $subDomainPrefix $.Values.global.domain }}
  {{- end }}
{{- end }}