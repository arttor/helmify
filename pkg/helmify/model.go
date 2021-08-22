package helmify

import (
	"io"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

// Processor - converts k8s object to helm template.
// Implement this interface and register it to a context to support a new k8s resource conversion.
type Processor interface {
	// Process - converts k8s object to Helm template.
	// return false if not able to process given object type.
	Process(chartInfo ChartInfo, unstructured *unstructured.Unstructured) (bool, Template, error)
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
	Create(chartInfo ChartInfo, templates []Template) error
}

// ChartInfo general chart information.
type ChartInfo struct {
	// ChartName - name of the directory of the helm chart
	ChartName string
	// ApplicationName application name in Chart.yaml
	ApplicationName string
	// Namespace namespace of application. Not used in resulted chart. Need only for correct templates processing.
	Namespace string
}
