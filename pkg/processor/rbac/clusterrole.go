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
	clusterRoleTempl = `apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRole
metadata:
  name: <NAME>
  labels:
  {{- include "<CHART_NAME>.labels" . | nindent 4 }}
rules:
`
)

var (
	clusterRoleGVC = schema.GroupVersionKind{
		Group:   "rbac.authorization.k8s.io",
		Version: "v1",
		Kind:    "ClusterRole",
	}
)

func ClusterRole() context.Processor {
	return &clusterRole{}
}

type clusterRole struct {
}

func (r clusterRole) Process(obj *unstructured.Unstructured) (bool, context.Template, error) {
	if obj.GroupVersionKind() != clusterRoleGVC {
		return false, nil, nil
	}

	rules, _ := yaml.Marshal(obj.Object["rules"])
	rules = yamlformat.Indent(rules, 2)
	rules = bytes.TrimRight(rules, "\n ")
	res := clusterRoleTempl + string(rules)
	return true, &crResult{
		name: obj.GetName(),
		data: res,
	}, nil
}

type crResult struct {
	name      string
	data      string
	chartName string
}

func (r *crResult) Filename() string {
	return strings.TrimSuffix(r.name, "-role") + "-rbac.yaml"
}

func (r *crResult) GVK() schema.GroupVersionKind {
	return clusterRoleGVC
}

func (r *crResult) Values() context.Values {
	return context.Values{}
}

func (r *crResult) Write(writer io.Writer) error {
	_, err := writer.Write([]byte(strings.ReplaceAll(r.data, "<CHART_NAME>", r.chartName)))
	return err
}

func (r *crResult) PostProcess(data context.Data) {
	r.name = strings.TrimPrefix(r.name, data.Name()+"-")
	r.data = strings.ReplaceAll(r.data, "<NAME>", `{{ include "<CHART_NAME>.fullname" . }}-`+r.name)
	crds, ok, err := unstructured.NestedMap(data.Values(), "crd")
	if err != nil || !ok {
		return
	}
	for k, v := range crds {
		group := v.(map[string]interface{})["group"].(string)
		plural := v.(map[string]interface{})["plural"].(string)
		r.data = strings.ReplaceAll(r.data, group, "{{ .Values.crd."+k+".group }}")
		r.data = strings.ReplaceAll(r.data, plural, "{{ .Values.crd."+k+".plural }}")
	}
}

func (r *crResult) SetChartName(name string) {
	r.chartName = name
}
