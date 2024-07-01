{{- define "plgd-hub.m2moauthserver.fullname" -}}
{{- if .Values.m2moauthserver.fullnameOverride }}
{{- .Values.m2moauthserver.fullnameOverride | trunc 63 | trimSuffix "-" }}
{{- else }}
{{- $name := default .Chart.Name .Values.nameOverride }}
{{- if contains $name .Release.Name }}
{{- .Values.m2moauthserver.name | trunc 63 | trimSuffix "-" }}
{{- else }}
{{- printf "%s-%s-%s" .Release.Name $name .Values.m2moauthserver.name | trunc 63 | trimSuffix "-" }}
{{- end }}
{{- end }}
{{- end }}

{{- define  "plgd-hub.m2moauthserver.configName" -}}
    {{- $fullName :=  include "plgd-hub.m2moauthserver.fullname" . -}}
    {{- printf "%s-cfg" $fullName }}
{{- end -}}

{{- define "plgd-hub.m2moauthserver.createServiceCertByCm" }}
    {{- $serviceTls := .Values.m2moauthserver.apis.http.tls.certFile }}
    {{- if $serviceTls }}
    {{- printf "" -}}
    {{- else }}
    {{- printf "true" -}}
    {{- end }}
{{- end }}

{{- define "plgd-hub.m2moauthserver.serviceCertName" -}}
  {{- $fullName := include "plgd-hub.m2moauthserver.fullname" . -}}
  {{- printf "%s-crt" $fullName -}}
{{- end }}


{{- define "plgd-hub.m2moauthserver.domainCertName" -}}
  {{- $fullName := include "plgd-hub.m2moauthserver.fullname" . -}}
  {{- printf "%s-domain-crt" $fullName -}}
{{- end }}

{{- define "plgd-hub.m2moauthserver.uri" -}}
{{- printf "https://%s" (include "plgd-hub.m2moauthserver.ingressDomain" .) }}
{{- end }}


{{- define "plgd-hub.m2moauthserver.ingressDomain" -}}
  {{- if .Values.m2moauthserver.domain }}
    {{- printf "%s" .Values.m2moauthserver.domain }}
  {{- else }}
    {{- printf "m2m-auth.%s" .Values.global.domain }}
  {{- end }}
{{- end }}

{{- define "plgd-hub.m2moauthserver.selectorLabels" -}}
app.kubernetes.io/name: {{ .Values.m2moauthserver.name }}
app.kubernetes.io/instance: {{ .Release.Name }}
{{- end }}


{{- define "plgd-hub.m2moauthserver.getJWTPrivateKeyClient" -}}
{{- $clientID := dict }}
{{- range . }}
{{- if .privateKeyJWT.enabled }}
{{- $clientID = . }}
{{- end }}
{{- end }}
{{- $clientID | toYaml }}
{{- end }}