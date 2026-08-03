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
{{- if .config -}}{{ toYaml .config }}{{- end -}}
{{- $cors := .root.Values.cors -}}
{{- if $cors -}}
{{- if .config -}}
{{- "\n" -}}
{{- end -}}
cors:
  allowed_origins:
{{- range $cors.allowOrigins }}
    - {{ . | toJson }}
{{- end }}
  allowed_methods:
{{- range $cors.allowMethods }}
    - {{ . | toJson }}
{{- end }}
  allowed_headers:
{{- range $cors.allowHeaders }}
    - {{ . | toJson }}
{{- end }}
  max_age: {{ $cors.maxAge }}
  allow_credentials: {{ $cors.allowCredentials }}
{{- end -}}
{{- end -}}

{{- define "uptime-monitor.toYaml" -}}
{{- if . -}}
{{- toYaml . -}}
{{- end -}}
{{- end -}}
