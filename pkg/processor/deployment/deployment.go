package deployment

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
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

var deploymentGVC = schema.GroupVersionKind{
	Group:   "apps",
	Version: "v1",
	Kind:    "Deployment",
}

var deploymentTempl, _ = template.New("deployment").Parse(
	`{{- .Meta }}
spec:
  replicas: {{ .Replicas }}
  selector:
{{ .Selector }}
  template:
    metadata:
      labels:
{{ .PodLabels }}
{{- .PodAnnotations }}
    spec:
{{ .Spec }}`)

const selectorTempl = `%[1]s
{{- include "%[2]s.selectorLabels" . | nindent 8 }}
%[3]s`

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
	name, meta, err := processor.ProcessMetadata(info, obj)
	if err != nil {
		return true, nil, err
	}

	values := helmify.Values{}

	replicas, err := values.Add(int64(*depl.Spec.Replicas), name, "replicas")
	if err != nil {
		return true, nil, err
	}

	matchLabels, err := yamlformat.Marshal(map[string]interface{}{"matchLabels": depl.Spec.Selector.MatchLabels}, 2)
	if err != nil {
		return true, nil, err
	}
	matchExpr := ""
	if depl.Spec.Selector.MatchExpressions != nil {
		matchExpr, err = yamlformat.Marshal(map[string]interface{}{"matchExpressions": depl.Spec.Selector.MatchExpressions}, 0)
		if err != nil {
			return true, nil, err
		}
	}
	selector := fmt.Sprintf(selectorTempl, matchLabels, info.ChartName, matchExpr)
	selector = strings.Trim(selector, " \n")
	selector = string(yamlformat.Indent([]byte(selector), 4))

	podLabels, err := yamlformat.Marshal(depl.Spec.Template.ObjectMeta.Labels, 8)
	if err != nil {
		return true, nil, err
	}
	podLabels += fmt.Sprintf("\n      {{- include \"%s.selectorLabels\" . | nindent 8 }}", info.ChartName)

	podAnnotations := ""
	if len(depl.Spec.Template.ObjectMeta.Annotations) != 0 {
		podAnnotations, err = yamlformat.Marshal(map[string]interface{}{"annotations": depl.Spec.Template.ObjectMeta.Annotations}, 6)
		if err != nil {
			return true, nil, err
		}
	}

	// TODO: postprocess container resources
	podValues, err := processPodSpec(info, &depl.Spec.Template.Spec)
	if err != nil {
		return true, nil, err
	}
	err = values.Merge(podValues)
	if err != nil {
		return true, nil, err
	}
	spec, err := yamlformat.Marshal(depl.Spec.Template.Spec, 6)
	if err != nil {
		return true, nil, err
	}

	return true, &result{
		values: values,
		data: struct {
			Meta           string
			Replicas       string
			Selector       string
			PodLabels      string
			PodAnnotations string
			Spec           string
		}{
			Meta:           meta,
			Replicas:       replicas,
			Selector:       selector,
			PodLabels:      podLabels,
			PodAnnotations: podAnnotations,
			Spec:           spec,
		},
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
	data struct {
		Meta           string
		Replicas       string
		Selector       string
		PodLabels      string
		PodAnnotations string
		Spec           string
	}
	values helmify.Values
}

func (r *result) Filename() string {
	return "deployment.yaml"
}

func (r *result) Values() helmify.Values {
	return r.values
}

func (r *result) Write(writer io.Writer) error {
	return deploymentTempl.Execute(writer, r.data)
}
