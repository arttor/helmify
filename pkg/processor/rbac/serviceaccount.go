package rbac

import (
	"fmt"
	"io"

	"github.com/arttor/helmify/pkg/helmify"
	"github.com/arttor/helmify/pkg/processor"
	"github.com/iancoleman/strcase"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

const (
	saTempl = `{{- $sa := .Values.%[2]s.serviceAccount -}}
{{- if $sa.create -}}
apiVersion: v1
kind: ServiceAccount
metadata:
  name: {{ default (include "%[1]s.fullname" .) $sa.name }}
  labels:
  {{- include "%[1]s.labels" . | nindent 4 }}
  {{- with $sa.annotations }}
  annotations:
    {{- toYaml . | nindent 4 }}
  {{- end }}
automountServiceAccountToken: {{ $sa.automount }}
{{- end }}`
)

var serviceAccountGVC = schema.GroupVersionKind{
	Group:   "",
	Version: "v1",
	Kind:    "ServiceAccount",
}

// ServiceAccount creates processor for k8s ServiceAccount resource.
func ServiceAccount() helmify.Processor {
	return &serviceAccount{}
}

type serviceAccount struct{}

// Process k8s ServiceAccount object into helm template. Returns false if not capable of processing given resource type.
func (sa serviceAccount) Process(appMeta helmify.AppMetadata, obj *unstructured.Unstructured) (bool, helmify.Template, error) {
	if obj.GroupVersionKind() != serviceAccountGVC {
		return false, nil, nil
	}
	valueName := processor.ObjectValueName(appMeta, obj)
	nameCamel := strcase.ToLowerCamel(valueName)
	values := helmify.Values{}
	_, _ = values.Add(true, nameCamel, "serviceAccount", "create")
	_, _ = values.Add("", nameCamel, "serviceAccount", "name")
	_, _ = values.Add(true, nameCamel, "serviceAccount", "automount")
	valuesAnnotations := make(map[string]interface{})
	for k, v := range obj.GetAnnotations() {
		valuesAnnotations[k] = v
	}
	err := unstructured.SetNestedField(values, valuesAnnotations, nameCamel, "serviceAccount", "annotations")
	if err != nil {
		return true, nil, err
	}
	tmpl := saTempl
	meta := fmt.Sprintf(tmpl, appMeta.ChartName(), nameCamel)

	return true, &saResult{
		name:   valueName,
		data:   []byte(meta),
		values: values,
	}, nil
}

type saResult struct {
	name   string
	data   []byte
	values helmify.Values
}

func (r *saResult) Filename() string {
	return fmt.Sprintf("%s-serviceaccount.yaml", r.name)
}

func (r *saResult) Values() helmify.Values {
	return r.values
}

func (r *saResult) Write(writer io.Writer) error {
	_, err := writer.Write(r.data)
	return err
}
