package helmify

import (
	"github.com/imdario/mergo"
	"github.com/pkg/errors"
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

// Template - represents Helm template in 'templates' directory
type Template interface {
	// Filename - returns template filename
	Filename() string
	// Values - returns set of values used in template
	Values() Values
	// Write - writes helm template into given writer
	Write(writer io.Writer) error
}

// Values - represents helm template values.yaml
type Values map[string]interface{}

// Merge given values with current instance.
func (v *Values) Merge(values Values) error {
	if err := mergo.Merge(v, values, mergo.WithAppendSlice); err != nil {
		return errors.Wrap(err, "unable to merge helm values")
	}
	return nil
}

// Output - converts Template into helm chart on disk
type Output interface {
	Create(chartInfo ChartInfo, templates []Template) error
}

// ChartInfo general chart information
type ChartInfo struct {
	// ChartName - name of the directory of the helm chart
	ChartName string
	// Name of the operator. Also equals to name in Chart.yaml
	OperatorName string
	// OperatorNamespace namespace of operator. Not used in resulted chart. Need only for correct templates processing.
	OperatorNamespace string
}
