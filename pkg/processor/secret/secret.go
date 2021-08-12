package secret

import (
	"bytes"
	"fmt"
	"github.com/arttor/helmify/pkg/helmify"
	yamlformat "github.com/arttor/helmify/pkg/yaml"
	"github.com/iancoleman/strcase"
	"github.com/pkg/errors"
	"io"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/yaml"
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
	sec := corev1.Secret{}
	err := runtime.DefaultUnstructuredConverter.FromUnstructured(obj.Object, &sec)
	if err != nil {
		return true, nil, errors.Wrap(err, "unable to cast to secret")
	}
	name := strings.TrimPrefix(sec.GetName(), info.OperatorName+"-")
	nameCamelCase := strcase.ToLowerCamel(name)
	values := helmify.Values{}
	templatedData := map[string]string{}
	for key, _ := range sec.Data {
		keyCamelCase := strcase.ToLowerCamel(key)
		if key == strings.ToUpper(key) {
			keyCamelCase = strcase.ToLowerCamel(strings.ToLower(key))
		}
		err = unstructured.SetNestedField(values, "", nameCamelCase, keyCamelCase)
		if err != nil {
			return true, nil, errors.Wrap(err, "unable add secret to values")
		}
		templatedData[key] = fmt.Sprintf(`{{ required "secret %[1]s.%[2]s is required" .Values.%[1]s.%[2]s | b64enc }}`, nameCamelCase, keyCamelCase)
	}
	data, _ := yaml.Marshal(templatedData)
	data = yamlformat.Indent(data, 2)
	data = bytes.TrimRight(data, "\n ")
	res := fmt.Sprintf(secretTempl, info.ChartName, name, string(data))

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
