{{- define  "plgd-hub.resourceaggregate.configName" -}}
    {{- $fullName :=  include "plgd-hub.resourceaggregate.fullname" . -}}
    {{- printf "%s-cfg" $fullName }}
{{- end -}}

{{- define "plgd-hub.resourceaggregate.createServiceCertByCm" }}
    {{- $serviceTls := .Values.resourceaggregate.apis.grpc.tls.certFile }}
    {{- if $serviceTls }}
    {{- printf "" -}}
    {{- else }}
    {{- printf "true" -}}
    {{- end }}
{{- end }}

{{- define "plgd-hub.resourceaggregate.serviceCertName" -}}
  {{- $fullName := include "plgd-hub.resourceaggregate.fullname" . -}}
  {{- printf "%s-crt" $fullName -}}
{{- end }}

{{- define "plgd-hub.resourceaggregate.selectorLabels" -}}
app.kubernetes.io/name: {{ .Values.resourceaggregate.name }}
app.kubernetes.io/instance: {{ .Release.Name }}
{{- end }}