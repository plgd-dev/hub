{{- define "plgd-hub.snippetservice.fullname" -}}
{{- if .Values.snippetservice.fullnameOverride }}
{{- .Values.snippetservice.fullnameOverride | trunc 63 | trimSuffix "-" }}
{{- else }}
{{- $name := default .Chart.Name .Values.nameOverride }}
{{- if contains $name .Release.Name }}
{{- .Values.snippetservice.name | trunc 63 | trimSuffix "-" }}
{{- else }}
{{- printf "%s-%s-%s" .Release.Name $name .Values.snippetservice.name | trunc 63 | trimSuffix "-" }}
{{- end }}
{{- end }}
{{- end }}

{{- define  "plgd-hub.snippetservice.configName" -}}
    {{- $fullName :=  include "plgd-hub.snippetservice.fullname" . -}}
    {{- printf "%s-cfg" $fullName }}
{{- end -}}

{{- define "plgd-hub.snippetservice.createServiceCertByCm" }}
    {{- $serviceTls := .Values.snippetservice.apis.grpc.tls.certFile }}
    {{- if $serviceTls }}
    {{- printf "" -}}
    {{- else }}
    {{- printf "true" -}}
    {{- end }}
{{- end }}

{{- define "plgd-hub.snippetservice.domain" -}}
  {{- if .Values.snippetservice.domain }}
    {{- printf "%s" .Values.snippetservice.domain }}
  {{- else }}
    {{- printf "api.%s" .Values.global.domain }}
  {{- end }}
{{- end }}

{{- define "plgd-hub.snippetservice.serviceCertName" -}}
  {{- $fullName := include "plgd-hub.snippetservice.fullname" . -}}
  {{- printf "%s-crt" $fullName -}}
{{- end }}

{{- define "plgd-hub.snippetservice.domainCertName" -}}
    {{- if .Values.snippetservice.ingress.secretName }}
        {{- printf "%s" .Values.snippetservice.ingress.secretName -}}
    {{- else }}
        {{- $fullName := include "plgd-hub.snippetservice.fullname" . -}}
        {{- printf "%s-domain-crt" $fullName -}}
    {{- end }}
{{- end }}

{{- define "plgd-hub.snippetservice.selectorLabels" -}}
app.kubernetes.io/name: {{ .Values.snippetservice.name }}
app.kubernetes.io/instance: {{ .Release.Name }}
{{- end }}