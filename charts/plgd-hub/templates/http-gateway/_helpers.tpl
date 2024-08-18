{{- define "plgd-hub.httpgateway.fullname" -}}
{{- if .Values.httpgateway.fullnameOverride }}
{{- .Values.httpgateway.fullnameOverride | trunc 63 | trimSuffix "-" }}
{{- else }}
{{- $name := default .Chart.Name .Values.nameOverride }}
{{- if contains $name .Release.Name }}
{{- .Values.httpgateway.name | trunc 63 | trimSuffix "-" }}
{{- else }}
{{- printf "%s-%s-%s" .Release.Name $name .Values.httpgateway.name | trunc 63 | trimSuffix "-" }}
{{- end }}
{{- end }}
{{- end }}

{{- define  "plgd-hub.httpgateway.configName" -}}
    {{- $fullName :=  include "plgd-hub.httpgateway.fullname" . -}}
    {{- printf "%s-cfg" $fullName }}
{{- end -}}

{{- define  "plgd-hub.httpgateway.configThemeName" -}}
    {{- $fullName :=  include "plgd-hub.httpgateway.fullname" . -}}
    {{- printf "%s-theme-cfg" $fullName }}
{{- end -}}

{{- define "plgd-hub.httpgateway.setCustomTheme" -}}
    {{- $theme := .Values.httpgateway.ui.theme }}
    {{- if and .Values.httpgateway.enabled .Values.httpgateway.ui.enabled $theme (gt (len (printf "%q" $theme)) 0) }}
      {{- printf "true" -}}
    {{- else }}
      {{- printf "" -}}
    {{- end }}
{{- end -}}

{{- define "plgd-hub.httpgateway.createServiceCertByCm" }}
    {{- $serviceTls := .Values.httpgateway.apis.http.tls.certFile }}
    {{- if $serviceTls }}
    {{- printf "" }}
    {{- else }}
    {{- printf "true" -}}
    {{- end }}
{{- end }}

{{- define "plgd-hub.httpgateway.serviceCertName" -}}
  {{- $fullName := include "plgd-hub.httpgateway.fullname" . -}}
  {{- printf "%s-crt" $fullName -}}
{{- end }}


{{- define "plgd-hub.httpgateway.uiDomainSecretName" -}}
  {{- if .Values.httpgateway.ingress.ui.secretName }}
  {{- printf "%s" .Values.httpgateway.ingress.ui.secretName -}}
  {{- else -}}
  {{- $fullName := include "plgd-hub.httpgateway.fullname" . -}}
  {{- printf "%s-ui-domain-crt" $fullName -}}
  {{- end }}
{{- end }}

{{- define "plgd-hub.httpgateway.apiDomainSecretName" -}}
  {{- if .Values.httpgateway.ingress.api.secretName }}
  {{- printf "%s" .Values.httpgateway.ingress.api.secretName -}}
  {{- else -}}
  {{- $fullName := include "plgd-hub.httpgateway.fullname" . -}}
  {{- printf "%s-api-domain-crt" $fullName -}}
  {{- end }}
{{- end }}

{{- define "plgd-hub.httpgateway.apiDomain" -}}
  {{- if .Values.httpgateway.apiDomain }}
    {{- printf "%s" .Values.httpgateway.apiDomain }}
  {{- else }}
    {{- printf "api.%s" .Values.global.domain }}
  {{- end }}
{{- end }}

{{- define "plgd-hub.httpgateway.dpsApiDomain" -}}
  {{ $domain := "" }}
  {{- if .Values.deviceProvisioningService }}
    {{- if .Values.deviceProvisioningService.enabled }}
      {{- if .Values.deviceProvisioningService.apiDomain }}
        {{- $domain  = printf "https://%s" .Values.deviceProvisioningService.apiDomain }}
      {{- else }}
        {{- $domain = printf "https://api.%s" .Values.global.domain }}
      {{- end }}
    {{- end }}
  {{- end }}
  {{- printf $domain }}
{{- end }}

{{- define "plgd-hub.httpgateway.snippetServiceApiDomain" -}}
  {{- $domain := "" }}
  {{- if .Values.snippetservice }}
    {{- if .Values.snippetservice.enabled }}
      {{- if .Values.snippetservice.domain }}
        {{- $domain = printf "https://%s" .Values.snippetservice.domain }}
      {{- else }}
        {{- $domain = printf "https://api.%s" .Values.global.domain }}
      {{- end }}
    {{- end }}
  {{- end }}
  {{- printf $domain }}
{{- end }}

{{- define "plgd-hub.httpgateway.uiDomain" -}}
  {{- if .Values.httpgateway.uiDomain }}
    {{- printf "%s" .Values.httpgateway.uiDomain }}
  {{- else }}
    {{- printf "%s" .Values.global.domain }}
  {{- end }}
{{- end }}

{{- define "plgd-hub.httpgateway.selectorLabels" -}}
app.kubernetes.io/name: {{ .Values.httpgateway.name }}
app.kubernetes.io/instance: {{ .Release.Name }}
{{- end }}

