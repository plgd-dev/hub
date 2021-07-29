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
{{- if .Values.coapgateway.fullnameOverride }}
{{- .Values.coapgateway.fullnameOverride | trunc 63 | trimSuffix "-" }}
{{- else }}
{{- $name := default .Chart.Name .Values.nameOverride }}
{{- if contains $name .Release.Name }}
{{- .Values.coapgateway.name | trunc 63 | trimSuffix "-" }}
{{- else }}
{{- printf "%s-%s-%s" .Release.Name $name .Values.resourceaggregate.name | trunc 63 | trimSuffix "-" }}
{{- end }}
{{- end }}
{{- end }}

{{- define  "plgd-cloud.coapgateway.image" -}}
    {{- $registryName := .Values.coapgateway.image.registry -}}
    {{- $repositoryName := .Values.coapgateway.image.repository -}}
    {{- $tag := .Values.coapgateway.image.tag | default .Chart.AppVersion | toString -}}
    {{- printf "%s/%s:%s" $registryName $repositoryName  $tag -}}
{{- end -}}

{{- define  "plgd-cloud.resourceaggregate.image" -}}
    {{- $registryName := .Values.resourceaggregate.image.registry -}}
    {{- $repositoryName := .Values.resourceaggregate.image.repository -}}
    {{- $tag := .Values.resourceaggregate.image.tag | default .Chart.AppVersion | toString -}}
    {{- printf "%s/%s:%s" $registryName $repositoryName  $tag -}}
{{- end -}}

{{- define  "plgd-cloud.coapgateway.configSecretName" -}}
    {{- $fullName :=  include "plgd-cloud.coapgateway.fullname" . -}}
    {{- printf "%s-cfg" $fullName }}
{{- end -}}

{{- define  "plgd-cloud.resourceaggregate.configSecretName" -}}
    {{- $fullName :=  include "plgd-cloud.resourceaggregate.fullname" . -}}
    {{- printf "%s-cfg" $fullName }}
{{- end -}}

{{- define "plgd-cloud.certificateConfig" }}
  {{- $ := index . 0 }}
  {{- $certDefinition := index . 1 }}
  {{- $certPath := index . 2 }}
  {{- if $certDefinition.caPool }}
  caPool: {{- printf "%s" $certDefinition.caPool }}
  {{- else if $.Values.certmanager.enabled }}
  caPool: {{- printf "%s/ca.crt" $certPath }}
  {{- end }}
  {{- if $certDefinition.keyFile }}
  keyFile: {{- printf "%s" $certDefinition.keyFile }}
  {{- else if $.Values.certmanager.enabled }}
  keyFile: {{- printf "%s/tls.key" $certPath }}
  {{- end }}
  {{- if $certDefinition.certFile }}
  certFile: {{- printf "%s" $certDefinition.certFile }}
  {{- else if $.Values.certmanager.enabled }}
  certFile: {{- printf "%s/tls.crt" $certPath }}
  {{- end }}
{{- end }}


{{- define "plgd-cloud.createInternalCertByCm" }}
    {{- $natsTls := .Values.coapgateway.clients.eventBus.nats.tls.certFile }}
    {{- $authClientTls := .Values.coapgateway.clients.authorizationServer.grpc.tls.certFile }}
    {{- $oauthHttpClientTls := .Values.coapgateway.clients.authorizationServer.oauth.http.tls.certFile }}
    {{- $raClientTls := .Values.coapgateway.clients.resourceAggregate.grpc.tls.certFile }}
    {{- $rdClientTls := .Values.coapgateway.clients.resourceDirectory.grpc.tls.certFile }}
    {{- if and $natsTls $authClientTls $oauthHttpClientTls $raClientTls $rdClientTls }}
    {{- printf "false" }}
    {{- else }}
    {{- printf "true" }}
    {{- end }}
{{- end }}

{{- define "plgd-cloud.coapgateway.createServiceCertByCm" }}
    {{- $serviceTls := .Values.coapgateway.apis.coap.tls.certFile }}
    {{- if $serviceTls }}
    {{- printf "false" }}
    {{- else }}
    {{- printf "true" }}
    {{- end }}
{{- end }}

{{- define "plgd-cloud.resourceaggregate.createServiceCertByCm" }}
    {{- $serviceTls := .Values.resourceaggregate.apis.grpc.tls.certFile }}
    {{- if $serviceTls }}
    {{- printf "false" }}
    {{- else }}
    {{- printf "true" }}
    {{- end }}
{{- end }}

{{- define "plgd-cloud.internalCertName" -}}
  {{- $fullName := include "plgd-cloud.fullname" . -}}
  {{- printf "%s-internal-crt" $fullName -}}
{{- end }}

{{- define "plgd-cloud.coapgateway.serviceCertName" -}}
  {{- $fullName := include "plgd-cloud.fullname" . -}}
  {{- printf "%s-coap-gateway-crt" $fullName -}}
{{- end }}

{{- define "plgd-cloud.resourceaggregate.serviceCertName" -}}
  {{- $fullName := include "plgd-cloud.fullname" . -}}
  {{- printf "%s-resource-aggregate-crt" $fullName -}}
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

{{/*
Create the name of the service account to use
*/}}
{{- define "plgd-cloud.serviceAccountName" -}}
{{- if .Values.serviceAccount.create }}
{{- default (include "plgd-cloud.fullname" .) .Values.serviceAccount.name }}
{{- else }}
{{- default "default" .Values.serviceAccount.name }}
{{- end }}
{{- end }}

{{- define "plgd-cloud.natsUri" }}
  {{- $ := index . 0 }}
  {{- $natsUri := index . 1 }}
  {{- if $natsUri }}
  {{- printf "%s" $natsUri }}
  {{- else }}
  {{- printf "nats://%s-nats:4222" $.Release.Namespace }}
  {{- end }}
{{- end }}

{{- define "plgd-cloud.mongoDBUri" }}
  {{- $ := index . 0 }}
  {{- $mongoUri := index . 1 }}
  {{- if $mongoUri }}
  {{- printf "%s" $mongoUri }}
  {{- else }}
  {{- printf "mongodb-0.mongodb-headless.%s.svc.cluster.local:27017" $.Release.Namespace }}
  {{- end }}
{{- end }}
