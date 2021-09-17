{{- define "plgd-cloud.grpcgateway.fullname" -}}
{{- if .Values.grpcgateway.fullnameOverride }}
{{- .Values.grpcgateway.fullnameOverride | trunc 63 | trimSuffix "-" }}
{{- else }}
{{- $name := default .Chart.Name .Values.nameOverride }}
{{- if contains $name .Release.Name }}
{{- .Values.grpcgateway.name | trunc 63 | trimSuffix "-" }}
{{- else }}
{{- printf "%s-%s-%s" .Release.Name $name .Values.grpcgateway.name | trunc 63 | trimSuffix "-" }}
{{- end }}
{{- end }}
{{- end }}

{{- define  "plgd-cloud.grpcgateway.image" -}}
    {{- $registryName := .Values.grpcgateway.image.registry | default "" -}}
    {{- $repositoryName := .Values.grpcgateway.image.repository -}}
    {{- $tag := .Values.grpcgateway.image.tag | default .Chart.AppVersion | toString -}}
    {{- printf "%s%s:%s" $registryName $repositoryName  $tag -}}
{{- end -}}

{{- define  "plgd-cloud.grpcgateway.configSecretName" -}}
    {{- $fullName :=  include "plgd-cloud.grpcgateway.fullname" . -}}
    {{- printf "%s-cfg" $fullName }}
{{- end -}}

{{- define "plgd-cloud.grpcgateway.createServiceCertByCm" }}
    {{- $serviceTls := .Values.grpcgateway.apis.grpc.tls.certFile }}
    {{- if $serviceTls }}
    {{- printf "false" }}
    {{- else }}
    {{- printf "true" }}
    {{- end }}
{{- end }}

{{- define "plgd-cloud.grpcgateway.serviceCertName" -}}
  {{- $fullName := include "plgd-cloud.grpcgateway.fullname" . -}}
  {{- printf "%s-crt" $fullName -}}
{{- end }}

{{- define "plgd-cloud.grpcgateway.domainCertName" -}}
  {{- $fullName := include "plgd-cloud.grpcgateway.fullname" . -}}
  {{- printf "%s-domain-crt" $fullName -}}
{{- end }}


{{- define "plgd-cloud.grpcgateway.domain" -}}
  {{- if .Values.httpgateway.domain }}
    {{- printf "%s" .Values.httpgateway.domain }}
  {{- else }}
    {{- printf "api.%s" .Values.global.domain }}
  {{- end }}
{{- end }}


{{- define "plgd-cloud.grpcgateway.internalDns" -}}
  {{- $fullName := include "plgd-cloud.grpcgateway.fullname" . -}}
  {{- printf "%s.%s.svc.cluster.local" $fullName .Release.Namespace }}
{{- end }}


{{- define "plgd-cloud.grpcgateway.selectorLabels" -}}
app.kubernetes.io/name: {{ .Values.grpcgateway.name }}
app.kubernetes.io/instance: {{ .Release.Name }}
{{- end }}


