package rbac

import (
	"bytes"
	"fmt"
	"io"
	"strings"

	"github.com/arttor/helmify/pkg/helmify"
	yamlformat "github.com/arttor/helmify/pkg/yaml"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/yaml"
)

const (
	roleTempl = `apiVersion: rbac.authorization.k8s.io/v1
kind: Role
metadata:
  name: {{ include "%[1]s.fullname" . }}-%[2]s
  labels:
  {{- include "%[1]s.labels" . | nindent 4 }}
rules:
%[3]s`
)

var roleGVC = schema.GroupVersionKind{
	Group:   "rbac.authorization.k8s.io",
	Version: "v1",
	Kind:    "Role",
}

// Role creates processor for k8s Role resource.
func Role() helmify.Processor {
	return &role{}
}

type role struct{}

// Process k8s Role object into helm template. Returns false if not capable of processing given resource type.
func (r role) Process(info helmify.ChartInfo, obj *unstructured.Unstructured) (bool, helmify.Template, error) {
	if obj.GroupVersionKind() != roleGVC {
		return false, nil, nil
	}
	name := strings.TrimPrefix(obj.GetName(), info.ApplicationName+"-")

	rules, _ := yaml.Marshal(obj.Object["rules"])
	rules = yamlformat.Indent(rules, 2)
	rules = bytes.TrimRight(rules, "\n ")
	res := fmt.Sprintf(roleTempl, info.ChartName, name, string(rules))

	return true, &rResult{
		name: name,
		data: res,
	}, nil
}

type rResult struct {
	name string
	data string
}

func (r *rResult) Filename() string {
	return strings.TrimSuffix(r.name, "-role") + "-rbac.yaml"
}

func (r *rResult) Values() helmify.Values {
	return helmify.Values{}
}

func (r *rResult) Write(writer io.Writer) error {
	_, err := writer.Write([]byte(r.data))
	return err
}
