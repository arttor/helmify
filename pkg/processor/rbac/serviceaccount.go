package rbac

import (
	"fmt"
	"github.com/arttor/helmify/pkg/helmify"
	"io"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"strings"
)

const (
	serviceAccountTempl = `apiVersion: v1
kind: ServiceAccount
metadata:
  name: {{ include "%[1]s.fullname" . }}-%[2]s
  labels:
  {{- include "%[1]s.labels" . | nindent 4 }}`
)

var (
	serviceAccountGVC = schema.GroupVersionKind{
		Group:   "",
		Version: "v1",
		Kind:    "ServiceAccount",
	}
)

// ServiceAccount creates processor for k8s ServiceAccount resource.
func ServiceAccount() helmify.Processor {
	return &serviceAccount{}
}

type serviceAccount struct {
}

// Process k8s ServiceAccount object into helm template. Returns false if not capable of processing given resource type.
func (sa serviceAccount) Process(info helmify.ChartInfo, obj *unstructured.Unstructured) (bool, helmify.Template, error) {
	if obj.GroupVersionKind() != serviceAccountGVC {
		return false, nil, nil
	}
	name := strings.TrimPrefix(obj.GetName(), info.OperatorName+"-")
	res := fmt.Sprintf(serviceAccountTempl, info.ChartName, name)
	return true, &saResult{
		data: []byte(res),
	}, nil
}

type saResult struct {
	data []byte
}

func (r *saResult) Filename() string {
	return "deployment.yaml"
}

func (r *saResult) Values() helmify.Values {
	return helmify.Values{}
}

func (r *saResult) Write(writer io.Writer) error {
	_, err := writer.Write(r.data)
	return err
}
