
{{- define "plgd-hub.mongodb-standby-tool.fullname" -}}
{{- if .Values.mongodb.standbyTool.fullnameOverride }}
{{- .Values.mongodb.standbyTool.fullnameOverride | trunc 63 | trimSuffix "-" }}
{{- else }}
{{- $name := default .Chart.Name .Values.nameOverride }}
{{- if contains $name .Release.Name }}
{{- .Values.mongodb.standbyTool.name | trunc 63 | trimSuffix "-" }}
{{- else }}
{{- printf "%s-%s-%s" .Release.Name $name .Values.mongodb.standbyTool.name | trunc 63 | trimSuffix "-" }}
{{- end }}
{{- end }}
{{- end }}

{{- define "plgd-hub.mongodb-standby-tool.selectorLabels" -}}
app.kubernetes.io/name: {{ .Values.mongodb.standbyTool.name }}
app.kubernetes.io/instance: {{ .Release.Name }}
{{- end }}

{{- define  "plgd-hub.mongodb-standby-tool.image" -}}
    {{- $registryName := .Values.mongodb.standbyTool.image.registry | default "" -}}
    {{- $repositoryName := .Values.mongodb.standbyTool.image.repository -}}
    {{- $tag := .Values.mongodb.standbyTool.image.tag | default .Chart.AppVersion | toString -}}
    {{- printf "%s%s:%s" $registryName $repositoryName  $tag -}}
{{- end -}}

{{- define "plgd-hub.mongodb-standby-tool.createCertByCm" }}
    {{- $serviceTls := .Values.mongodb.standbyTool.clients.storage.mongoDB.tls.certFile }}
    {{- if $serviceTls }}
    {{- printf "" -}}
    {{- else }}
    {{- printf "true" -}}
    {{- end }}
{{- end }}

{{- define  "plgd-hub.mongodb-standby-tool.configName" -}}
    {{- $fullName :=  include "plgd-hub.mongodb-standby-tool.fullname" . -}}
    {{- printf "%s-cfg" $fullName }}
{{- end -}}

{{- define "plgd-hub.mongodb-standby-tool.jobCertName" -}}
  {{- printf "mongodb-cm-crt" -}}
{{- end }}

{{- define "plgd-hub.mongodb-standby-tool.enabled" }}
  {{- if and .Values.mongodb.enabled .Values.mongodb.standbyTool.enabled (not (empty .Values.mongodb.standbyTool.replicaSet.standby.members)) }}
  {{- printf "true" -}}
  {{- else }}
  {{- printf "" -}}
  {{- end }}
{{- end }}