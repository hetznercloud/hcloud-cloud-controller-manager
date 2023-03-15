{{- define "hcloud-cloud-controller-manager.name" -}}
{{- $.Values.nameOverride | default $.Chart.Name | trunc 63 | trimSuffix "-" }}
{{- end }}

{{- define "hcloud-cloud-controller-manager.selectorLabels" -}}
app.kubernetes.io/name: {{ include "hcloud-cloud-controller-manager.name" $ }}
app.kubernetes.io/instance: {{ $.Release.Name }}
{{- end }}
