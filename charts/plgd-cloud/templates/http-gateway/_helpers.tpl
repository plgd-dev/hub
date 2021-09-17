{{- define "plgd-cloud.httpgateway.fullname" -}}
{{- if .Values.httpgateway.fullnameOverride }}
{{- .Values.httpgateway.fullnameOverride | trunc 63 | trimSuffix "-" }}
{{- else }}
{{- $name := default .Chart.Name .Values.nameOverride }}
{{- if contains $name .Release.Name }}
{{- .Values.httpgateway.name | trunc 63 | trimSuffix "-" }}
{{- else }}
{{- printf "%s-%s-%s" .Release.Name $name .Values.httpgateway.name | trunc 63 | trimSuffix "-" }}
{{- end }}
{{- end }}
{{- end }}

{{- define  "plgd-cloud.httpgateway.image" -}}
    {{- $registryName := .Values.httpgateway.image.registry | default "" -}}
    {{- $repositoryName := .Values.httpgateway.image.repository -}}
    {{- $tag := .Values.httpgateway.image.tag | default .Chart.AppVersion | toString -}}
    {{- printf "%s%s:%s" $registryName $repositoryName  $tag -}}
{{- end -}}

{{- define  "plgd-cloud.httpgateway.configSecretName" -}}
    {{- $fullName :=  include "plgd-cloud.httpgateway.fullname" . -}}
    {{- printf "%s-cfg" $fullName }}
{{- end -}}

{{- define "plgd-cloud.httpgateway.createServiceCertByCm" }}
    {{- $serviceTls := .Values.httpgateway.apis.http.tls.certFile }}
    {{- if $serviceTls }}
    {{- printf "false" }}
    {{- else }}
    {{- printf "true" }}
    {{- end }}
{{- end }}

{{- define "plgd-cloud.httpgateway.serviceCertName" -}}
  {{- $fullName := include "plgd-cloud.httpgateway.fullname" . -}}
  {{- printf "%s-crt" $fullName -}}
{{- end }}


{{- define "plgd-cloud.httpgateway.domainCertName" -}}
  {{- $fullName := include "plgd-cloud.httpgateway.fullname" . -}}
  {{- printf "%s-domain-crt" $fullName -}}
{{- end }}


{{- define "plgd-cloud.httpgateway.domain" -}}
  {{- if .Values.httpgateway.domain }}
    {{- printf "%s" .Values.httpgateway.domain }}
  {{- else }}
    {{- printf "api.%s" .Values.global.domain }}
  {{- end }}
{{- end }}

{{- define "plgd-cloud.httpgateway.selectorLabels" -}}
app.kubernetes.io/name: {{ .Values.httpgateway.name }}
app.kubernetes.io/instance: {{ .Release.Name }}
{{- end }}

