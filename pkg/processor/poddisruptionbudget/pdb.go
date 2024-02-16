package poddisruptionbudget

import (
	"bytes"
	"fmt"
	"io"

	"github.com/arttor/helmify/pkg/helmify"
	"github.com/arttor/helmify/pkg/processor"
	yamlformat "github.com/arttor/helmify/pkg/yaml"
	"github.com/iancoleman/strcase"
	policyv1 "k8s.io/api/policy/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/yaml"
)

const (
	pdbTempSpec = `
spec:
  %[1]s
  selector:
%[2]s
    {{- include "%[3]s.selectorLabels" . | nindent 6 }}`
)

var pdbGVC = schema.GroupVersionKind{
	Group:   "policy",
	Version: "v1",
	Kind:    "PodDisruptionBudget",
}

func New() helmify.Processor {
	return &pdb{}
}

type pdb struct{}

func (r pdb) Process(appMeta helmify.AppMetadata, obj *unstructured.Unstructured) (bool, helmify.Template, error) {
	if obj.GroupVersionKind() != pdbGVC {
		return false, nil, nil
	}
	pdb := policyv1.PodDisruptionBudget{}
	err := runtime.DefaultUnstructuredConverter.FromUnstructured(obj.Object, &pdb)
	if err != nil {
		return true, nil, fmt.Errorf("%w: unable to cast to pdb", err)
	}

	// Extract the name and namespace for use in the error message
	name := pdb.GetName()
	namespace := pdb.GetNamespace()
	if namespace == "" {
		namespace = "default" // Assuming 'default' if no namespace is specified
	}

	// Check if both MinAvailable and MaxUnavailable are specified
	if pdb.Spec.MinAvailable != nil && pdb.Spec.MaxUnavailable != nil {
		return true, nil, fmt.Errorf("error in PodDisruptionBudget '%s' in namespace '%s': both MinAvailable and MaxUnavailable are specified, but only one is allowed", name, namespace)
	}

	meta, err := processor.ProcessObjMeta(appMeta, obj)
	if err != nil {
		return true, nil, err
	}

	nameCamel := strcase.ToLowerCamel(name)

	selector, _ := yaml.Marshal(pdb.Spec.Selector)
	selectorIndented := yamlformat.Indent(selector, 4)
	selectorIndented = bytes.TrimRight(selectorIndented, "\n ")

	values := helmify.Values{}
	specSection := ""

	if pdb.Spec.MaxUnavailable != nil {
		specSection = "maxUnavailable: {{ .Values." + nameCamel + ".maxUnavailable }}"
		_, err := values.Add(pdb.Spec.MaxUnavailable.IntValue(), nameCamel, "maxUnavailable")
		if err != nil {
			return true, nil, err
		}
	} else if pdb.Spec.MinAvailable != nil {
		specSection = "minAvailable: {{ .Values." + nameCamel + ".minAvailable }}"
		_, err := values.Add(pdb.Spec.MinAvailable.IntValue(), nameCamel, "minAvailable")
		if err != nil {
			return true, nil, err
		}
	}

	res := meta + fmt.Sprintf(pdbTempSpec, specSection, selectorIndented, appMeta.ChartName())

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
