package secret

import (
	"io"
	"strings"
	"text/template"

	"github.com/arttor/helmify/pkg/processor"

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
{{- if .Data }}
{{ .Data }}
{{- end }}
{{- if .StringData }}
{{ .StringData }}
{{- end }}
{{- if .Type }}
{{ .Type }}
{{- end }}`)

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
func (d secret) Process(appMeta helmify.AppMetadata, obj *unstructured.Unstructured) (bool, helmify.Template, error) {
	if obj.GroupVersionKind() != configMapGVC {
		return false, nil, nil
	}
	sec := corev1.Secret{}
	err := runtime.DefaultUnstructuredConverter.FromUnstructured(obj.Object, &sec)
	if err != nil {
		return true, nil, errors.Wrap(err, "unable to cast to secret")
	}
	meta, err := processor.ProcessObjMeta(appMeta, obj)
	if err != nil {
		return true, nil, err
	}

	name := appMeta.TrimName(obj.GetName())
	nameCamelCase := strcase.ToLowerCamel(name)

	secretType := string(sec.Type)
	if secretType != "" {
		secretType, err = yamlformat.Marshal(map[string]interface{}{"type": secretType}, 0)
		if err != nil {
			return true, nil, err
		}
	}

	values := helmify.Values{}
	var data, stringData string
	templatedData := map[string]string{}
	for key := range sec.Data {
		keyCamelCase := strcase.ToLowerCamel(key)
		if key == strings.ToUpper(key) {
			keyCamelCase = strcase.ToLowerCamel(strings.ToLower(key))
		}
		templatedName, err := values.AddSecret(true, nameCamelCase, keyCamelCase)
		if err != nil {
			return true, nil, errors.Wrap(err, "unable add secret to values")
		}
		templatedData[key] = templatedName
	}
	if len(templatedData) != 0 {
		data, err = yamlformat.Marshal(map[string]interface{}{"data": templatedData}, 0)
		if err != nil {
			return true, nil, err
		}
		data = strings.ReplaceAll(data, "'", "")
	}

	templatedData = map[string]string{}
	for key := range sec.StringData {
		keyCamelCase := strcase.ToLowerCamel(key)
		if key == strings.ToUpper(key) {
			keyCamelCase = strcase.ToLowerCamel(strings.ToLower(key))
		}
		templatedName, err := values.AddSecret(false, nameCamelCase, keyCamelCase)
		if err != nil {
			return true, nil, errors.Wrap(err, "unable add secret to values")
		}
		templatedData[key] = templatedName
	}
	if len(templatedData) != 0 {
		stringData, err = yamlformat.Marshal(map[string]interface{}{"stringData": templatedData}, 0)
		if err != nil {
			return true, nil, err
		}
		stringData = strings.ReplaceAll(stringData, "'", "")
	}

	return true, &result{
		name: name + ".yaml",
		data: struct {
			Type       string
			Meta       string
			Data       string
			StringData string
		}{Type: secretType, Meta: meta, Data: data, StringData: stringData},
		values: values,
	}, nil
}

type result struct {
	name string
	data struct {
		Type       string
		Meta       string
		Data       string
		StringData string
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
