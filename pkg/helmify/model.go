package helmify

import (
	"io"

	"github.com/arttor/helmify/pkg/config"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

// Processor - converts k8s object to helm template.
// Implement this interface and register it to a context to support a new k8s resource conversion.
type Processor interface {
	// Process - converts k8s object to Helm template.
	// return false if not able to process given object type.
	Process(appMeta AppMetadata, unstructured *unstructured.Unstructured) (bool, Template, error)
}

// Template - represents Helm template in 'templates' directory.
type Template interface {
	// Filename - returns template filename
	Filename() string
	// Values - returns set of values used in template
	Values() Values
	// Write - writes helm template into given writer
	Write(writer io.Writer) error
}

// Output - converts Template into helm chart on disk.
type Output interface {
	Create(chartName, chartDir string, Crd bool, certManagerAsSubchart bool, certManagerVersion string, certManagerInstallCRD bool, templates []Template, filenames []string) error
}

// AppMetadata handle common information about K8s objects in the chart.
type AppMetadata interface {
	// Namespace returns app namespace.
	Namespace() string
	// ChartName returns chart name
	ChartName() string
	// TemplatedName converts object name to templated Helm name.
	// Example: 	"my-app-service1"	-> "{{ include "chart.fullname" . }}-service1"
	//				"my-app-secret"		-> "{{ include "chart.fullname" . }}-secret"
	//				etc...
	TemplatedName(objName string) string
	// TemplatedString converts a string to templated string with chart name.
	TemplatedString(str string) string
	// TrimName trims common prefix from object name if exists.
	// We trim common prefix because helm already using release for this purpose.
	TrimName(objName string) string

	Config() config.Config

	// HasConfigMap returns true if a ConfigMap with the given name is part of the chart.
	HasConfigMap(name string) bool
	// HasSecret returns true if a Secret with the given name is part of the chart.
	HasSecret(name string) bool

	// ConfigMapFiles returns a map of ConfigMap object names to their template filenames.
	// Only populated when AddChecksumAnnotations is enabled.
	ConfigMapFiles() map[string]string
	// SecretFiles returns a map of Secret object names to their template filenames.
	// Only populated when AddChecksumAnnotations is enabled.
	SecretFiles() map[string]string
}
