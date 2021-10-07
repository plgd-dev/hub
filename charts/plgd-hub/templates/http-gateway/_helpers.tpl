{{- define "plgd-hub.httpgateway.fullname" -}}
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

{{- define  "plgd-hub.httpgateway.image" -}}
    {{- $registryName := .Values.httpgateway.image.registry | default "" -}}
    {{- $repositoryName := .Values.httpgateway.image.repository -}}
    {{- $tag := .Values.httpgateway.image.tag | default .Chart.AppVersion | toString -}}
    {{- printf "%s%s:%s" $registryName $repositoryName  $tag -}}
{{- end -}}

{{- define  "plgd-hub.httpgateway.configSecretName" -}}
    {{- $fullName :=  include "plgd-hub.httpgateway.fullname" . -}}
    {{- printf "%s-cfg" $fullName }}
{{- end -}}

{{- define "plgd-hub.httpgateway.createServiceCertByCm" }}
    {{- $serviceTls := .Values.httpgateway.apis.http.tls.certFile }}
    {{- if $serviceTls }}
    {{- printf "false" }}
    {{- else }}
    {{- printf "true" }}
    {{- end }}
{{- end }}

{{- define "plgd-hub.httpgateway.serviceCertName" -}}
  {{- $fullName := include "plgd-hub.httpgateway.fullname" . -}}
  {{- printf "%s-crt" $fullName -}}
{{- end }}


{{- define "plgd-hub.httpgateway.domainCertName" -}}
  {{- $fullName := include "plgd-hub.httpgateway.fullname" . -}}
  {{- printf "%s-domain-crt" $fullName -}}
{{- end }}

{{- define "plgd-hub.httpgateway.apiDomain" -}}
  {{- if .Values.httpgateway.apiDomain }}
    {{- printf "%s" .Values.httpgateway.apiDomain }}
  {{- else }}
    {{- printf "api.%s" .Values.global.domain }}
  {{- end }}
{{- end }}

{{- define "plgd-hub.httpgateway.uiDomain" -}}
  {{- if .Values.httpgateway.uiDomain }}
    {{- printf "%s" .Values.httpgateway.uiDomain }}
  {{- else }}
    {{- printf "%s" .Values.global.domain }}
  {{- end }}
{{- end }}

{{- define "plgd-hub.httpgateway.selectorLabels" -}}
app.kubernetes.io/name: {{ .Values.httpgateway.name }}
app.kubernetes.io/instance: {{ .Release.Name }}
{{- end }}

