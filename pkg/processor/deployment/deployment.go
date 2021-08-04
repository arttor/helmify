package deployment

import (
	"bytes"
	_ "embed"
	"github.com/arttor/helmify/pkg/context"
	yamlformat "github.com/arttor/helmify/pkg/yaml"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"io"
	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/yaml"
	"strings"
	//"k8s.io/apimachinery/pkg/util/yaml"
)

var (
	deploymentGVC = schema.GroupVersionKind{
		Group:   "apps",
		Version: "v1",
		Kind:    "Deployment",
	}
	//go:embed deployment.yaml
	deploymentYaml []byte
)

func New() context.Processor {
	return &deployment{}
}

type deployment struct {
}

func (d deployment) Process(obj *unstructured.Unstructured) (bool, context.Template, error) {
	if obj.GroupVersionKind() != deploymentGVC {
		return false, nil, nil
	}
	depl := appsv1.Deployment{}
	err := runtime.DefaultUnstructuredConverter.FromUnstructured(obj.Object, &depl)
	if err != nil {
		return true, nil, errors.Wrap(err, "unable to cast to deployment")
	}
	if depl.Labels["control-plane"] != "controller-manager" {
		logrus.Warn("got deployment but not controller manager")
		return false, nil, nil
	}
	var repo, tag string
	for i, c := range depl.Spec.Template.Spec.Containers {
		if c.Name != "manager" {
			continue
		}
		index := strings.LastIndex(c.Image, ":")
		if index < 0 {
			return true, nil, errors.New("wrong image format: " + c.Image)
		}
		repo = c.Image[:index]
		tag = c.Image[index+1:]
		c.Image = "{{ .Values.manager.image.repository }}:{{ .Values.manager.image.tag | default .Chart.AppVersion }}"
		depl.Spec.Template.Spec.Containers[i] = c
	}
	name := strings.TrimSuffix(depl.GetNamespace(), "-system")
	for _, v := range depl.Spec.Template.Spec.Volumes {
		if v.ConfigMap != nil {
			v.ConfigMap.Name = strings.ReplaceAll(v.ConfigMap.Name, name, `{{ include "<CHART_NAME>.fullname" . }}`)
		}
	}
	depl.Spec.Template.Spec.ServiceAccountName = strings.ReplaceAll(depl.Spec.Template.Spec.ServiceAccountName, name, `{{ include "<CHART_NAME>.fullname" . }}`)

	spec, _ := yaml.Marshal(depl.Spec.Template.Spec)
	spec = yamlformat.Indent(spec, 6)
	spec = bytes.TrimRight(spec, "\n ")
	res := append(deploymentYaml, spec...)
	values := context.Values{}
	err = unstructured.SetNestedField(values, false, "autoscaling", "enabled")
	if err != nil {
		return true, nil, errors.Wrap(err, "unable to set deployment value field")
	}
	err = unstructured.SetNestedField(values, int64(*depl.Spec.Replicas), "replicaCount")
	if err != nil {
		return true, nil, errors.Wrap(err, "unable to set deployment value field")
	}
	err = unstructured.SetNestedStringMap(values, depl.Spec.Template.ObjectMeta.Annotations, "podAnnotations")
	if err != nil {
		return true, nil, errors.Wrap(err, "unable to set deployment value field")
	}
	err = unstructured.SetNestedField(values, repo, "manager", "image", "repository")
	if err != nil {
		return true, nil, errors.Wrap(err, "unable to set deployment value field")
	}
	err = unstructured.SetNestedField(values, tag, "manager", "image", "tag")
	if err != nil {
		return true, nil, errors.Wrap(err, "unable to set deployment value field")
	}
	return true, &result{
		values: values,
		data:   res,
	}, nil
}

type result struct {
	data      []byte
	values    context.Values
	chartName string
}

func (r *result) Filename() string {
	return "deployment.yaml"
}

func (r *result) GVK() schema.GroupVersionKind {
	return deploymentGVC
}

func (r *result) Values() context.Values {
	return r.values
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
