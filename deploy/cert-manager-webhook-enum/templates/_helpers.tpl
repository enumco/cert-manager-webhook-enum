{{/* vim: set filetype=mustache: */}}
{{- define "cert-manager-webhook-enum.name" -}}
{{- default .Chart.Name .Values.nameOverride | trunc 63 | trimSuffix "-" -}}
{{- end -}}

{{/*
Create a default fully qualified app name.
We truncate at 63 chars because some Kubernetes name fields are limited to this
(by the DNS naming spec). If release name contains chart name it will be used
as a full name.
*/}}
{{- define "cert-manager-webhook-enum.fullname" -}}
{{- if .Values.fullnameOverride -}}
{{- .Values.fullnameOverride | trunc 63 | trimSuffix "-" -}}
{{- else -}}
{{- $name := default .Chart.Name .Values.nameOverride -}}
{{- if contains $name .Release.Name -}}
{{- .Release.Name | trunc 63 | trimSuffix "-" -}}
{{- else -}}
{{- printf "%s-%s" .Release.Name $name | trunc 63 | trimSuffix "-" -}}
{{- end -}}
{{- end -}}
{{- end -}}

{{- define "cert-manager-webhook-enum.chart" -}}
{{- printf "%s-%s" .Chart.Name .Chart.Version | replace "+" "_" | trunc 63 | trimSuffix "-" -}}
{{- end -}}

{{- define "cert-manager-webhook-enum.labels" -}}
helm.sh/chart: {{ include "cert-manager-webhook-enum.chart" . }}
{{ include "cert-manager-webhook-enum.selectorLabels" . }}
{{- if .Chart.AppVersion }}
app.kubernetes.io/version: {{ .Chart.AppVersion | quote }}
{{- end }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
{{- end -}}

{{- define "cert-manager-webhook-enum.selectorLabels" -}}
app.kubernetes.io/name: {{ include "cert-manager-webhook-enum.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
{{- end -}}

{{- define "cert-manager-webhook-enum.selfSignedIssuer" -}}
{{ printf "%s-selfsign" (include "cert-manager-webhook-enum.fullname" .) }}
{{- end -}}

{{- define "cert-manager-webhook-enum.rootCAIssuer" -}}
{{ printf "%s-ca" (include "cert-manager-webhook-enum.fullname" .) }}
{{- end -}}

{{- define "cert-manager-webhook-enum.rootCACertificate" -}}
{{ printf "%s-ca" (include "cert-manager-webhook-enum.fullname" .) }}
{{- end -}}

{{- define "cert-manager-webhook-enum.servingCertificate" -}}
{{ printf "%s-webhook-tls" (include "cert-manager-webhook-enum.fullname" .) }}
{{- end -}}
