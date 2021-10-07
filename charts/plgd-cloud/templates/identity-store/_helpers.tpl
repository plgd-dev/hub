{{- define  "plgd-hub.identitystore.image" -}}
    {{- $registryName :=  .Values.identitystore.image.registry | default "" -}}
    {{- $repositoryName := .Values.identitystore.image.repository -}}
    {{- $tag := .Values.identitystore.image.tag | default .Chart.AppVersion | toString -}}
    {{- printf "%s%s:%s" $registryName $repositoryName  $tag -}}
{{- end -}}

{{- define  "plgd-hub.identitystore.configSecretName" -}}
    {{- $fullName :=  include "plgd-hub.identitystore.fullname" . -}}
    {{- printf "%s-cfg" $fullName }}
{{- end -}}

{{- define "plgd-hub.identitystore.createServiceCertByCm" }}
    {{- $serviceTls := .Values.identitystore.apis.grpc.tls.certFile }}
    {{- if $serviceTls }}
    {{- printf "false" }}
    {{- else }}
    {{- printf "true" }}
    {{- end }}
{{- end }}

{{- define "plgd-hub.identitystore.serviceCertName" -}}
  {{- $fullName := include "plgd-hub.identitystore.fullname" . -}}
  {{- printf "%s-crt" $fullName -}}
{{- end }}

{{- define "plgd-hub.identitystore.domainCertName" -}}
  {{- $fullName := include "plgd-hub.fullname" . -}}
  {{- printf "%s-identityStore-domain-crt" $fullName -}}
{{- end }}

{{- define "plgd-hub.identitystore.selectorLabels" -}}
app.kubernetes.io/name: {{ .Values.identitystore.name }}
app.kubernetes.io/instance: {{ .Release.Name }}
{{- end }}

