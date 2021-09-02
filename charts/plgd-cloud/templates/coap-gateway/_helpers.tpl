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

{{- define  "plgd-cloud.coapgateway.image" -}}
    {{- $registryName := .Values.coapgateway.image.registry | default "" -}}
    {{- $repositoryName := .Values.coapgateway.image.repository -}}
    {{- $tag := .Values.coapgateway.image.tag | default .Chart.AppVersion | toString -}}
    {{- printf "%s%s:%s" $registryName $repositoryName  $tag -}}
{{- end -}}

{{- define  "plgd-cloud.coapgateway.configSecretName" -}}
    {{- $fullName :=  include "plgd-cloud.coapgateway.fullname" . -}}
    {{- printf "%s-cfg" $fullName }}
{{- end -}}

{{- define "plgd-cloud.coapgateway.createServiceCertByCm" }}
    {{- $serviceTls := .Values.coapgateway.apis.coap.tls.certFile }}
    {{- if $serviceTls }}
    {{- printf "false" }}
    {{- else }}
    {{- printf "true" }}
    {{- end }}
{{- end }}

{{- define "plgd-cloud.coapgateway.serviceCertName" -}}
  {{- $fullName := include "plgd-cloud.coapgateway.fullname" . -}}
  {{- printf "%s-crt" $fullName -}}
{{- end }}

{{- define "plgd-cloud.coapgateway.selectorLabels" -}}
app.kubernetes.io/name: {{ .Values.coapgateway.name }}
app.kubernetes.io/instance: {{ .Release.Name }}
{{- end }}