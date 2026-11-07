{{/*
Chart name.
*/}}
{{- define "gitwebhookproxy.name" -}}
{{- default .Chart.Name .Values.nameOverride | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Fully qualified app name, truncated to 63 characters.
*/}}
{{- define "gitwebhookproxy.fullname" -}}
{{- if .Values.fullnameOverride }}
{{- .Values.fullnameOverride | trunc 63 | trimSuffix "-" }}
{{- else }}
{{- $name := default .Chart.Name .Values.nameOverride }}
{{- if contains $name .Release.Name }}
{{- .Release.Name | trunc 63 | trimSuffix "-" }}
{{- else }}
{{- printf "%s-%s" .Release.Name $name | trunc 63 | trimSuffix "-" }}
{{- end }}
{{- end }}
{{- end }}

{{/*
Chart name and version for the helm.sh/chart label.
*/}}
{{- define "gitwebhookproxy.chart" -}}
{{- printf "%s-%s" .Chart.Name .Chart.Version | replace "+" "_" | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Common labels.
*/}}
{{- define "gitwebhookproxy.labels" -}}
helm.sh/chart: {{ include "gitwebhookproxy.chart" . }}
{{ include "gitwebhookproxy.selectorLabels" . }}
app.kubernetes.io/version: {{ include "gitwebhookproxy.imageTag" . | trunc 63 | trimSuffix "-" | quote }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
{{- end }}

{{/*
Selector labels. These must not change between releases.
*/}}
{{- define "gitwebhookproxy.selectorLabels" -}}
app.kubernetes.io/name: {{ include "gitwebhookproxy.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
{{- end }}

{{/*
ServiceAccount name.
*/}}
{{- define "gitwebhookproxy.serviceAccountName" -}}
{{- if .Values.serviceAccount.create }}
{{- default (include "gitwebhookproxy.fullname" .) .Values.serviceAccount.name }}
{{- else }}
{{- default "default" .Values.serviceAccount.name }}
{{- end }}
{{- end }}

{{/*
Image tag. Defaults to "v" + appVersion.
*/}}
{{- define "gitwebhookproxy.imageTag" -}}
{{- default (printf "v%s" .Chart.AppVersion) .Values.image.tag }}
{{- end }}

{{/*
Full image reference. A digest takes precedence over the tag.
*/}}
{{- define "gitwebhookproxy.image" -}}
{{- if .Values.image.digest }}
{{- printf "%s@%s" .Values.image.repository .Values.image.digest }}
{{- else }}
{{- printf "%s:%s" .Values.image.repository (include "gitwebhookproxy.imageTag" .) }}
{{- end }}
{{- end }}

{{/*
Name of the Secret that holds the webhook secret, empty if none is configured.
*/}}
{{- define "gitwebhookproxy.secretName" -}}
{{- if .Values.secret.existingSecret }}
{{- .Values.secret.existingSecret }}
{{- else if .Values.secret.value }}
{{- include "gitwebhookproxy.fullname" . }}
{{- end }}
{{- end }}

{{/*
Key inside the webhook Secret.
*/}}
{{- define "gitwebhookproxy.secretKey" -}}
{{- if .Values.secret.existingSecret }}
{{- default "secret" .Values.secret.existingSecretKey }}
{{- else }}
{{- print "secret" }}
{{- end }}
{{- end }}

{{/*
GKE resource names.
*/}}
{{- define "gitwebhookproxy.backendConfigName" -}}
{{- default (printf "%s-backend" (include "gitwebhookproxy.fullname" .)) .Values.gke.backendConfig.name }}
{{- end }}

{{- define "gitwebhookproxy.frontendConfigName" -}}
{{- default (printf "%s-frontend" (include "gitwebhookproxy.fullname" .)) .Values.gke.frontendConfig.name }}
{{- end }}

{{/*
Input validation.
*/}}
{{- define "gitwebhookproxy.validate" -}}
{{- $_ := required "config.upstreamURL is required: set it to the URL that validated webhooks are forwarded to" .Values.config.upstreamURL }}
{{- if and .Values.secret.value .Values.secret.existingSecret }}
{{- fail "secret.value and secret.existingSecret are mutually exclusive: set only one" }}
{{- end }}
{{- if and .Values.gke.frontendConfig.enabled (not .Values.ingress.enabled) }}
{{- fail "gke.frontendConfig.enabled requires ingress.enabled=true" }}
{{- end }}
{{- end }}
