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
  {{- $fullName := include "plgd-cloud.resourcedirectory.fullname" . -}}
  {{- printf "%s-crt" $fullName -}}
{{- end }}

{{- define "plgd-cloud.resourcedirectory.selectorLabels" -}}
app.kubernetes.io/name: {{ .Values.resourcedirectory.name }}
app.kubernetes.io/instance: {{ .Release.Name }}
{{- end }}

{{- define "plgd-cloud.oauthConfig" }}
  {{- $ := index . 0 }}
  {{- $serviceOAuth := index . 1 }}
  {{- $pathPrefix := index . 2 }}
  clientID:{{ printf " " }}{{ required ( printf "%s.clientID is required variable" $pathPrefix ) ( $serviceOAuth.clientID | default $.Values.global.oauth.clientID ) }}
  clientSecret:{{ printf " " }}{{ required ( printf "%s.clientSecret is required variable" $pathPrefix ) ( $serviceOAuth.clientSecret | default $.Values.global.oauth.clientSecret ) }}
  {{- if ( $serviceOAuth.scopes | default $.Values.global.oauth.scopes ) }}
  scopes:
  {{- range  ( $serviceOAuth.scopes | default $.Values.global.oauth.scopes ) }}
    - {{ . | quote }}
  {{- end }}
  {{- else }}
  scopes: []
  {{- end }}
  authority:{{ printf " " }}{{ required ( printf "%s.authority is required variable" $pathPrefix ) ( $serviceOAuth.authority | default $.Values.global.oauth.authority ) }}
  redirectURL:{{ printf " " }}{{ required ( printf "%s.redirectURL is required variable" $pathPrefix ) ( $serviceOAuth.redirectURL | default $.Values.global.oauth.redirectURL ) }}
  tokenURL:{{ printf " " }}{{ required ( printf "%s.tokenURL is required variable" $pathPrefix ) ( $serviceOAuth.tokenURL | default $.Values.global.oauth.tokenURL ) }}
  audience:{{ printf " " }}{{ required ( printf "%s.audience is required variable" $pathPrefix ) ( $serviceOAuth.audience | default $.Values.global.oauth.audience ) }}
{{- end }}

{{- define "plgd-cloud.authorizationConfig" }}
  {{- $ := index . 0 }}
  {{- $serviceOAuth := index . 1 }}
  {{- $pathPrefix := index . 2 }}
  clientID:{{ printf " " }}{{ required ( printf "%s.clientID is required variable" $pathPrefix ) ( $serviceOAuth.clientID | default $.Values.global.authorization.clientID ) }}
  clientSecret:{{ printf " " }}{{ required ( printf "%s.clientSecret is required variable" $pathPrefix ) ( $serviceOAuth.clientSecret | default $.Values.global.authorization.clientSecret ) }}
  {{- if ( $serviceOAuth.scopes | default $.Values.global.authorization.scopes ) }}
  scopes:
  {{- range  ( $serviceOAuth.scopes | default $.Values.global.authorization.scopes ) }}
    - {{ . | quote }}
  {{- end }}
  {{- else }}
  scopes: []
  {{- end }}
  authority:{{ printf " " }}{{ required ( printf "%s.authority is required variable" $pathPrefix ) ( $serviceOAuth.authority | default $.Values.global.authorization.authority ) }}
  redirectURL:{{ printf " " }}{{ required ( printf "%s.redirectURL is required variable" $pathPrefix ) ( $serviceOAuth.redirectURL | default $.Values.global.authorization.redirectURL ) }}
{{- end }}


