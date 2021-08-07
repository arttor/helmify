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
	clusterRoleBindingTempl = `apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRoleBinding
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
	clusterRoleBindingGVC = schema.GroupVersionKind{
		Group:   "rbac.authorization.k8s.io",
		Version: "v1",
		Kind:    "ClusterRoleBinding",
	}
)

func ClusterRoleBinding() helmify.Processor {
	return &clusterRoleBinding{}
}

type clusterRoleBinding struct {
}

func (r clusterRoleBinding) Process(info helmify.ChartInfo, obj *unstructured.Unstructured) (bool, helmify.Template, error) {
	if obj.GroupVersionKind() != clusterRoleBindingGVC {
		return false, nil, nil
	}

	rb := rbacv1.ClusterRoleBinding{}
	err := runtime.DefaultUnstructuredConverter.FromUnstructured(obj.Object, &rb)
	if err != nil {
		return true, nil, errors.Wrap(err, "unable to cast to RoleBinding")
	}

	name := strings.TrimPrefix(obj.GetName(), info.OperatorName+"-")

	fullNameTempl := fmt.Sprintf(`{{ include "%s.fullname" . }}`, info.ChartName)
	rb.RoleRef.Name = strings.ReplaceAll(rb.RoleRef.Name, info.OperatorName, fullNameTempl)

	roleRef, _ := yaml.Marshal(&rb.RoleRef)
	roleRef = yamlformat.Indent(roleRef, 2)
	roleRef = bytes.TrimRight(roleRef, "\n ")

	for i, s := range rb.Subjects {
		s.Namespace = "{{ .Release.Namespace }}"
		s.Name = strings.ReplaceAll(s.Name, info.OperatorName, fullNameTempl)
		rb.Subjects[i] = s
	}
	subjects, _ := yaml.Marshal(&rb.Subjects)
	subjects = yamlformat.Indent(subjects, 2)
	subjects = bytes.TrimRight(subjects, "\n ")
	res := fmt.Sprintf(clusterRoleBindingTempl, info.ChartName, name, string(roleRef), string(subjects))

	return true, &crbResult{
		name: name,
		data: []byte(res),
	}, nil
}

type crbResult struct {
	name string
	data []byte
}

func (r *crbResult) Filename() string {
	return strings.TrimSuffix(r.name, "-rolebinding") + "-rbac.yaml"
}

func (r *crbResult) Values() helmify.Values {
	return helmify.Values{}
}

func (r *crbResult) Write(writer io.Writer) error {
	_, err := writer.Write(r.data)
	return err
}
