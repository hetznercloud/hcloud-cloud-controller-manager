{{- define "hcloud-cloud-controller-manager.name" -}}
{{- default $.Chart.Name $.Values.nameOverride | trunc 63 | trimSuffix "-" }}
{{- end }}

{{- define "hcloud-cloud-controller-manager.selectorLabels" -}}
app.kubernetes.io/name: {{ include "hcloud-cloud-controller-manager.name" $ }}
app.kubernetes.io/instance: {{ $.Release.Name }}
{{- end }}
