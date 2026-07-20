{{- define "uptime-monitor.namespace" -}}
{{- .Values.namespace | default "uptime-monitor" -}}
{{- end -}}

{{- define "uptime-monitor.fullname" -}}
{{- .name -}}
{{- end -}}

{{- define "uptime-monitor.labels" -}}
app.kubernetes.io/name: {{ .name }}
app.kubernetes.io/part-of: uptime-monitor
{{- end -}}

{{- define "uptime-monitor.image" -}}
{{- printf "%s/%s:%s" .root.Values.global.image.registry .name .root.Values.global.image.tag -}}
{{- end -}}

{{- define "uptime-monitor.config" -}}
{{- include "uptime-monitor.toYaml" .config -}}
{{- end -}}

{{- define "uptime-monitor.toYaml" -}}
{{- if . -}}
{{- toYaml . -}}
{{- end -}}
{{- end -}}
