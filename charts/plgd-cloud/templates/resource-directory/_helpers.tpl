{{- define "plgd-cloud.resourcedirectory.fullname" -}}
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

{{- define  "plgd-cloud.resourcedirectory.image" -}}
    {{- $registryName := .Values.resourcedirectory.image.registry | default "" -}}
    {{- $repositoryName := .Values.resourcedirectory.image.repository -}}
    {{- $tag := .Values.resourcedirectory.image.tag | default .Chart.AppVersion | toString -}}
    {{- printf "%s%s:%s" $registryName $repositoryName  $tag -}}
{{- end -}}

{{- define  "plgd-cloud.resourcedirectory.configSecretName" -}}
    {{- $fullName :=  include "plgd-cloud.resourcedirectory.fullname" . -}}
    {{- printf "%s-cfg" $fullName }}
{{- end -}}

{{- define "plgd-cloud.resourcedirectory.createServiceCertByCm" }}
    {{- $serviceTls := .Values.resourcedirectory.apis.grpc.tls.certFile }}
    {{- if $serviceTls }}
    {{- printf "false" }}
    {{- else }}
    {{- printf "true" }}
    {{- end }}
{{- end }}

{{- define "plgd-cloud.resourcedirectory.serviceCertName" -}}
  {{- $fullName := include "plgd-cloud.fullname" . -}}
  {{- printf "%s-resource-directory-crt" $fullName -}}
{{- end }}

{{- define "plgd-cloud.resourcedirectory.selectorLabels" -}}
app.kubernetes.io/name: {{ .Values.resourcedirectory.name }}
app.kubernetes.io/instance: {{ .Release.Name }}
{{- end }}

