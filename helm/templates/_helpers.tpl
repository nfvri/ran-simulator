# SPDX-FileCopyrightText: 2022 2020-present Open Networking Foundation <info@opennetworking.org>
#
# SPDX-License-Identifier: Apache-2.0

{{/* vim: set filetype=mustache: */}}
{{/*
Expand the name of the chart.
*/}}
{{- define "ran-simulator.name" -}}
{{- default .Chart.Name .Values.nameOverride | trunc 63 | trimSuffix "-" -}}
{{- end -}}

{{/*
Create a default fully qualified app name.
We truncate at 63 chars because some Kubernetes name fields are limited to this (by the DNS naming spec).
If release name contains chart name it will be used as a full name.
*/}}
{{- define "ran-simulator.fullname" -}}
{{- if .Values.fullnameOverride -}}
{{- .Values.fullnameOverride | trunc 63 | trimSuffix "-" -}}
{{- else -}}
{{- $name := default .Chart.Name .Values.nameOverride -}}
{{- if contains $name .Release.Name -}}
{{- .Release.Name | trunc 63 | trimSuffix "-" -}}
{{- else -}}
{{- printf "%s-%s" .Release.Name $name | trunc 63 | trimSuffix "-" -}}
{{- end -}}
{{- end -}}
{{- end -}}

{{/*
Create chart name and version as used by the chart label.
*/}}
{{- define "ran-simulator.chart" -}}
{{- printf "%s-%s" .Chart.Name .Chart.Version | replace "+" "_" | trunc 63 | trimSuffix "-" -}}
{{- end -}}

{{/*
Common labels
*/}}
{{- define "ran-simulator.labels" -}}
helm.sh/chart: {{ include "ran-simulator.chart" . }}
{{ include "ran-simulator.selectorLabels" . }}
{{- if .Chart.AppVersion }}
app.kubernetes.io/version: {{ .Chart.AppVersion | quote }}
{{- end }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
{{- end -}}

{{/*
Selector labels
*/}}
{{- define "ran-simulator.selectorLabels" -}}
app.kubernetes.io/name: {{ include "ran-simulator.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
{{- end -}}

{{/*
ran-simulator image name
*/}}
{{- define "ran-simulator.container-image" -}}
  {{- printf "%s:%s" .Values.image.registry (default "latest" .Values.image.tag ) -}}
{{- end -}}

{{/*
Extracts the base registry URL by truncating the string up to the last "/".
*/}}
{{- define "ran-simulator.registryname" -}}
  {{- $registry := .Values.image.registry -}}
  {{- $parts := split "/" $registry -}}
  {{- $length := sub (len $parts) 1 -}}
  {{- $base := slice (splitList "/" $registry) 0 $length | join "/" | quote -}}
  {{- printf "%s/" $base | trimPrefix "/" -}}
{{- end -}}



{{/*
Create secret image pull credentials
*/}}
{{- define "ran-simulator.imagePullSecret" }}
{{- with .Values.imageCredentials }}
{{- printf "{\"auths\":{\"%s\":{\"username\":\"%s\",\"password\":%s,\"email\":\"%s\",\"auth\":\"%s\"}}}" .registry .username ( .password | toJson ) .email (printf "%s:%s" .username .password | b64enc) | b64enc }}
{{- end }}
{{- end }}