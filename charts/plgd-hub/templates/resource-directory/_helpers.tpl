{{- define "plgd-hub.resourcedirectory.fullname" -}}
{{- if .Values.resourcedirectory.fullnameOverride }}
{{- .Values.resourcedirectory.fullnameOverride | trunc 63 | trimSuffix "-" }}
{{- else }}
{{- $name := default .Chart.Name .Values.nameOverride }}
{{- if contains $name .Release.Name }}
{{- .Values.resourcedirectory.name | trunc 63 | trimSuffix "-" }}
{{- else }}
{{- printf "%s-%s-%s" .Release.Name $name .Values.resourcedirectory.name | trunc 63 | trimSuffix "-" }}
{{- end }}
{{- end }}
{{- end }}

{{- define  "plgd-hub.resourcedirectory.image" -}}
    {{- $registryName := .Values.resourcedirectory.image.registry | default "" -}}
    {{- $repositoryName := .Values.resourcedirectory.image.repository -}}
    {{- $tag := .Values.resourcedirectory.image.tag | default .Chart.AppVersion | toString -}}
    {{- printf "%s%s:%s" $registryName $repositoryName  $tag -}}
{{- end -}}

{{- define  "plgd-hub.resourcedirectory.configName" -}}
    {{- $fullName :=  include "plgd-hub.resourcedirectory.fullname" . -}}
    {{- printf "%s-cfg" $fullName }}
{{- end -}}

{{- define "plgd-hub.resourcedirectory.createServiceCertByCm" }}
    {{- $serviceTls := .Values.resourcedirectory.apis.grpc.tls.certFile }}
    {{- if $serviceTls }}
    {{- printf "" -}}
    {{- else }}
    {{- printf "true" -}}
    {{- end }}
{{- end }}

{{- define "plgd-hub.resourcedirectory.serviceCertName" -}}
  {{- $fullName := include "plgd-hub.resourcedirectory.fullname" . -}}
  {{- printf "%s-crt" $fullName -}}
{{- end }}

{{- define "plgd-hub.resourcedirectory.selectorLabels" -}}
app.kubernetes.io/name: {{ .Values.resourcedirectory.name }}
app.kubernetes.io/instance: {{ .Release.Name }}
{{- end }}



