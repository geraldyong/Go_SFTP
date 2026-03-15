{{- define "sftp.name" -}}
{{- default .Chart.Name .Values.global.nameOverride | trunc 63 | trimSuffix "-" -}}
{{- end -}}

{{- define "sftp.fullname" -}}
{{- if .Values.global.fullnameOverride -}}
{{- .Values.global.fullnameOverride | trunc 63 | trimSuffix "-" -}}
{{- else -}}
{{- printf "%s-%s" .Release.Name (include "sftp.name" .) | trunc 63 | trimSuffix "-" -}}
{{- end -}}
{{- end -}}

{{- define "sftp.labels" -}}
app.kubernetes.io/name: {{ include "sftp.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
{{- end -}}

{{- define "sftp.selectorLabels" -}}
app.kubernetes.io/name: {{ include "sftp.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
{{- end -}}

{{- define "sftp.dataPvcName" -}}
{{- if .Values.storage.data.existingClaim -}}
{{- .Values.storage.data.existingClaim -}}
{{- else -}}
{{- printf "%s-data" (include "sftp.fullname" .) -}}
{{- end -}}
{{- end -}}

{{- define "sftp.keysPvcName" -}}
{{- if .Values.storage.keys.existingClaim -}}
{{- .Values.storage.keys.existingClaim -}}
{{- else -}}
{{- printf "%s-keys" (include "sftp.fullname" .) -}}
{{- end -}}
{{- end -}}

{{- define "sftp.vaultSvc" -}}
{{- printf "%s-vault" (include "sftp.fullname" .) -}}
{{- end -}}
