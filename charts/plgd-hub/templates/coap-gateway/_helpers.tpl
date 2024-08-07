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

{{- define  "plgd-hub.coapgateway.configName" -}}
    {{- $fullName :=  include "plgd-hub.coapgateway.fullname" . -}}
    {{- printf "%s-cfg" $fullName }}
{{- end -}}

{{- define "plgd-hub.coapgateway.createServiceCertByCm" }}
    {{- $serviceTls := .Values.coapgateway.apis.coap.tls.certFile }}
    {{- if $serviceTls }}
    {{- printf "" -}}
    {{- else }}
    {{- printf "true" -}}
    {{- end }}
{{- end }}

{{- define "plgd-hub.coapgateway.serviceCertName" -}}
  {{- $fullName := include "plgd-hub.coapgateway.fullname" . -}}
  {{- printf "%s-crt" $fullName -}}
{{- end }}

{{- define "plgd-hub.coapgateway.clientCertName" -}}
  {{- $fullName := include "plgd-hub.coapgateway.fullname" . -}}
  {{- printf "%s-client-crt" $fullName -}}
{{- end }}

{{- define "plgd-hub.coapgateway.selectorLabels" -}}
app.kubernetes.io/name: {{ .Values.coapgateway.name }}
app.kubernetes.io/instance: {{ .Release.Name }}
{{- end }}