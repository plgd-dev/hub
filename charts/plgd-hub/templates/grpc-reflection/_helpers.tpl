{{- define "plgd-hub.grpcreflection.fullname" -}}
{{- if .Values.grpcreflection.fullnameOverride }}
{{- .Values.grpcreflection.fullnameOverride | trunc 63 | trimSuffix "-" }}
{{- else }}
{{- $name := default .Chart.Name .Values.nameOverride }}
{{- if contains $name .Release.Name }}
{{- .Values.grpcreflection.name | trunc 63 | trimSuffix "-" }}
{{- else }}
{{- printf "%s-%s-%s" .Release.Name $name .Values.grpcreflection.name | trunc 63 | trimSuffix "-" }}
{{- end }}
{{- end }}
{{- end }}

{{- define  "plgd-hub.grpcreflection.configName" -}}
    {{- $domain  := index . 0 }}
    {{- $fullName := index . 1 }}
    {{- printf "%s-cfg" ( include "plgd-hub.grpcreflection.domainToName" (list $domain $fullName)) }}
{{- end -}}

{{- define "plgd-hub.grpcreflection.domainToName" -}}
    {{- $domain := index . 0 }}
    {{- $fullname := index . 1 }}
    {{- (printf "%s-%s" $domain $fullname) | replace "+" "-" | replace "." "-" | trunc 63 | trimSuffix "-" }}
{{- end -}}

{{- define "plgd-hub.grpcreflection.mapServicesToDomains" -}}
    {{- $ret := dict }}
    {{- if .Values.grpcgateway.enabled }}
      {{- $ret = merge $ret (dict (include "plgd-hub.grpcgateway.domain" .) (list "grpcgateway.pb.GrpcGateway")) -}}
    {{- end }}
    {{- if .Values.certificateauthority.enabled }}
      {{- $key := include "plgd-hub.certificateauthority.domain" . }}
      {{- $service := "certificateauthority.pb.CertificateAuthority" }}
      {{- if hasKey $ret $key }}
        {{- $existingList := index $ret $key }}
        {{- $newList := append $existingList $service }}
        {{- $ret = merge (dict $key $newList) $ret  -}}
      {{- else }}
        {{- $ret = merge $ret (dict $key (list $service)) -}}
      {{- end }}
    {{- end }}
    {{- if .Values.snippetservice.enabled }}
      {{- $key := include "plgd-hub.snippetservice.domain" . }}
      {{- $service := "snippetservice.pb.SnippetService" }}
      {{- if hasKey $ret $key }}
        {{- $existingList := index $ret $key }}
        {{- $newList := append $existingList $service }}
        {{- $ret = merge (dict $key $newList) $ret -}}
      {{- else }}
        {{- $ret = merge $ret (dict $key (list $service)) -}}
      {{- end }}
    {{- end }}
    {{- if include "plgd-hub.m2moauthserver.enabled" . }}
      {{- $key := include "plgd-hub.m2moauthserver.ingressDomain" . }}
      {{- $service := "m2moauthserver.pb.M2MOAuthService" }}
      {{- if hasKey $ret $key }}
        {{- $existingList := index $ret $key }}
        {{- $newList := append $existingList $service }}
        {{- $ret = merge (dict $key $newList) $ret -}}
      {{- else }}
        {{- $ret = merge $ret (dict $key (list $service)) -}}
      {{- end }}
    {{- end }}
    {{- toYaml $ret }}
{{- end -}}

{{- define "plgd-hub.grpcreflection.createServiceCertByCm" }}
    {{- $serviceTls := .Values.grpcreflection.apis.grpc.tls.certFile }}
    {{- if $serviceTls }}
    {{- printf "" -}}
    {{- else }}
    {{- printf "true" -}}
    {{- end }}
{{- end }}

{{- define "plgd-hub.grpcreflection.serviceCertName" -}}
  {{- $fullName := include "plgd-hub.grpcreflection.fullname" . -}}
  {{- printf "%s-crt" $fullName -}}
{{- end }}

{{- define "plgd-hub.grpcreflection.domainCertName" -}}
    {{- if .Values.grpcreflection.ingress.secretName }}
        {{- printf "%s" .Values.grpcreflection.ingress.secretName -}}
    {{- else }}
        {{- $fullName := include "plgd-hub.grpcreflection.fullname" . -}}
        {{- printf "%s-domain-crt" $fullName -}}
    {{- end }}
{{- end }}

{{- define "plgd-hub.grpcreflection.domain" -}}
  {{- if .Values.grpcreflection.domain }}
    {{- printf "%s" .Values.grpcreflection.domain }}
  {{- else }}
    {{- printf "api.%s" .Values.global.domain }}
  {{- end }}
{{- end }}

{{- define "plgd-hub.grpcreflection.selectorLabels" -}}
{{- $ := index . 0 }}
{{- $domain := index . 1 }}
app.kubernetes.io/name: {{ include "plgd-hub.grpcreflection.domainToName" (list $domain $.Values.grpcreflection.name) }}
app.kubernetes.io/instance: {{ $.Release.Name }}
{{- end }}
