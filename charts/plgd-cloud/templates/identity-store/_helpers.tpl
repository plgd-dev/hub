{{- define  "plgd-cloud.identitystore.image" -}}
    {{- $registryName :=  .Values.identitystore.image.registry | default "" -}}
    {{- $repositoryName := .Values.identitystore.image.repository -}}
    {{- $tag := .Values.identitystore.image.tag | default .Chart.AppVersion | toString -}}
    {{- printf "%s%s:%s" $registryName $repositoryName  $tag -}}
{{- end -}}

{{- define  "plgd-cloud.identitystore.configSecretName" -}}
    {{- $fullName :=  include "plgd-cloud.identitystore.fullname" . -}}
    {{- printf "%s-cfg" $fullName }}
{{- end -}}

{{- define "plgd-cloud.identitystore.createServiceCertByCm" }}
    {{- $serviceTls := .Values.identitystore.apis.grpc.tls.certFile }}
    {{- if $serviceTls }}
    {{- printf "false" }}
    {{- else }}
    {{- printf "true" }}
    {{- end }}
{{- end }}

{{- define "plgd-cloud.identitystore.serviceCertName" -}}
  {{- $fullName := include "plgd-cloud.identitystore.fullname" . -}}
  {{- printf "%s-crt" $fullName -}}
{{- end }}

{{- define "plgd-cloud.identitystore.domainCertName" -}}
  {{- $fullName := include "plgd-cloud.fullname" . -}}
  {{- printf "%s-identityStore-domain-crt" $fullName -}}
{{- end }}

{{- define "plgd-cloud.identitystore.selectorLabels" -}}
app.kubernetes.io/name: {{ .Values.identitystore.name }}
app.kubernetes.io/instance: {{ .Release.Name }}
{{- end }}

