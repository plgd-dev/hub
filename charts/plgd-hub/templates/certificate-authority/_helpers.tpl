{{- define "plgd-hub.certificateauthority.fullname" -}}
{{- if .Values.certificateauthority.fullnameOverride }}
{{- .Values.certificateauthority.fullnameOverride | trunc 63 | trimSuffix "-" }}
{{- else }}
{{- $name := default .Chart.Name .Values.nameOverride }}
{{- if contains $name .Release.Name }}
{{- .Values.certificateauthority.name | trunc 63 | trimSuffix "-" }}
{{- else }}
{{- printf "%s-%s-%s" .Release.Name $name .Values.certificateauthority.name | trunc 63 | trimSuffix "-" }}
{{- end }}
{{- end }}
{{- end }}

{{- define  "plgd-hub.certificateauthority.configName" -}}
    {{- $fullName :=  include "plgd-hub.certificateauthority.fullname" . -}}
    {{- printf "%s-cfg" $fullName }}
{{- end -}}

{{- define "plgd-hub.certificateauthority.createServiceCertByCm" }}
    {{- $serviceTls := .Values.certificateauthority.apis.grpc.tls.certFile }}
    {{- if $serviceTls }}
    {{- printf "" -}}
    {{- else }}
    {{- printf "true" -}}
    {{- end }}
{{- end }}

{{- define "plgd-hub.certificateauthority.domain" -}}
  {{- if .Values.certificateauthority.domain }}
    {{- printf "%s" .Values.certificateauthority.domain }}
  {{- else }}
    {{- printf "api.%s" .Values.global.domain }}
  {{- end }}
{{- end }}

{{- define "plgd-hub.certificateauthority.serviceCertName" -}}
  {{- $fullName := include "plgd-hub.certificateauthority.fullname" . -}}
  {{- printf "%s-crt" $fullName -}}
{{- end }}

{{- define "plgd-hub.certificateauthority.domainCertName" -}}
    {{- if .Values.certificateauthority.ingress.secretName }}
        {{- printf "%s" .Values.certificateauthority.ingress.secretName -}}
    {{- else }}
        {{- $fullName := include "plgd-hub.certificateauthority.fullname" . -}}
        {{- printf "%s-domain-crt" $fullName -}}
    {{- end }}
{{- end }}

{{- define "plgd-hub.certificateauthority.selectorLabels" -}}
app.kubernetes.io/name: {{ .Values.certificateauthority.name }}
app.kubernetes.io/instance: {{ .Release.Name }}
{{- end }}