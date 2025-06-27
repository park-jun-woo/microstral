{{- define "microstral.fullname" -}}
{{- if and .Release.Name (ne .Release.Name "microstral") -}}
{{ .Release.Name }}-microstral
{{- else -}}
microstral
{{- end -}}
{{- end }}

{{- define "redis.fullname" -}}
{{- if and .Release.Name (ne .Release.Name "microstral") -}}
{{ .Release.Name }}-redis-microstral
{{- else -}}
redis-microstral
{{- end -}}
{{- end }}

{{- define "postgres.fullname" -}}
{{- if and .Release.Name (ne .Release.Name "microstral") -}}
{{ .Release.Name }}-postgres-microstral
{{- else -}}
postgres-microstral
{{- end -}}
{{- end }}

{{- define "postgres.pvcname" -}}
{{- if and .Release.Name (ne .Release.Name "microstral") -}}
{{ .Release.Name }}-postgres-microstral
{{- else -}}
postgres-microstral
{{- end -}}
{{- end }}

{{- define "microstral.config" -}}
{{ include "microstral.fullname" . }}-config
{{- end }}
