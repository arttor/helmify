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
	roleBindingTempl = `apiVersion: rbac.authorization.k8s.io/v1
kind: RoleBinding
metadata:
  name: {{ include "<CHART_NAME>.fullname" . }}-<NAME>
  labels:
  {{- include "<CHART_NAME>.labels" . | nindent 4 }}
roleRef:
`
)

var (
	roleBindingGVC = schema.GroupVersionKind{
		Group:   "rbac.authorization.k8s.io",
		Version: "v1",
		Kind:    "RoleBinding",
	}
)

func RoleBinding() context.Processor {
	return &roleBinding{}
}

type roleBinding struct {
}

func (r roleBinding) Process(obj *unstructured.Unstructured) (bool, context.Template, error) {
	if obj.GroupVersionKind() != roleBindingGVC {
		return false, nil,  nil
	}
	prefix := strings.TrimSuffix(obj.GetNamespace(), "system")
	name := strings.TrimPrefix(obj.GetName(), prefix)
	res := strings.ReplaceAll(roleBindingTempl, "<NAME>", name)

	rb:=rbacv1.RoleBinding{}
	err := runtime.DefaultUnstructuredConverter.FromUnstructured(obj.Object, &rb)
	if err != nil {
		return true, nil,  errors.Wrap(err, "unable to cast to RoleBinding")
	}
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

	return true, &rbResult{
		name: name,
		data: []byte(res),
	}, nil
}

type rbResult struct {
	name      string
	data      []byte
	chartName string
}

func (r *rbResult) Filename() string {
	return strings.TrimSuffix(r.name, "-rolebinding") + "-rbac.yaml"
}

func (r *rbResult) GVK() schema.GroupVersionKind {
	return roleBindingGVC
}

func (r *rbResult) Values() context.Values {
	return context.Values{}
}

func (r *rbResult) Write(writer io.Writer) error {
	_, err := writer.Write(bytes.ReplaceAll(r.data, []byte("<CHART_NAME>"), []byte(r.chartName)))
	return err
}

func (r *rbResult) PostProcess(data context.Data) {
}

func (r *rbResult) SetChartName(name string) {
	r.chartName = name
}
