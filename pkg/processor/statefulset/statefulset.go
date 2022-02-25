package statefulset

import (
	"fmt"
	"io"
	"text/template"

	"github.com/arttor/helmify/pkg/helmify"
	"github.com/arttor/helmify/pkg/processor"

	"github.com/iancoleman/strcase"
	"github.com/pkg/errors"

	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

var statefulsetGVC = schema.GroupVersionKind{
	Group:   "apps",
	Version: "v1",
	Kind:    "StatefulSet",
}

var statefulsetTempl, _ = template.New("statefulset").Parse(
	`{{- .Meta }}
spec:
{{- if .Replicas }}
{{ .Replicas }}
{{- end }}
  selector:
{{ .Selector }}
  template:
    metadata:
      labels:
{{ .PodLabels }}
{{- .PodAnnotations }}
    spec:
{{ .Spec }}`)

// New creates processor for k8s StatefulSet resource.
func New() helmify.Processor {
	return &statefulset{}
}

type statefulset struct{}

// Process k8s StatefulSet object into template. Returns false if not capable of processing given resource type.
func (d statefulset) Process(appMeta helmify.AppMetadata, obj *unstructured.Unstructured) (bool, helmify.Template, error) {
	if obj.GroupVersionKind() != statefulsetGVC {
		return false, nil, nil
	}
	typedObj := appsv1.StatefulSet{}
	err := runtime.DefaultUnstructuredConverter.FromUnstructured(obj.Object, &typedObj)
	if err != nil {
		return true, nil, errors.Wrap(err, "unable to cast to statefulset")
	}

	values := helmify.Values{}

	name := appMeta.TrimName(obj.GetName())

	meta, err := processor.ProcessObjMeta(appMeta, obj)
	if err != nil {
		return true, nil, err
	}

	replicas, err := processor.ProcessReplicas(name, typedObj.Spec.Replicas, &values)
	if err != nil {
		return true, nil, err
	}

	selector, err := processor.ProcessSelector(appMeta, typedObj.Spec.Selector)
	if err != nil {
		return true, nil, err
	}

	pod := processor.Pod{
		Name:    strcase.ToLowerCamel(name),
		AppMeta: appMeta,
		Pod:     &typedObj.Spec.Template,
	}

	podLabels, podAnnotations, err := pod.ProcessObjectMeta()
	if err != nil {
		return true, nil, err
	}
	podLabels += fmt.Sprintf("\n      {{- include \"%s.selectorLabels\" . | nindent 8 }}", appMeta.ChartName())

	spec, err := pod.ProcessSpec(&values)
	if err != nil {
		return true, nil, err
	}

	return true, &result{
		values: values,
		data: struct {
			Meta           string
			Replicas       string
			Selector       string
			PodLabels      string
			PodAnnotations string
			Spec           string
		}{
			Meta:           meta,
			Replicas:       replicas,
			Selector:       selector,
			PodLabels:      podLabels,
			PodAnnotations: podAnnotations,
			Spec:           spec,
		},
	}, nil
}

type result struct {
	data struct {
		Meta           string
		Replicas       string
		Selector       string
		PodLabels      string
		PodAnnotations string
		Spec           string
	}
	values helmify.Values
}

func (r *result) Filename() string {
	return "statefulset.yaml"
}

func (r *result) Values() helmify.Values {
	return r.values
}

func (r *result) Write(writer io.Writer) error {
	return statefulsetTempl.Execute(writer, r.data)
}
