package rbac

import (
	"fmt"
	"io"

	"github.com/arttor/helmify/pkg/helmify"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

const (
	saTempl = `{{ if .Values.serviceAccount.create }}
apiVersion: v1
kind: ServiceAccount
metadata:
  name: {{ include "%[1]s.serviceAccountName" . }}
  labels:
  {{- include "%[1]s.labels" . | nindent 4 }}
  annotations:
  {{- toYaml .Values.serviceAccount.annotations | nindent 4 }}
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
	values := helmify.Values{}
	_, _ = values.Add(true, "serviceAccount", "create")
	_, _ = values.Add("", "serviceAccount", "name")
	valuesAnnotations := make(map[string]interface{})
	for k, v := range obj.GetAnnotations() {
		valuesAnnotations[k] = v
	}
	err := unstructured.SetNestedField(values, valuesAnnotations, "serviceAccount", "annotations")
	if err != nil {
		return true, nil, err
	}
	tmpl := saTempl
	meta := fmt.Sprintf(tmpl, appMeta.ChartName())

	return true, &saResult{
		data:   []byte(meta),
		values: values,
	}, nil
}

type saResult struct {
	data   []byte
	values helmify.Values
}

func (r *saResult) Filename() string {
	return "serviceaccount.yaml"
}

func (r *saResult) Values() helmify.Values {
	return r.values
}

func (r *saResult) Write(writer io.Writer) error {
	_, err := writer.Write(r.data)
	return err
}
