package configmap

import (
	"bytes"
	_ "embed"
	"github.com/arttor/helmify/pkg/context"
	yamlformat "github.com/arttor/helmify/pkg/yaml"
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
	configmapTempl = `apiVersion: v1
kind: ConfigMap
metadata:
  name: {{ include "<CHART_NAME>.fullname" . }}-<NAME>
  labels:
  {{- include "<CHART_NAME>.labels" . | nindent 4 }}
data:
`
)

var (
	configMapGVC = schema.GroupVersionKind{
		Group:   "",
		Version: "v1",
		Kind:    "ConfigMap",
	}
)

func New() context.Processor {
	return &configMap{}
}

type configMap struct {
}

func (d configMap) Process(obj *unstructured.Unstructured) (bool, context.Template, error) {
	if obj.GroupVersionKind() != configMapGVC {
		return false, nil, nil
	}
	cm := corev1.ConfigMap{}
	err := runtime.DefaultUnstructuredConverter.FromUnstructured(obj.Object, &cm)
	if err != nil {
		return true, nil, errors.Wrap(err, "unable to cast to configmap")
	}
	prefix := strings.TrimSuffix(cm.GetNamespace(), "system")
	name := strings.TrimPrefix(cm.GetName(), prefix)
	res := strings.ReplaceAll(configmapTempl, "<NAME>", name)
	if cm.Data != nil && len(cm.Data) != 0 {
		data, _ := yaml.Marshal(cm.Data)
		data = yamlformat.Indent(data, 2)
		data = bytes.TrimRight(data, "\n ")
		res = res + string(data)
	}
	return true, &result{
		name: name + ".yaml",
		data: []byte(res),
	}, nil
}

type result struct {
	name      string
	data      []byte
	chartName string
}

func (r *result) Filename() string {
	return r.name
}

func (r *result) GVK() schema.GroupVersionKind {
	return configMapGVC
}

func (r *result) Values() context.Values {
	return context.Values{}
}

func (r *result) Write(writer io.Writer) error {
	_, err := writer.Write(bytes.ReplaceAll(r.data, []byte("<CHART_NAME>"), []byte(r.chartName)))
	return err
}

func (r *result) PostProcess(data context.Data) {
}

func (r *result) SetChartName(name string) {
	r.chartName = name
}
