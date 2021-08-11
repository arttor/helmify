package secret

import (
	"fmt"
	"github.com/arttor/helmify/pkg/helmify"
	"github.com/pkg/errors"
	"io"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"strings"
)

const (
	secretTempl = `apiVersion: v1
kind: Secret
metadata:
  name: {{ include "%[1]s.fullname" . }}-%[2]s
  labels:
  {{- include "%[1]s.labels" . | nindent 4 }}
data:
%[3]s`
)

var (
	configMapGVC = schema.GroupVersionKind{
		Group:   "",
		Version: "v1",
		Kind:    "Secret",
	}
)

func Secret() helmify.Processor {
	return &secret{}
}

type secret struct {
}

func (d secret) Process(info helmify.ChartInfo, obj *unstructured.Unstructured) (bool, helmify.Template, error) {
	if obj.GroupVersionKind() != configMapGVC {
		return false, nil, nil
	}
	secret := corev1.Secret{}
	err := runtime.DefaultUnstructuredConverter.FromUnstructured(obj.Object, &secret)
	if err != nil {
		return true, nil, errors.Wrap(err, "unable to cast to secret")
	}
	name := strings.TrimPrefix(secret.GetName(), info.OperatorName+"-")
	values := helmify.Values{}
	tmpl := ""
	if secret.Data != nil && len(secret.Data) != 0 {
		subValues := helmify.Values{}
		secretValues := helmify.Values{}
		for key := range secret.Data {
			secretValues[key] = ""
			valName := fmt.Sprintf("secrets.%s.%s", name, key)
			tmpl += fmt.Sprintf("  %s: {{ .Values.%s | b64enc }}\n", key, valName)
		}
		subValues[name] = secretValues
		values["secrets"] = subValues
	}
	res := fmt.Sprintf(secretTempl, info.ChartName, name, tmpl)

	return true, &result{
		name:   name + ".yaml",
		data:   []byte(res),
		values: values,
	}, nil
}

type result struct {
	name   string
	data   []byte
	values helmify.Values
}

func (r *result) Filename() string {
	return r.name
}

func (r *result) Values() helmify.Values {
	return r.values
}

func (r *result) Write(writer io.Writer) error {
	_, err := writer.Write(r.data)
	return err
}
