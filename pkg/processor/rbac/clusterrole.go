package rbac

import (
	"bytes"
	"fmt"
	"github.com/arttor/helmify/pkg/helmify"
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
  name: {{ include "%[1]s.fullname" . }}-%[2]s
  labels:
  {{- include "%[1]s.labels" . | nindent 4 }}
rules:
%[3]s`
)

var (
	clusterRoleGVC = schema.GroupVersionKind{
		Group:   "rbac.authorization.k8s.io",
		Version: "v1",
		Kind:    "ClusterRole",
	}
)

func ClusterRole() helmify.Processor {
	return &clusterRole{}
}

type clusterRole struct {
}

func (r clusterRole) Process(info helmify.ChartInfo, obj *unstructured.Unstructured) (bool, helmify.Template, error) {
	if obj.GroupVersionKind() != clusterRoleGVC {
		return false, nil, nil
	}
	rules, _ := yaml.Marshal(obj.Object["rules"])
	rules = yamlformat.Indent(rules, 2)
	rules = bytes.TrimRight(rules, "\n ")
	name := strings.TrimPrefix(obj.GetName(), info.OperatorName+"-")
	res := fmt.Sprintf(clusterRoleTempl, info.ChartName, name, string(rules))
	return true, &crResult{
		name: name,
		data: res,
	}, nil
}

type crResult struct {
	name string
	data string
}

func (r *crResult) Filename() string {
	return strings.TrimSuffix(r.name, "-role") + "-rbac.yaml"
}

func (r *crResult) GVK() schema.GroupVersionKind {
	return clusterRoleGVC
}

func (r *crResult) Values() helmify.Values {
	return helmify.Values{}
}

func (r *crResult) Write(writer io.Writer) error {
	_, err := writer.Write([]byte(r.data))
	return err
}
