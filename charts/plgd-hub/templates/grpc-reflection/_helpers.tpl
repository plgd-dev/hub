{{- define "plgd-hub.grpcreflection.fullname" -}}
{{- if .Values.grpcreflection.fullnameOverride }}
{{- .Values.grpcreflection.fullnameOverride | trunc 63 | trimSuffix "-" }}
{{- else }}
{{- $name := default .Chart.Name .Values.nameOverride }}
{{- if contains $name .Release.Name }}
{{- .Values.grpcreflection.name | trunc 63 | trimSuffix "-" }}
{{- else }}
{{- printf "%s-%s-%s" .Release.Name $name .Values.grpcreflection.name | trunc 63 | trimSuffix "-" }}
{{- end }}
{{- end }}
{{- end }}

{{- define  "plgd-hub.grpcreflection.image" -}}
    {{- $registryName := .Values.grpcreflection.image.registry | default "" -}}
    {{- $repositoryName := .Values.grpcreflection.image.repository -}}
    {{- $tag := .Values.grpcreflection.image.tag | default .Chart.AppVersion | toString -}}
    {{- printf "%s%s:%s" $registryName $repositoryName  $tag -}}
{{- end -}}

{{- define  "plgd-hub.grpcreflection.configName" -}}
    {{- $fullName :=  include "plgd-hub.grpcreflection.fullname" . -}}
    {{- printf "%s-cfg" $fullName }}
{{- end -}}

{{- define "plgd-hub.grpcreflection.createServiceCertByCm" }}
    {{- $serviceTls := .Values.grpcreflection.apis.grpc.tls.certFile }}
    {{- if $serviceTls }}
    {{- printf "" -}}
    {{- else }}
    {{- printf "true" -}}
    {{- end }}
{{- end }}

{{- define "plgd-hub.grpcreflection.serviceCertName" -}}
  {{- $fullName := include "plgd-hub.grpcreflection.fullname" . -}}
  {{- printf "%s-crt" $fullName -}}
{{- end }}

{{- define "plgd-hub.grpcreflection.domainCertName" -}}
    {{- if .Values.grpcreflection.ingress.secretName }}
        {{- printf "%s" .Values.grpcreflection.ingress.secretName -}}
    {{- else }}
        {{- $fullName := include "plgd-hub.grpcreflection.fullname" . -}}
        {{- printf "%s-domain-crt" $fullName -}}
    {{- end }}
{{- end }}


{{- define "plgd-hub.grpcreflection.domain" -}}
  {{- if .Values.grpcreflection.domain }}
    {{- printf "%s" .Values.grpcreflection.domain }}
  {{- else }}
    {{- printf "api.%s" .Values.global.domain }}
  {{- end }}
{{- end }}


{{- define "plgd-hub.grpcreflection.internalDns" -}}
  {{- $fullName := include "plgd-hub.grpcreflection.fullname" . -}}
  {{- printf "%s.%s.svc.%s" $fullName .Release.Namespace .cluster.dns }}
{{- end }}


{{- define "plgd-hub.grpcreflection.selectorLabels" -}}
app.kubernetes.io/name: {{ .Values.grpcreflection.name }}
app.kubernetes.io/instance: {{ .Release.Name }}
{{- end }}


