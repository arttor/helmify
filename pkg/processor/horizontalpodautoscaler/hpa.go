package horizontalpodautoscaler

import (
	"bytes"
	"fmt"
	"io"

	"github.com/arttor/helmify/pkg/processor"

	"github.com/arttor/helmify/pkg/helmify"
	yamlformat "github.com/arttor/helmify/pkg/yaml"
	"github.com/iancoleman/strcase"
	autoscalingv1 "k8s.io/api/autoscaling/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/yaml"
)

const (
	hpaTempSpec = `
spec:
  scaleTargetRef:
%[1]s
  minReplicas: {{ .Values.%[2]s.minReplicas }}
  maxReplicas: {{ .Values.%[2]s.maxReplicas }}
  targetCPUUtilizationPercentage: {{ .Values.%[2]s.targetCPUUtilizationPercentage }}`
)

var hpaGVC = schema.GroupVersionKind{
	Group:   "autoscaling",
	Version: "v1",
	Kind:    "HorizontalPodAutoscaler",
}

// New creates processor for k8s Service resource.
func New() helmify.Processor {
	return &hpa{}
}

type hpa struct{}

// Process k8s Service object into template. Returns false if not capable of processing given resource type.
func (r hpa) Process(appMeta helmify.AppMetadata, obj *unstructured.Unstructured) (bool, helmify.Template, error) {
	if obj.GroupVersionKind() != hpaGVC {
		return false, nil, nil
	}
	hpa := autoscalingv1.HorizontalPodAutoscaler{}
	err := runtime.DefaultUnstructuredConverter.FromUnstructured(obj.Object, &hpa)
	if err != nil {
		return true, nil, fmt.Errorf("%w: unable to cast to hpa", err)
	}
	spec := hpa.Spec
	values := helmify.Values{}

	meta, err := processor.ProcessObjMeta(appMeta, obj)
	if err != nil {
		return true, nil, err
	}

	name := appMeta.TrimName(obj.GetName())
	nameCamel := strcase.ToLowerCamel(name)

	scaleTargetRef, _ := yaml.Marshal(hpa.Spec.ScaleTargetRef)
	scaleTargetRef = yamlformat.Indent(scaleTargetRef, 4)
	scaleTargetRef = bytes.TrimRight(scaleTargetRef, "\n ")

	if spec.MinReplicas != nil && *spec.MinReplicas != 0 {
		_, err := values.Add(*spec.MinReplicas, nameCamel, "minReplicas")
		if err != nil {
			return true, nil, err
		}
	}

	if spec.MaxReplicas != 0 {
		_, err := values.Add(spec.MaxReplicas, nameCamel, "maxReplicas")
		if err != nil {
			return true, nil, err
		}
	}

	if spec.TargetCPUUtilizationPercentage != nil && *spec.TargetCPUUtilizationPercentage != 0 {
		_, err := values.Add(*spec.TargetCPUUtilizationPercentage, nameCamel, "targetCPUUtilizationPercentage")
		if err != nil {
			return true, nil, err
		}
	}

	res := meta + fmt.Sprintf(hpaTempSpec, scaleTargetRef, nameCamel)
	return true, &result{
		name:   name,
		data:   res,
		values: values,
	}, nil
}

type result struct {
	name   string
	data   string
	values helmify.Values
}

func (r *result) Filename() string {
	return r.name + ".yaml"
}

func (r *result) Values() helmify.Values {
	return r.values
}

func (r *result) Write(writer io.Writer) error {
	_, err := writer.Write([]byte(r.data))
	return err
}
