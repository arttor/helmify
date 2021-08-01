package rbac

import (
	"bytes"
	"github.com/arttor/helmify/pkg/context"
	yamlformat "github.com/arttor/helmify/pkg/yaml"
	"io"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/yaml"
	"strings"
)

const (
	roleTempl = `apiVersion: rbac.authorization.k8s.io/v1
kind: Role
metadata:
  name: {{ include "<CHART_NAME>.fullname" . }}-<NAME>
  labels:
  {{- include "<CHART_NAME>.labels" . | nindent 4 }}
rules:
`
)

var (
	roleGVC = schema.GroupVersionKind{
		Group:   "rbac.authorization.k8s.io",
		Version: "v1",
		Kind:    "Role",
	}
)

func Role() context.Processor {
	return &role{}
}

type role struct {
}

func (r role) Process(obj *unstructured.Unstructured) (bool, context.Template, error) {
	if obj.GroupVersionKind() != roleGVC {
		return false, nil,  nil
	}
	prefix := strings.TrimSuffix(obj.GetNamespace(), "system")
	name := strings.TrimPrefix(obj.GetName(), prefix)
	res := strings.ReplaceAll(roleTempl, "<NAME>", name)

	rules, _ := yaml.Marshal(obj.Object["rules"])
	rules = yamlformat.Indent(rules, 2)
	rules = bytes.TrimRight(rules, "\n ")
	res = res + string(rules)

	return true, &rResult{
		name: name,
		data: res,
	}, nil
}

type rResult struct {
	name      string
	data      string
	chartName string
}

func (r *rResult) Filename() string {
	return strings.TrimSuffix(r.name, "-role") + "-rbac.yaml"
}

func (r *rResult) GVK() schema.GroupVersionKind {
	return roleGVC
}

func (r *rResult) Values() context.Values {
	return context.Values{}
}

func (r *rResult) Write(writer io.Writer) error {
	_, err := writer.Write([]byte(strings.ReplaceAll(r.data, "<CHART_NAME>", r.chartName)))
	return err
}

func (r *rResult) PostProcess(data context.Data) {
	crds, ok, err := unstructured.NestedMap(data.Values(), "crd")
	if err != nil || !ok {
		return
	}
	for k, v := range crds {
		group := v.(map[string]interface{})["group"].(string)
		plural := v.(map[string]interface{})["plural"].(string)
		r.data = strings.ReplaceAll(r.data, group,"{{ .Values.crd."+k+".group }}")
		r.data = strings.ReplaceAll(r.data,plural,"{{ .Values.crd."+k+".plural }}")
	}
}

func (r *rResult) SetChartName(name string) {
	r.chartName = name
}
