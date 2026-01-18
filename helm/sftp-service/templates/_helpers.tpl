{{- define "sftp-service.name" -}}
{{- default .Chart.Name .Values.nameOverride | trunc 63 | trimSuffix "-" -}}
{{- end -}}

{{- define "sftp-service.fullname" -}}
{{- if .Values.fullnameOverride -}}
{{- .Values.fullnameOverride | trunc 63 | trimSuffix "-" -}}
{{- else -}}
{{- printf "%s-%s" .Release.Name (include "sftp-service.name" .) | trunc 63 | trimSuffix "-" -}}
{{- end -}}
{{- end -}}

{{- define "sftp-service.labels" -}}
app.kubernetes.io/name: {{ include "sftp-service.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
app.kubernetes.io/version: {{ .Chart.AppVersion | quote }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
{{- end -}}

{{- define "sftp-service.selectorLabels" -}}
app.kubernetes.io/name: {{ include "sftp-service.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
{{- end -}}

{{- define "sftp-service.serviceAccountName" -}}
{{- if .Values.serviceAccount.create -}}
{{- if .Values.serviceAccount.name -}}
{{- .Values.serviceAccount.name -}}
{{- else -}}
{{- include "sftp-service.fullname" . -}}
{{- end -}}
{{- else -}}
{{- default "default" .Values.serviceAccount.name -}}
{{- end -}}
{{- end -}}

{{- define "sftp-service.hostKeySecretName" -}}
{{- if .Values.hostKey.existingSecret -}}
{{- .Values.hostKey.existingSecret -}}
{{- else if and .Values.hostKey.create .Values.hostKey.secretName -}}
{{- .Values.hostKey.secretName -}}
{{- else -}}
{{- printf "%s-hostkey" (include "sftp-service.fullname" .) -}}
{{- end -}}
{{- end -}}

