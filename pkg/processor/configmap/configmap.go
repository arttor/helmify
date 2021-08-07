package configmap

import (
	"bytes"
	_ "embed"
	"fmt"
	"github.com/arttor/helmify/pkg/helmify"
	yamlformat "github.com/arttor/helmify/pkg/yaml"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"io"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/yaml"
	"strconv"
	"strings"
)

const (
	configmapTempl = `apiVersion: v1
kind: ConfigMap
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
		Kind:    "ConfigMap",
	}
)

func New() helmify.Processor {
	return &configMap{}
}

type configMap struct {
}

func (d configMap) Process(info helmify.ChartInfo, obj *unstructured.Unstructured) (bool, helmify.Template, error) {
	if obj.GroupVersionKind() != configMapGVC {
		return false, nil, nil
	}
	cm := corev1.ConfigMap{}
	err := runtime.DefaultUnstructuredConverter.FromUnstructured(obj.Object, &cm)
	if err != nil {
		return true, nil, errors.Wrap(err, "unable to cast to configmap")
	}
	name := strings.TrimPrefix(cm.GetName(), info.OperatorName+"-")
	var values helmify.Values
	var data []byte
	if cm.Data != nil && len(cm.Data) != 0 {
		cm.Data, values = parseMapData(cm.Data)
		data, _ = yaml.Marshal(cm.Data)
		data = yamlformat.Indent(data, 2)
		data = bytes.TrimRight(data, "\n ")
		data = bytes.ReplaceAll(data, []byte("'"), []byte(""))
	}
	res := fmt.Sprintf(configmapTempl, info.ChartName, name, string(data))

	return true, &result{
		name:   name + ".yaml",
		data:   []byte(res),
		values: values,
	}, nil
}

func parseMapData(data map[string]string) (map[string]string, helmify.Values) {
	configStr := data["controller_manager_config.yaml"]
	values := helmify.Values{}
	if configStr == "" {
		return data, values
	}
	config := map[string]interface{}{}
	err := yaml.Unmarshal([]byte(configStr), &config)
	if err != nil {
		logrus.WithError(err).Warn("unable to unmarshal controller_manager_config.yaml")
		return data, values
	}
	parseConfig(&config, &values, []string{"managerConfig"})
	confBytes, err := yaml.Marshal(config)
	if err != nil {
		logrus.WithError(err).Warn("unable to marshal controller_manager_config.yaml")
		return data, helmify.Values{}
	}
	data["controller_manager_config.yaml"] = string(confBytes)
	return data, values
}

func parseConfig(config *map[string]interface{}, values *helmify.Values, path []string) {
	for k, v := range *config {
		switch t := v.(type) {
		case string, bool, float64, int64:
			replace(config, values, path, k)
		case []interface{}:
			logrus.Warn("configmap: arrays not supported")
		case map[string]interface{}:
			parseConfig(&t, values, append(path, k))
		case map[interface{}]interface{}:
			c, ok := v.(map[string]interface{})
			if !ok {
				logrus.Warn("configmap: unable to cast to map[string]interface{}")
			} else {
				parseConfig(&c, values, append(path, k))
			}
		default:
			logrus.Warn("configmap: unknown type ", t)
			fmt.Printf("\n%T\n", t)
		}
	}
}
func replace(config *map[string]interface{}, values *helmify.Values, path []string, key string) {
	if key == "kind" || key == "apiVersion" {
		return
	}
	valName := append(path, key)
	val, ok := (*config)[key].(string)
	if !ok {
		_ = unstructured.SetNestedField(*values, (*config)[key], valName...)
		(*config)[key] = "{{ .Values." + strings.Join(valName, ".") + " }}"
		return
	}
	i, err := strconv.ParseInt(val, 10, 64)
	if err == nil {
		_ = unstructured.SetNestedField(*values, i, valName...)
		(*config)[key] = "{{ .Values." + strings.Join(valName, ".") + " }}"
		return
	}
	f, err := strconv.ParseFloat(val, 64)
	if err == nil {
		_ = unstructured.SetNestedField(*values, f, valName...)
		(*config)[key] = "{{ .Values." + strings.Join(valName, ".") + " }}"
		return
	}
	b, err := strconv.ParseBool(val)
	if err == nil {
		_ = unstructured.SetNestedField(*values, b, valName...)
		(*config)[key] = "{{ .Values." + strings.Join(valName, ".") + " }}"
		return
	}
	_ = unstructured.SetNestedField(*values, val, valName...)
	(*config)[key] = "{{ .Values." + strings.Join(valName, ".") + " | quote }}"
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
