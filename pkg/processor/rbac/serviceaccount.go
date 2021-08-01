package rbac

import (
	"bytes"
	"github.com/arttor/helmify/pkg/context"
	"io"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"strings"
)

const (
	serviceAccountTempl = `apiVersion: v1
kind: ServiceAccount
metadata:
  name: {{ include "<CHART_NAME>.fullname" . }}-<NAME>
  labels:
  {{- include "<CHART_NAME>.labels" . | nindent 4 }}`
)

var (
	serviceAccountGVC = schema.GroupVersionKind{
		Group:   "",
		Version: "v1",
		Kind:    "ServiceAccount",
	}
)

func ServiceAccount() context.Processor {
	return &serviceAccount{}
}

type serviceAccount struct {
}

func (sa serviceAccount) Process(obj *unstructured.Unstructured) (bool, context.Template, error) {
	if obj.GroupVersionKind() != serviceAccountGVC {
		return false, nil, nil
	}
	prefix := strings.TrimSuffix(obj.GetNamespace(), "system")
	name := strings.TrimPrefix(obj.GetName(), prefix)
	res := strings.ReplaceAll(serviceAccountTempl, "<NAME>", name)
	return true, &saResult{
		data: []byte(res),
	}, nil
}


type saResult struct {
	data      []byte
	chartName string
}

func (r *saResult) Filename() string {
	return "deployment.yaml"
}

func (r *saResult) GVK() schema.GroupVersionKind {
	return serviceAccountGVC
}

func (r *saResult) Values() context.Values {
	return context.Values{}
}

func (r *saResult) Write(writer io.Writer) error {
	_, err := writer.Write(bytes.ReplaceAll(r.data, []byte("<CHART_NAME>"), []byte(r.chartName)))
	return err
}

func (r *saResult) PostProcess(data context.Data) {
}

func (r *saResult) SetChartName(name string) {
	r.chartName = name
}
