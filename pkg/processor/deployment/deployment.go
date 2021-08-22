package deployment

import (
	"bytes"
	"fmt"
	"io"
	"strings"

	"github.com/arttor/helmify/pkg/helmify"
	yamlformat "github.com/arttor/helmify/pkg/yaml"
	"github.com/iancoleman/strcase"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/yaml"
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
%[2]s`

// New creates processor for k8s Deployment resource.
func New() helmify.Processor {
	return &deployment{}
}

type deployment struct{}

// Process k8s Deployment object into template. Returns false if not capable of processing given resource type.
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
	}
	values, err := processPodSpec(info, &depl.Spec.Template.Spec)
	if err != nil {
		return true, nil, err
	}
	spec, _ := yaml.Marshal(depl.Spec.Template.Spec)
	spec = yamlformat.Indent(spec, 6)
	spec = bytes.TrimRight(spec, "\n ")

	res := fmt.Sprintf(deploymentTempl, info.ChartName, string(spec))

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

	return true, &result{
		values: values,
		data:   []byte(res),
	}, nil
}

func processPodSpec(info helmify.ChartInfo, pod *corev1.PodSpec) (helmify.Values, error) {
	name := info.ApplicationName
	templatedName := fmt.Sprintf(`{{ include "%s.fullname" . }}`, info.ChartName)
	values := helmify.Values{}
	for i, c := range pod.Containers {
		processed, err := processPodContainer(name, templatedName, c, &values)
		if err != nil {
			return nil, err
		}
		pod.Containers[i] = processed
	}
	for _, v := range pod.Volumes {
		if v.ConfigMap != nil {
			v.ConfigMap.Name = strings.ReplaceAll(v.ConfigMap.Name, name, templatedName)
		}
		if v.Secret != nil {
			v.Secret.SecretName = strings.ReplaceAll(v.Secret.SecretName, name, templatedName)
		}
	}
	pod.ServiceAccountName = strings.ReplaceAll(pod.ServiceAccountName, name, templatedName)
	return values, nil
}

func processPodContainer(name, templatedName string, c corev1.Container, values *helmify.Values) (corev1.Container, error) {
	index := strings.LastIndex(c.Image, ":")
	if index < 0 {
		return c, errors.New("wrong image format: " + c.Image)
	}
	repo, tag := c.Image[:index], c.Image[index+1:]
	nameCamel := strcase.ToLowerCamel(c.Name)
	c.Image = fmt.Sprintf("{{ .Values.image.%[1]s.repository }}:{{ .Values.image.%[1]s.tag | default .Chart.AppVersion }}", nameCamel)

	err := unstructured.SetNestedField(*values, repo, "image", nameCamel, "repository")
	if err != nil {
		return c, errors.Wrap(err, "unable to set deployment value field")
	}
	err = unstructured.SetNestedField(*values, tag, "image", nameCamel, "tag")
	if err != nil {
		return c, errors.Wrap(err, "unable to set deployment value field")
	}
	for _, e := range c.Env {
		if e.ValueFrom != nil && e.ValueFrom.SecretKeyRef != nil {
			e.ValueFrom.SecretKeyRef.Name = strings.ReplaceAll(e.ValueFrom.SecretKeyRef.Name, name, templatedName)
		}
		if e.ValueFrom != nil && e.ValueFrom.ConfigMapKeyRef != nil {
			e.ValueFrom.ConfigMapKeyRef.Name = strings.ReplaceAll(e.ValueFrom.ConfigMapKeyRef.Name, name, templatedName)
		}
	}
	for _, e := range c.EnvFrom {
		if e.SecretRef != nil {
			e.SecretRef.Name = strings.ReplaceAll(e.SecretRef.Name, name, templatedName)
		}
		if e.ConfigMapRef != nil {
			e.ConfigMapRef.Name = strings.ReplaceAll(e.ConfigMapRef.Name, name, templatedName)
		}
	}
	return c, nil
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
