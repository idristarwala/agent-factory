{{- define "agent-factory.name" -}}
{{- default .Chart.Name .Values.nameOverride | trunc 63 | trimSuffix "-" }}
{{- end }}

{{- define "agent-factory.fullname" -}}
{{- printf "%s-%s" .Release.Name (include "agent-factory.name" .) | trunc 63 | trimSuffix "-" }}
{{- end }}
