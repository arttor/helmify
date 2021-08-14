package helm

import (
	"fmt"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

const defaultIgnore = `# Patterns to ignore when building packages.
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

// init Helm chart structure in chartName directory if not presented
func (o *output) init(chartName, appName string) error {
	if err := validateChartName(chartName); err != nil {
		return err
	}
	_, err := os.Stat(filepath.Join(chartName, "Chart.yaml"))
	if os.IsNotExist(err) {
		return createSkeleton(chartName, appName)
	}
	logrus.Info("Skip creating Chart skeleton: Chart.yaml already exists.")
	return err
}

// createSkeleton - creates helm chart skeleton:
//    chartName/
//    ├── .helmignore   	# Contains patterns to ignore when packaging Helm charts.
//    ├── Chart.yaml    	# Information about your chart
//    └── templates/    	# The template files
//        └── _helpers.tp   # Helm default template partials
func createSkeleton(chartName, appName string) error {
	logrus.Debug("Creating chart skeleton")
	if _, err := os.Stat(filepath.Join(chartName)); os.IsNotExist(err) {
		err = os.Mkdir(chartName, 0755)
		if err != nil {
			return errors.Wrap(err, "unable create chart dir")
		}
		logrus.Infof("Created './%s' chart directory", chartName)
	}

	err := os.WriteFile(filepath.Join(chartName, "Chart.yaml"), []byte(fmt.Sprintf(defaultChartfile, appName)), 0755)
	if err != nil {
		return errors.Wrap(err, "unable create Chart.yaml")
	}
	logrus.Infof("'./%s/Chart.yaml' created", chartName)

	err = os.WriteFile(filepath.Join(chartName, ".helmignore"), []byte(defaultIgnore), 0755)
	if err != nil {
		return errors.Wrap(err, "unable create .helmignorer")
	}
	logrus.Infof("'./%s/.helmignore' created", chartName)

	err = os.Mkdir(filepath.Join(chartName, "templates"), 0755)
	if err != nil {
		return errors.Wrap(err, "unable create templates dir")
	}
	logrus.Infof("'./%s/templates/' dir created", chartName)

	err = os.WriteFile(filepath.Join(chartName, "templates", "_helpers.tpl"), []byte(strings.ReplaceAll(defaultHelpers, "<CHARTNAME>", chartName)), 0755)
	if err != nil {
		return errors.Wrap(err, "unable create _helpers.tpl")
	}
	logrus.Infof("'./%s/templates/_helpers.tpl' created", chartName)
	return nil
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
