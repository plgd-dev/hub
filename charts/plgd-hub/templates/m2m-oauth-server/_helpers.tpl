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
    {{- $serviceTls := .Values.m2moauthserver.apis.grpc.tls.certFile }}
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
{{- printf "https://%s/m2m-oauth-server" (include "plgd-hub.m2moauthserver.ingressDomain" .) }}
{{- end }}


{{- define "plgd-hub.m2moauthserver.ingressDomain" -}}
  {{- if .Values.m2moauthserver.domain }}
    {{- printf "%s" .Values.m2moauthserver.domain }}
  {{- else }}
    {{- printf "%s" .Values.global.domain }}
  {{- end }}
{{- end }}

{{- define "plgd-hub.m2moauthserver.selectorLabels" -}}
app.kubernetes.io/name: {{ .Values.m2moauthserver.name }}
app.kubernetes.io/instance: {{ .Release.Name }}
{{- end }}


{{- define "plgd-hub.m2moauthserver.getJwtPrivateKeyClient" -}}
{{- $ := . -}}
{{- $clientID := dict }}
{{- if include "plgd-hub.m2moauthserver.enabled" $ }}
{{- range $.Values.m2moauthserver.oauthSigner.clients }}
{{- if .jwtPrivateKey }}
{{- if .jwtPrivateKey.enabled }}
{{- $clientID = . }}
{{- end }}
{{- end }}
{{- end }}
{{- end }}
{{- $clientID | toYaml }}
{{- end }}

{{- define "plgd-hub.m2moauthserver.privateKeySecretEnabled" -}}
{{- if or .Values.global.m2mOAuthServer.privateKey .Values.m2moauthserver.privateKey.enabled }}
true
{{- else }}
{{- printf "" }}
{{- end }}
{{- end }}

{{- define "plgd-hub.m2moauthserver.getPrivateKeyFile" -}}
{{- $privateKeyFile := .Values.m2moauthserver.oauthSigner.privateKeyFile }}
{{- if and (not $privateKeyFile) (include "plgd-hub.m2moauthserver.privateKeySecretEnabled" $) }}
{{- $privateKeyFile = printf "%s/%s" .Values.m2moauthserver.privateKey.mountPath .Values.m2moauthserver.privateKey.fileName }}
{{- end }}
{{- printf "%s" $privateKeyFile }}
{{- end -}}

{{- define "plgd-hub.m2moauthserver.enabled" -}}
{{- if and .Values.m2moauthserver.enabled (include "plgd-hub.m2moauthserver.privateKeySecretEnabled" .) }}
true
{{- else }}
{{- printf "" }}
{{- end }}
{{- end }}