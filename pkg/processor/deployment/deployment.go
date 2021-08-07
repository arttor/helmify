package deployment

import (
	"bytes"
	"fmt"
	"github.com/arttor/helmify/pkg/helmify"
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
)

var deploymentGVC = schema.GroupVersionKind{
	Group:   "apps",
	Version: "v1",
	Kind:    "Deployment",
}

const deploymentTempl = `apiVersion: apps/v1
kind: Deployment
metadata:
  name: {{ include "%[1]s.fullname" . }}-controller-manager
  labels:
    control-plane: controller-manager
  {{- include "%[1]s.labels" . | nindent 4 }}
spec:
  {{- if not .Values.autoscaling.enabled }}
  replicas: {{ .Values.replicaCount }}
  {{- end }}
  selector:
    matchLabels:
      control-plane: controller-manager
  {{- include "%[1]s.selectorLabels" . | nindent 6 }}
  template:
    metadata:
      {{- with .Values.podAnnotations }}
      annotations:
      {{- toYaml . | nindent 8 }}
      {{- end }}
      labels:
        control-plane: controller-manager
    {{- include "%[1]s.selectorLabels" . | nindent 8 }}
    spec:
%[2]s
`

func New() helmify.Processor {
	return &deployment{}
}

type deployment struct {
}

func (d deployment) Process(info helmify.ChartInfo, obj *unstructured.Unstructured) (bool, helmify.Template, error) {
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
	name := info.OperatorName

	fullNameTeml := fmt.Sprintf(`{{ include "%s.fullname" . }}`, info.ChartName)

	for _, v := range depl.Spec.Template.Spec.Volumes {
		if v.ConfigMap != nil {
			v.ConfigMap.Name = strings.ReplaceAll(v.ConfigMap.Name, name, fullNameTeml)
		}
	}
	depl.Spec.Template.Spec.ServiceAccountName = strings.ReplaceAll(depl.Spec.Template.Spec.ServiceAccountName, name, fullNameTeml)

	spec, _ := yaml.Marshal(depl.Spec.Template.Spec)
	spec = yamlformat.Indent(spec, 6)
	spec = bytes.TrimRight(spec, "\n ")

	res := fmt.Sprintf(deploymentTempl, info.ChartName, string(spec))

	values := helmify.Values{}
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
		data:   []byte(res),
	}, nil
}

type result struct {
	data   []byte
	values helmify.Values
}

func (r *result) Filename() string {
	return "deployment.yaml"
}

func (r *result) Values() helmify.Values {
	return r.values
}

func (r *result) Write(writer io.Writer) error {
	_, err := writer.Write(r.data)
	return err
}

func (r *result) PostProcess(helmify.Values) {
}
