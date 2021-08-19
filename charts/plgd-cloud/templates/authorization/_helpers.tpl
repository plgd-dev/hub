{{- define  "plgd-cloud.authorization.image" -}}
    {{- $registryName :=  .Values.authorization.image.registry | default "" -}}
    {{- $repositoryName := .Values.authorization.image.repository -}}
    {{- $tag := .Values.authorization.image.tag | default .Chart.AppVersion | toString -}}
    {{- printf "%s%s:%s" $registryName $repositoryName  $tag -}}
{{- end -}}

{{- define  "plgd-cloud.authorization.configSecretName" -}}
    {{- $fullName :=  include "plgd-cloud.authorization.fullname" . -}}
    {{- printf "%s-cfg" $fullName }}
{{- end -}}

{{- define "plgd-cloud.authorization.createServiceCertByCm" }}
    {{- $serviceTls := .Values.authorization.apis.grpc.tls.certFile }}
    {{- if $serviceTls }}
    {{- printf "false" }}
    {{- else }}
    {{- printf "true" }}
    {{- end }}
{{- end }}

{{- define "plgd-cloud.authorization.serviceCertName" -}}
  {{- $fullName := include "plgd-cloud.fullname" . -}}
  {{- printf "%s-authorization-crt" $fullName -}}
{{- end }}

{{- define "plgd-cloud.authorization.domainCertName" -}}
  {{- $fullName := include "plgd-cloud.fullname" . -}}
  {{- printf "%s-authorization-domain-crt" $fullName -}}
{{- end }}

{{- define "plgd-cloud.authorization.selectorLabels" -}}
app.kubernetes.io/name: {{ .Values.authorization.name }}
app.kubernetes.io/instance: {{ .Release.Name }}
{{- end }}

