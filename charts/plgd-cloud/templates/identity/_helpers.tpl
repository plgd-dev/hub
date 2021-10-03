{{- define  "plgd-cloud.identity.image" -}}
    {{- $registryName :=  .Values.identity.image.registry | default "" -}}
    {{- $repositoryName := .Values.identity.image.repository -}}
    {{- $tag := .Values.identity.image.tag | default .Chart.AppVersion | toString -}}
    {{- printf "%s%s:%s" $registryName $repositoryName  $tag -}}
{{- end -}}

{{- define  "plgd-cloud.identity.configSecretName" -}}
    {{- $fullName :=  include "plgd-cloud.identity.fullname" . -}}
    {{- printf "%s-cfg" $fullName }}
{{- end -}}

{{- define "plgd-cloud.identity.createServiceCertByCm" }}
    {{- $serviceTls := .Values.identity.apis.grpc.tls.certFile }}
    {{- if $serviceTls }}
    {{- printf "false" }}
    {{- else }}
    {{- printf "true" }}
    {{- end }}
{{- end }}

{{- define "plgd-cloud.identity.serviceCertName" -}}
  {{- $fullName := include "plgd-cloud.identity.fullname" . -}}
  {{- printf "%s-crt" $fullName -}}
{{- end }}

{{- define "plgd-cloud.identity.domainCertName" -}}
  {{- $fullName := include "plgd-cloud.fullname" . -}}
  {{- printf "%s-identity-domain-crt" $fullName -}}
{{- end }}

{{- define "plgd-cloud.identity.selectorLabels" -}}
app.kubernetes.io/name: {{ .Values.identity.name }}
app.kubernetes.io/instance: {{ .Release.Name }}
{{- end }}

