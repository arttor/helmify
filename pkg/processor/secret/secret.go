package secret

import (
	"fmt"
	"github.com/arttor/helmify/pkg/processor"
	"io"
	"strings"
	"text/template"

	"github.com/arttor/helmify/pkg/helmify"
	yamlformat "github.com/arttor/helmify/pkg/yaml"
	"github.com/iancoleman/strcase"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

var secretTempl, _ = template.New("secret").Parse(
	`{{ .Meta }}
{{ .Data }}`)

var configMapGVC = schema.GroupVersionKind{
	Group:   "",
	Version: "v1",
	Kind:    "Secret",
}

// New creates processor for k8s Secret resource.
func New() helmify.Processor {
	return &secret{}
}

type secret struct{}

// Process k8s Secret object into template. Returns false if not capable of processing given resource type.
func (d secret) Process(info helmify.ChartInfo, obj *unstructured.Unstructured) (bool, helmify.Template, error) {
	if obj.GroupVersionKind() != configMapGVC {
		return false, nil, nil
	}
	sec := corev1.Secret{}
	err := runtime.DefaultUnstructuredConverter.FromUnstructured(obj.Object, &sec)
	if err != nil {
		return true, nil, errors.Wrap(err, "unable to cast to secret")
	}
	name, meta, err := processor.ProcessMetadata(info, obj)
	if err != nil {
		return true, nil, err
	}

	nameCamelCase := strcase.ToLowerCamel(name)
	values := helmify.Values{}
	templatedData := map[string]string{}
	for key := range sec.Data {
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

	data, err := yamlformat.Marshal(map[string]interface{}{"data": templatedData}, 0)
	if err != nil {
		return true, nil, err
	}

	return true, &result{
		name: name + ".yaml",
		data: struct {
			Meta string
			Data string
		}{Meta: meta, Data: data},
		values: values,
	}, nil
}

type result struct {
	name string
	data struct {
		Meta string
		Data string
	}
	values helmify.Values
}

func (r *result) Filename() string {
	return r.name
}

func (r *result) Values() helmify.Values {
	return r.values
}

func (r *result) Write(writer io.Writer) error {
	return secretTempl.Execute(writer, r.data)
}
