{{- define "plgd-hub.mockoauthserver.fullname" -}}
{{- if .Values.mockoauthserver.fullnameOverride }}
{{- .Values.mockoauthserver.fullnameOverride | trunc 63 | trimSuffix "-" }}
{{- else }}
{{- $name := default .Chart.Name .Values.nameOverride }}
{{- if contains $name .Release.Name }}
{{- .Values.mockoauthserver.name | trunc 63 | trimSuffix "-" }}
{{- else }}
{{- printf "%s-%s-%s" .Release.Name $name .Values.mockoauthserver.name | trunc 63 | trimSuffix "-" }}
{{- end }}
{{- end }}
{{- end }}

{{- define  "plgd-hub.mockoauthserver.image" -}}
    {{- $registryName := .Values.mockoauthserver.image.registry | default "" -}}
    {{- $repositoryName := .Values.mockoauthserver.image.repository -}}
    {{- $tag := .Values.mockoauthserver.image.tag | default .Chart.AppVersion | toString -}}
    {{- printf "%s%s:%s" $registryName $repositoryName  $tag -}}
{{- end -}}

{{- define  "plgd-hub.mockoauthserver.configSecretName" -}}
    {{- $fullName :=  include "plgd-hub.mockoauthserver.fullname" . -}}
    {{- printf "%s-cfg" $fullName }}
{{- end -}}

{{- define "plgd-hub.mockoauthserver.createServiceCertByCm" }}
    {{- $serviceTls := .Values.mockoauthserver.apis.http.tls.certFile }}
    {{- if $serviceTls }}
    {{- printf "false" }}
    {{- else }}
    {{- printf "true" }}
    {{- end }}
{{- end }}

{{- define "plgd-hub.mockoauthserver.serviceCertName" -}}
  {{- $fullName := include "plgd-hub.mockoauthserver.fullname" . -}}
  {{- printf "%s-crt" $fullName -}}
{{- end }}


{{- define "plgd-hub.mockoauthserver.domainCertName" -}}
  {{- $fullName := include "plgd-hub.mockoauthserver.fullname" . -}}
  {{- printf "%s-domain-crt" $fullName -}}
{{- end }}

{{- define "plgd-hub.mockoauthserver.uri" -}}
  {{- if .Values.mockoauthserver.domain }}
    {{- printf "https://%s" .Values.mockoauthserver.domain }}
  {{- else }}
    {{- printf "https://%s" .Values.global.domain }}
  {{- end }}
{{- end }}


{{- define "plgd-hub.mockoauthserver.ingressDomain" -}}
  {{- if .Values.mockoauthserver.domain }}
    {{- printf "%s" .Values.mockoauthserver.domain }}
  {{- else }}
    {{- printf "%s" .Values.global.domain }}
  {{- end }}
{{- end }}

{{- define "plgd-hub.mockoauthserver.selectorLabels" -}}
app.kubernetes.io/name: {{ .Values.mockoauthserver.name }}
app.kubernetes.io/instance: {{ .Release.Name }}
{{- end }}

