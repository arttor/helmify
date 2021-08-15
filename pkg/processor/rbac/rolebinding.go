package rbac

import (
	"bytes"
	"fmt"
	"github.com/arttor/helmify/pkg/helmify"
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
  name: {{ include "%[1]s.fullname" . }}-%[2]s
  labels:
  {{- include "%[1]s.labels" . | nindent 4 }}
roleRef:
%[3]s
subjects:
%[4]s`
)

var (
	roleBindingGVC = schema.GroupVersionKind{
		Group:   "rbac.authorization.k8s.io",
		Version: "v1",
		Kind:    "RoleBinding",
	}
)

// RoleBinding creates processor for k8s RoleBinding resource.
func RoleBinding() helmify.Processor {
	return &roleBinding{}
}

type roleBinding struct {
}

// Process k8s RoleBinding object into helm template. Returns false if not capable of processing given resource type.
func (r roleBinding) Process(info helmify.ChartInfo, obj *unstructured.Unstructured) (bool, helmify.Template, error) {
	if obj.GroupVersionKind() != roleBindingGVC {
		return false, nil, nil
	}

	name := strings.TrimPrefix(obj.GetName(), info.OperatorName+"-")
	rb := rbacv1.RoleBinding{}
	err := runtime.DefaultUnstructuredConverter.FromUnstructured(obj.Object, &rb)
	if err != nil {
		return true, nil, errors.Wrap(err, "unable to cast to RoleBinding")
	}
	fullNameTeml := fmt.Sprintf(`{{ include "%s.fullname" . }}`, info.ChartName)

	rb.RoleRef.Name = strings.ReplaceAll(rb.RoleRef.Name, info.OperatorName, fullNameTeml)

	roleRef, _ := yaml.Marshal(&rb.RoleRef)
	roleRef = yamlformat.Indent(roleRef, 2)
	roleRef = bytes.TrimRight(roleRef, "\n ")

	for i, s := range rb.Subjects {
		s.Namespace = "{{ .Release.Namespace }}"
		s.Name = strings.ReplaceAll(s.Name, info.OperatorName, fullNameTeml)
		rb.Subjects[i] = s
	}
	subjects, _ := yaml.Marshal(&rb.Subjects)
	subjects = yamlformat.Indent(subjects, 2)
	subjects = bytes.TrimRight(subjects, "\n ")
	res := fmt.Sprintf(roleBindingTempl, info.ChartName, name, string(roleRef), string(subjects))

	return true, &rbResult{
		name: name,
		data: []byte(res),
	}, nil
}

type rbResult struct {
	name string
	data []byte
}

func (r *rbResult) Filename() string {
	return strings.TrimSuffix(r.name, "-rolebinding") + "-rbac.yaml"
}

func (r *rbResult) Values() helmify.Values {
	return helmify.Values{}
}

func (r *rbResult) Write(writer io.Writer) error {
	_, err := writer.Write(r.data)
	return err
}
