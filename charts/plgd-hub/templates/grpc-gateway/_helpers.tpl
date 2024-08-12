{{- define "plgd-hub.grpcgateway.fullname" -}}
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

{{- define  "plgd-hub.grpcgateway.configName" -}}
    {{- $fullName :=  include "plgd-hub.grpcgateway.fullname" . -}}
    {{- printf "%s-cfg" $fullName }}
{{- end -}}

{{- define "plgd-hub.grpcgateway.createServiceCertByCm" }}
    {{- $serviceTls := .Values.grpcgateway.apis.grpc.tls.certFile }}
    {{- if $serviceTls }}
    {{- printf "" -}}
    {{- else }}
    {{- printf "true" -}}
    {{- end }}
{{- end }}

{{- define "plgd-hub.grpcgateway.serviceCertName" -}}
  {{- $fullName := include "plgd-hub.grpcgateway.fullname" . -}}
  {{- printf "%s-crt" $fullName -}}
{{- end }}

{{- define "plgd-hub.grpcgateway.domainCertName" -}}
    {{- if .Values.grpcgateway.ingress.secretName }}
        {{- printf "%s" .Values.grpcgateway.ingress.secretName -}}
    {{- else }}
        {{- $fullName := include "plgd-hub.grpcgateway.fullname" . -}}
        {{- printf "%s-domain-crt" $fullName -}}
    {{- end }}
{{- end }}


{{- define "plgd-hub.grpcgateway.domain" -}}
  {{- if .Values.grpcgateway.domain }}
    {{- printf "%s" .Values.grpcgateway.domain }}
  {{- else }}
    {{- printf "api.%s" .Values.global.domain }}
  {{- end }}
{{- end }}

{{- define "plgd-hub.grpcgateway.selectorLabels" -}}
app.kubernetes.io/name: {{ .Values.grpcgateway.name }}
app.kubernetes.io/instance: {{ .Release.Name }}
{{- end }}


