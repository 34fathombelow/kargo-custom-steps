{{- define "app.labels" -}}
app.kubernetes.io/name: guestbook
app.kubernetes.io/instance: {{ .Release.Name }}
app.kubernetes.io/version: {{ .Values.image.tag | quote }}
{{- end }}

{{- define "app.selectorLabels" -}}
app.kubernetes.io/name: guestbook
app.kubernetes.io/instance: {{ .Release.Name }}
{{- end }}
