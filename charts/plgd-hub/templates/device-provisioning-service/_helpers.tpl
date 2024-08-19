{{- define "plgd-hub.deviceProvisioningService.fullname" -}}
{{- if .Values.deviceProvisioningService.fullnameOverride }}
{{- .Values.deviceProvisioningService.fullnameOverride | trunc 63 | trimSuffix "-" }}
{{- else }}
{{- $name := default .Chart.Name .Values.nameOverride }}
{{- if contains $name .Release.Name }}
{{- .Values.deviceProvisioningService.name | trunc 63 | trimSuffix "-" }}
{{- else }}
{{- printf "%s-%s-%s" .Release.Name $name .Values.deviceProvisioningService.name | trunc 63 | trimSuffix "-" }}
{{- end }}
{{- end }}
{{- end }}

{{- define  "plgd-hub.deviceProvisioningService.configName" -}}
    {{- $fullName :=  include "plgd-hub.deviceProvisioningService.fullname" . -}}
    {{- printf "%s-cfg" $fullName }}
{{- end -}}

{{- define "plgd-hub.deviceProvisioningService.apiDomain" -}}
  {{- if .Values.deviceProvisioningService.apiDomain }}
    {{- printf "%s" .Values.deviceProvisioningService.apiDomain }}
  {{- else }}
    {{- printf "api.%s" .Values.global.domain }}
  {{- end }}
{{- end }}

{{- define "plgd-hub.deviceProvisioningService.serviceCertificateMountPath" }}
{{- .Values.deviceProvisioningService.service.certificate.mountPath | default "/certs" }}
{{- end }}

{{- define "plgd-hub.deviceProvisioningService.clientCertName" -}}
  {{- $fullName := include "plgd-hub.deviceProvisioningService.fullname" . }}
  {{- printf "%s-client-crt" $fullName -}}
{{- end }}

{{- define "plgd-hub.deviceProvisioningService.serviceCertName" -}}
  {{- $fullName := include "plgd-hub.deviceProvisioningService.fullname" . }}
  {{- printf "%s-crt" $fullName -}}
{{- end }}

{{- define "plgd-hub.deviceProvisioningService.domainCertName" -}}
  {{- $fullName := include "plgd-hub.deviceProvisioningService.fullname" . }}
  {{- if .Values.deviceProvisioningService.ingress.domainCertName }}
    {{- .Values.deviceProvisioningService.ingress.domainCertName }}
  {{- else }}
    {{- printf "%s-domain-crt" $fullName -}}
  {{- end }}
{{- end }}

{{- define "plgd-hub.deviceProvisioningService.selectorLabels" -}}
app.kubernetes.io/name: {{ .Values.deviceProvisioningService.name }}
app.kubernetes.io/instance: {{ .Release.Name }}
{{- end }}

{{- define "plgd-hub.deviceProvisioningService.labels" -}}
helm.sh/chart: {{ include "plgd-hub.chart" . }}
{{ include "plgd-hub.deviceProvisioningService.selectorLabels" . }}
{{- if .Chart.AppVersion }}
app.kubernetes.io/version: {{ .Chart.AppVersion | quote }}
{{- end }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
{{- end }}


{{- define "plgd-hub.deviceProvisioningService.authority" }}
  {{- $ := index . 0 }}
  {{- $authProvider := index . 1 }}
  {{- $authority := required "Authorization authority is required. Set global.authority or authority per enrollment group or in case you use the DPS with the mock OAuth Server, make sure global.domain is set." ( $authProvider.authority | default ( $.Values.global.authority | default $.Values.global.domain )) }}
  {{- if hasPrefix "http" $authority }}
    {{- $authority }}
  {{- else }}
    {{- printf "https://%s" $authority }}
  {{- end }}
{{- end }}

{{- define "plgd-hub.deviceProvisioningService.coapGateway" }}
  {{- $ := index . 0 }}
  {{- $hub := index . 1 }}
  {{- if $hub.coapGateway }}
    {{- $hub.coapGateway }}
  {{- else if $.Values.global.domain }}
    {{- printf "%s:%v" $.Values.global.domain ( $.Values.coapgateway.service.nodePort | default 5684 ) }}
  {{- else }}
    {{- fail "CoAP Gateway address is required. Use global.domain or deviceProvisioningService.enrollmentGroups[].hub.coapGateway. In case of using global domain, default port 5684 or coapgateway.service.nodePort.port if specified is used" }}
  {{- end }}
{{- end }}

{{- define "plgd-hub.deviceProvisioningService.certificateAuthority" }}
  {{- $ := index . 0 }}
  {{- $certificateAuthority := index . 1 }}
  {{- $ret := "" }}
  {{- if $certificateAuthority }}
    {{- if $certificateAuthority.grpc }}
      {{- if $certificateAuthority.grpc.address }}
        {{- $ret = $certificateAuthority.grpc.address }}
      {{- end }}
    {{- end }}
  {{- end }}
  {{- if and (empty $ret) $.Values.certificateauthority }}
    {{- if $.Values.certificateauthority.domain }}
      {{- $ret = printf "%s:443" $.Values.certificateauthority.domain }}
    {{- end }}
  {{- end }}
  {{- if empty $ret }}
    {{- $ret = printf "api.%s:443" $.Values.global.domain }}
  {{- end }}
  {{- $ret }}
{{- end }}