package configmap

import (
	"github.com/arttor/helmify/pkg/format"
	"io"
	"strings"
	"text/template"

	"github.com/arttor/helmify/pkg/processor"

	"github.com/arttor/helmify/pkg/helmify"
	yamlformat "github.com/arttor/helmify/pkg/yaml"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

var configMapTempl, _ = template.New("configMap").Parse(
	`{{ .Meta }}
{{- if .Immutable }}
{{ .Immutable }}
{{- end }}
{{- if .BinaryData }}
{{ .BinaryData }}
{{- end }}
{{- if .Data }}
{{ .Data }}
{{- end }}`)

var configMapGVC = schema.GroupVersionKind{
	Group:   "",
	Version: "v1",
	Kind:    "ConfigMap",
}

// New creates processor for k8s ConfigMap resource.
func New() helmify.Processor {
	return &configMap{}
}

type configMap struct{}

// Process k8s ConfigMap object into template. Returns false if not capable of processing given resource type.
func (d configMap) Process(appMeta helmify.AppMetadata, obj *unstructured.Unstructured) (bool, helmify.Template, error) {
	if obj.GroupVersionKind() != configMapGVC {
		return false, nil, nil
	}
	var meta, immutable, binaryData, data string
	meta, err := processor.ProcessObjMeta(appMeta, obj)
	if err != nil {
		return true, nil, err
	}

	if field, exists, _ := unstructured.NestedBool(obj.Object, "immutable"); exists {
		immutable, err = yamlformat.Marshal(map[string]interface{}{"immutable": field}, 0)
		if err != nil {
			return true, nil, err
		}
	}
	if field, exists, _ := unstructured.NestedStringMap(obj.Object, "binaryData"); exists {
		binaryData, err = yamlformat.Marshal(map[string]interface{}{"binaryData": field}, 0)
		if err != nil {
			return true, nil, err
		}
	}

	name := appMeta.TrimName(obj.GetName())
	var values helmify.Values
	if field, exists, _ := unstructured.NestedStringMap(obj.Object, "data"); exists {
		field, values = parseMapData(field, name)
		data, err = yamlformat.Marshal(map[string]interface{}{"data": field}, 0)
		if err != nil {
			return true, nil, err
		}
		data = strings.ReplaceAll(data, "'", "")
	}

	return true, &result{
		name: name + ".yaml",
		data: struct {
			Meta       string
			Immutable  string
			BinaryData string
			Data       string
		}{Meta: meta, Immutable: immutable, BinaryData: binaryData, Data: data},
		values: values,
	}, nil
}

func parseMapData(data map[string]string, configName string) (map[string]string, helmify.Values) {
	values := helmify.Values{}
	for key, value := range data {
		valuesNamePath := []string{configName, key}
		if strings.HasSuffix(key, ".properties") {
			// handle properties
			templated, err := parseProperties(value, valuesNamePath, values)
			if err != nil {
				logrus.WithError(err).Errorf("unable to process configmap data: %v", valuesNamePath)
				continue
			}
			data[key] = templated
			continue
		}
		if strings.Contains(value, "\n") {
			value = format.RemoveTrailingWhitespaces(value)
			templatedVal, err := values.AddYaml(value, 1, false, valuesNamePath...)
			if err != nil {
				logrus.WithError(err).Errorf("unable to process multiline configmap data: %v", valuesNamePath)
				continue
			}
			data[key] = templatedVal
			continue
		}
		// handle plain string
		templatedVal, err := values.Add(value, valuesNamePath...)
		if err != nil {
			logrus.WithError(err).Errorf("unable to process configmap data: %v", valuesNamePath)
			continue
		}
		data[key] = templatedVal
	}
	return data, values
}

// func parseProperties(properties string, path []string, values helmify.Values) (string, error) {
func parseProperties(properties interface{}, path []string, values helmify.Values) (string, error) {
	var res strings.Builder
	for _, line := range strings.Split(strings.TrimSuffix(properties.(string), "\n"), "\n") {
		prop := strings.Split(line, "=")
		if len(prop) != 2 {
			return "", errors.Errorf("wrong property format in %v: %s", path, line)
		}
		propName, propVal := prop[0], prop[1]
		propNamePath := strings.Split(propName, ".")
		templatedVal, err := values.Add(propVal, append(path, propNamePath...)...)
		if err != nil {
			return "", err
		}
		_, err = res.WriteString(propName + "=" + templatedVal + "\n")
		if err != nil {
			return "", errors.Wrap(err, "unable to write to string builder")
		}
	}
	return res.String(), nil
}

type result struct {
	name string
	data struct {
		Meta       string
		Immutable  string
		BinaryData string
		Data       string
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
	return configMapTempl.Execute(writer, r.data)
}
