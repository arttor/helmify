package deployment

import (
	"fmt"
	"io"
	"strings"
	"text/template"

	"github.com/arttor/helmify/pkg/cluster"
	"github.com/arttor/helmify/pkg/processor"
	"github.com/arttor/helmify/pkg/processor/imagePullSecrets"

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
{{- if .Replicas }}
{{ .Replicas }}
{{- end }}
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
{{- include "%[2]s.selectorLabels" . | nindent 6 }}
%[3]s`

const envValue = "{{ .Values.%[1]s.%[2]s.%[3]s }}"

// New creates processor for k8s Deployment resource.
func New() helmify.Processor {
	return &deployment{}
}

type deployment struct{}

// Process k8s Deployment object into template. Returns false if not capable of processing given resource type.
func (d deployment) Process(appMeta helmify.AppMetadata, obj *unstructured.Unstructured) (bool, helmify.Template, error) {
	if obj.GroupVersionKind() != deploymentGVC {
		return false, nil, nil
	}
	depl := appsv1.Deployment{}
	err := runtime.DefaultUnstructuredConverter.FromUnstructured(obj.Object, &depl)
	if err != nil {
		return true, nil, errors.Wrap(err, "unable to cast to deployment")
	}
	meta, err := processor.ProcessObjMeta(appMeta, obj)
	if err != nil {
		return true, nil, err
	}

	values := helmify.Values{}

	name := appMeta.TrimName(obj.GetName())
	replicas, err := processReplicas(name, &depl, &values)
	if err != nil {
		return true, nil, err
	}

	matchLabels, err := yamlformat.Marshal(map[string]interface{}{"matchLabels": depl.Spec.Selector.MatchLabels}, 0)
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
	selector := fmt.Sprintf(selectorTempl, matchLabels, appMeta.ChartName(), matchExpr)
	selector = strings.Trim(selector, " \n")
	selector = string(yamlformat.Indent([]byte(selector), 4))

	podLabels, err := yamlformat.Marshal(depl.Spec.Template.ObjectMeta.Labels, 8)
	if err != nil {
		return true, nil, err
	}
	podLabels += fmt.Sprintf("\n      {{- include \"%s.selectorLabels\" . | nindent 8 }}", appMeta.ChartName())

	podAnnotations := ""
	if len(depl.Spec.Template.ObjectMeta.Annotations) != 0 {
		podAnnotations, err = yamlformat.Marshal(map[string]interface{}{"annotations": depl.Spec.Template.ObjectMeta.Annotations}, 6)
		if err != nil {
			return true, nil, err
		}

		podAnnotations = "\n" + podAnnotations
	}

	nameCamel := strcase.ToLowerCamel(name)
	podValues, err := processPodSpec(nameCamel, appMeta, &depl.Spec.Template.Spec)
	if err != nil {
		return true, nil, err
	}
	err = values.Merge(podValues)
	if err != nil {
		return true, nil, err
	}

	// replace PVC to templated name
	for i := 0; i < len(depl.Spec.Template.Spec.Volumes); i++ {
		vol := depl.Spec.Template.Spec.Volumes[i]
		if vol.PersistentVolumeClaim == nil {
			continue
		}
		tempPVCName := appMeta.TemplatedName(vol.PersistentVolumeClaim.ClaimName)
		depl.Spec.Template.Spec.Volumes[i].PersistentVolumeClaim.ClaimName = tempPVCName
	}

	// replace container resources with template to values.
	specMap, err := runtime.DefaultUnstructuredConverter.ToUnstructured(&depl.Spec.Template.Spec)
	if err != nil {
		return true, nil, err
	}
	containers, _, err := unstructured.NestedSlice(specMap, "containers")
	if err != nil {
		return true, nil, err
	}
	for i := range containers {
		containerName := strcase.ToLowerCamel((containers[i].(map[string]interface{})["name"]).(string))
		res, exists, err := unstructured.NestedMap(values, nameCamel, containerName, "resources")
		if err != nil {
			return true, nil, err
		}
		if !exists || len(res) == 0 {
			continue
		}
		err = unstructured.SetNestedField(containers[i].(map[string]interface{}), fmt.Sprintf(`{{- toYaml .Values.%s.%s.resources | nindent 10 }}`, nameCamel, containerName), "resources")
		if err != nil {
			return true, nil, err
		}
	}
	err = unstructured.SetNestedSlice(specMap, containers, "containers")
	if err != nil {
		return true, nil, err
	}

	if appMeta.Config().ImagePullSecrets {
		imagePullSecrets.ProcessSpecMap(specMap, &values)
	}

	spec, err := yamlformat.Marshal(specMap, 6)
	if err != nil {
		return true, nil, err
	}
	spec = strings.ReplaceAll(spec, "'", "")

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

func processReplicas(name string, deployment *appsv1.Deployment, values *helmify.Values) (string, error) {
	if deployment.Spec.Replicas == nil {
		return "", nil
	}
	replicasTpl, err := values.Add(int64(*deployment.Spec.Replicas), name, "replicas")
	if err != nil {
		return "", err
	}
	replicas, err := yamlformat.Marshal(map[string]interface{}{"replicas": replicasTpl}, 2)
	if err != nil {
		return "", err
	}
	replicas = strings.ReplaceAll(replicas, "'", "")
	return replicas, nil
}

func processPodSpec(name string, appMeta helmify.AppMetadata, pod *corev1.PodSpec) (helmify.Values, error) {
	values := helmify.Values{}
	for i, c := range pod.Containers {
		processed, err := processPodContainer(name, appMeta, c, &values)
		if err != nil {
			return nil, err
		}
		pod.Containers[i] = processed
	}
	for _, v := range pod.Volumes {
		if v.ConfigMap != nil {
			v.ConfigMap.Name = appMeta.TemplatedName(v.ConfigMap.Name)
		}
		if v.Secret != nil {
			v.Secret.SecretName = appMeta.TemplatedName(v.Secret.SecretName)
		}
	}
	pod.ServiceAccountName = appMeta.TemplatedName(pod.ServiceAccountName)

	for i, s := range pod.ImagePullSecrets {
		pod.ImagePullSecrets[i].Name = appMeta.TemplatedName(s.Name)
	}

	return values, nil
}

func processPodContainer(name string, appMeta helmify.AppMetadata, c corev1.Container, values *helmify.Values) (corev1.Container, error) {
	index := strings.LastIndex(c.Image, ":")
	if index < 0 {
		return c, errors.New("wrong image format: " + c.Image)
	}
	repo, tag := c.Image[:index], c.Image[index+1:]
	containerName := strcase.ToLowerCamel(c.Name)
	c.Image = fmt.Sprintf("{{ .Values.%[1]s.%[2]s.image.repository }}:{{ .Values.%[1]s.%[2]s.image.tag | default .Chart.AppVersion }}", name, containerName)

	err := unstructured.SetNestedField(*values, repo, name, containerName, "image", "repository")
	if err != nil {
		return c, errors.Wrap(err, "unable to set deployment value field")
	}
	err = unstructured.SetNestedField(*values, tag, name, containerName, "image", "tag")
	if err != nil {
		return c, errors.Wrap(err, "unable to set deployment value field")
	}

	for i := 0; i < len(c.Env); i++ {
		if c.Env[i].ValueFrom != nil && c.Env[i].ValueFrom.SecretKeyRef != nil {
			c.Env[i].ValueFrom.SecretKeyRef.Name = appMeta.TemplatedName(c.Env[i].ValueFrom.SecretKeyRef.Name)
		} else if c.Env[i].ValueFrom != nil && c.Env[i].ValueFrom.ConfigMapKeyRef != nil {
			c.Env[i].ValueFrom.ConfigMapKeyRef.Name = appMeta.TemplatedName(c.Env[i].ValueFrom.ConfigMapKeyRef.Name)
		} else {

			err = unstructured.SetNestedField(*values, c.Env[i].Value, name, containerName, "env", strcase.ToLowerCamel(strings.ToLower(c.Env[i].Name)))
			if err != nil {
				return c, errors.Wrap(err, "unable to set deployment value field")
			}

			c.Env[i].Value = fmt.Sprintf(envValue, name, containerName, strcase.ToLowerCamel(strings.ToLower(c.Env[i].Name)))
		}
	}

	for _, e := range c.EnvFrom {
		if e.SecretRef != nil {
			e.SecretRef.Name = appMeta.TemplatedName(e.SecretRef.Name)
		}
		if e.ConfigMapRef != nil {
			e.ConfigMapRef.Name = appMeta.TemplatedName(e.ConfigMapRef.Name)
		}
	}
	c.Env = append(c.Env, corev1.EnvVar{
		Name:  cluster.DomainEnv,
		Value: fmt.Sprintf("{{ .Values.%s }}", cluster.DomainKey),
	})
	for k, v := range c.Resources.Requests {
		err = unstructured.SetNestedField(*values, v.ToUnstructured(), name, containerName, "resources", "requests", k.String())
		if err != nil {
			return c, errors.Wrap(err, "unable to set container resources value")
		}
	}
	for k, v := range c.Resources.Limits {
		err = unstructured.SetNestedField(*values, v.ToUnstructured(), name, containerName, "resources", "limits", k.String())
		if err != nil {
			return c, errors.Wrap(err, "unable to set container resources value")
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
