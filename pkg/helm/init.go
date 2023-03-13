package helm

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

const helmIgnore = `# Patterns to ignore when building packages.
# This supports shell glob matching, relative path matching, and
# negation (prefixed with !). Only one pattern per line.
.DS_Store
# Common VCS dirs
.git/
.gitignore
.bzr/
.bzrignore
.hg/
.hgignore
.svn/
# Common backup files
*.swp
*.bak
*.tmp
*.orig
*~
# Various IDEs
.project
.idea/
*.tmproj
.vscode/
`

const defaultHelpers = `{{/*
Expand the name of the chart.
*/}}
{{- define "<CHARTNAME>.name" -}}
{{- default .Chart.Name .Values.nameOverride | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Create a default fully qualified app name.
We truncate at 63 chars because some Kubernetes name fields are limited to this (by the DNS naming spec).
If release name contains chart name it will be used as a full name.
*/}}
{{- define "<CHARTNAME>.fullname" -}}
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
Create chart name and version as used by the chart label.
*/}}
{{- define "<CHARTNAME>.chart" -}}
{{- printf "%s-%s" .Chart.Name .Chart.Version | replace "+" "_" | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Common labels
*/}}
{{- define "<CHARTNAME>.labels" -}}
helm.sh/chart: {{ include "<CHARTNAME>.chart" . }}
{{ include "<CHARTNAME>.selectorLabels" . }}
{{- if .Chart.AppVersion }}
app.kubernetes.io/version: {{ .Chart.AppVersion | quote }}
{{- end }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
{{- end }}

{{/*
Selector labels
*/}}
{{- define "<CHARTNAME>.selectorLabels" -}}
app.kubernetes.io/name: {{ include "<CHARTNAME>.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
{{- end }}

{{/*
Create the name of the service account to use
*/}}
{{- define "<CHARTNAME>.serviceAccountName" -}}
{{- if .Values.serviceAccount.create }}
{{- default (include "<CHARTNAME>.fullname" .) .Values.serviceAccount.name }}
{{- else }}
{{- default "default" .Values.serviceAccount.name }}
{{- end }}
{{- end }}

{{/*
Renders a value that contains template.
Usage:
{{ include "tplvalues.render" ( dict "value" .Values.path.to.the.Value "context" $) }}
*/}}
{{- define "tplvalues.render" -}}
    {{- if typeIs "string" .value }}
        {{- tpl .value .context }}
    {{- else }}
        {{- tpl (.value | toYaml) .context }}
    {{- end }}
{{- end -}}

`

const defaultChartfile = `apiVersion: v2
name: %s
description: A Helm chart for Kubernetes
# A chart can be either an 'application' or a 'library' chart.
#
# Application charts are a collection of templates that can be packaged into versioned archives
# to be deployed.
#
# Library charts provide useful utilities or functions for the chart developer. They're included as
# a dependency of application charts to inject those utilities and functions into the rendering
# pipeline. Library charts do not define any templates and therefore cannot be deployed.
type: application
# This is the chart version. This version number should be incremented each time you make changes
# to the chart and its templates, including the app version.
# Versions are expected to follow Semantic Versioning (https://semver.org/)
version: 0.1.0
# This is the version number of the application being deployed. This version number should be
# incremented each time you make changes to the application. Versions are not expected to
# follow Semantic Versioning. They should reflect the version the application is using.
# It is recommended to use it with quotes.
appVersion: "0.1.0"
`

var chartName = regexp.MustCompile("^[a-zA-Z0-9._-]+$")

const maxChartNameLength = 250

// initChartDir - creates Helm chart structure in chartName directory if not presented.
func initChartDir(chartDir, chartName string, crd bool) error {
	if err := validateChartName(chartName); err != nil {
		return err
	}

	cDir := filepath.Join(chartDir, chartName)
	_, err := os.Stat(filepath.Join(cDir, "Chart.yaml"))
	if os.IsNotExist(err) {
		return createCommonFiles(chartDir, chartName, crd)
	}
	logrus.Info("Skip creating Chart skeleton: Chart.yaml already exists.")
	return err
}

func validateChartName(name string) error {
	if name == "" || len(name) > maxChartNameLength {
		return fmt.Errorf("chart name must be between 1 and %d characters", maxChartNameLength)
	}
	if !chartName.MatchString(name) {
		return fmt.Errorf("chart name must match the regular expression %q", chartName.String())
	}
	return nil
}

func createCommonFiles(chartDir, chartName string, crd bool) error {
	cDir := filepath.Join(chartDir, chartName)
	err := os.MkdirAll(filepath.Join(cDir, "templates"), 0750)
	if err != nil {
		return errors.Wrap(err, "unable create chart/templates dir")
	}
	if crd {
		err = os.MkdirAll(filepath.Join(cDir, "crds"), 0750)
		if err != nil {
			return errors.Wrap(err, "unable create crds dir")
		}
	}
	createFile := func(content []byte, path ...string) {
		if err != nil {
			return
		}
		file := filepath.Join(path...)
		err = ioutil.WriteFile(file, content, 0640)
		if err == nil {
			logrus.WithField("file", file).Info("created")
		}
	}
	createFile(chartYAML(chartName), cDir, "Chart.yaml")
	createFile([]byte(helmIgnore), cDir, ".helmignore")
	createFile(helpersYAML(chartName), cDir, "templates", "_helpers.tpl")
	return err
}

func chartYAML(appName string) []byte {
	return []byte(fmt.Sprintf(defaultChartfile, appName))
}

func helpersYAML(chartName string) []byte {
	return []byte(strings.ReplaceAll(defaultHelpers, "<CHARTNAME>", chartName))
}
