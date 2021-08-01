package rbac

import (
	"bytes"
	"github.com/arttor/helmify/pkg/context"
	yamlformat "github.com/arttor/helmify/pkg/yaml"
	"github.com/pkg/errors"
	"io"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/yaml"
	"strings"
)

const (
	clusterRoleBindingTempl = `apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRoleBinding
metadata:
  name: {{ include "<CHART_NAME>.fullname" . }}-<NAME>
  labels:
  {{- include "<CHART_NAME>.labels" . | nindent 4 }}
roleRef:
`
)

var (
	clusterRoleBindingGVC = schema.GroupVersionKind{
		Group:   "rbac.authorization.k8s.io",
		Version: "v1",
		Kind:    "ClusterRoleBinding",
	}
)

func ClusterRoleBinding() context.Processor {
	return &clusterRoleBinding{}
}

type clusterRoleBinding struct {
}

func (r clusterRoleBinding) Process(obj *unstructured.Unstructured) (bool, context.Template, error) {
	if obj.GroupVersionKind() != clusterRoleBindingGVC {
		return false, nil,  nil
	}

	rb:=rbacv1.ClusterRoleBinding{}
	err := runtime.DefaultUnstructuredConverter.FromUnstructured(obj.Object, &rb)
	if err != nil {
		return true, nil,  errors.Wrap(err, "unable to cast to RoleBinding")
	}
	var ns string
	for _,s:=range rb.Subjects{
		if s.Namespace!=""{
			ns=s.Namespace
			break
		}
	}

	prefix := strings.TrimSuffix(ns, "system")
	name := strings.TrimPrefix(obj.GetName(), prefix)
	res := strings.ReplaceAll(clusterRoleBindingTempl, "<NAME>", name)

	rb.RoleRef.Name = strings.ReplaceAll(rb.RoleRef.Name,prefix,`{{ include "<CHART_NAME>.fullname" . }}-`)
	rb.RoleRef.Name = strings.ReplaceAll(rb.RoleRef.Name,prefix,`{{ include "<CHART_NAME>.fullname" . }}-`)

	rules, _ := yaml.Marshal(&rb.RoleRef)
	rules = yamlformat.Indent(rules, 2)
	rules = bytes.TrimRight(rules, "\n ")
	res = res + string(rules) +"\nsubjects:\n"
	for i,s:=range rb.Subjects{
		s.Namespace="{{ .Release.Namespace }}"
		s.Name=strings.ReplaceAll(s.Name,prefix,`{{ include "<CHART_NAME>.fullname" . }}-`)
		rb.Subjects[i]=s
	}
	subjects, _ := yaml.Marshal(&rb.Subjects)
	subjects = yamlformat.Indent(subjects, 2)
	subjects = bytes.TrimRight(subjects, "\n ")
	res = res + string(subjects)

	return true, &crbResult{
		name: name,
		data: []byte(res),
	}, nil
}
type crbResult struct {
	name      string
	data      []byte
	chartName string
}

func (r *crbResult) Filename() string {
	return strings.TrimSuffix(r.name, "-rolebinding") + "-rbac.yaml"
}

func (r *crbResult) GVK() schema.GroupVersionKind {
	return clusterRoleBindingGVC
}

func (r *crbResult) Values() context.Values {
	return context.Values{}
}

func (r *crbResult) Write(writer io.Writer) error {
	_, err := writer.Write(bytes.ReplaceAll(r.data, []byte("<CHART_NAME>"), []byte(r.chartName)))
	return err
}

func (r *crbResult) PostProcess(data context.Data) {
}

func (r *crbResult) SetChartName(name string) {
	r.chartName = name
}
