package processor

import (
	"fmt"
	"strings"

	"github.com/arttor/helmify/pkg/helmify"
	yamlformat "github.com/arttor/helmify/pkg/yaml"

	"github.com/iancoleman/strcase"
	"github.com/pkg/errors"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
)

type Pod struct {
	Name    string
	AppMeta helmify.AppMetadata
	Pod     *corev1.PodTemplateSpec
}

func (p *Pod) ProcessSpec(values *helmify.Values) (string, error) {
	podValues, err := p.getValues()
	if err != nil {
		return "", err
	}
	err = values.Merge(podValues)
	if err != nil {
		return "", err
	}

	template, err := p.template(values)
	if err != nil {
		return  "", err
	}

	return template, nil
}

func (p *Pod) ProcessObjectMeta() (string, string, error) {
	podLabels, err := yamlformat.Marshal(p.Pod.ObjectMeta.Labels, 8)
	if err != nil {
		return "", "", err
	}

	podAnnotations := ""
	if len(p.Pod.ObjectMeta.Annotations) != 0 {
		podAnnotations, err = yamlformat.Marshal(map[string]interface{}{"annotations": p.Pod.ObjectMeta.Annotations}, 6)
		if err != nil {
			return "", "", err
		}

		podAnnotations = "\n" + podAnnotations
	}

	return podLabels, podAnnotations, nil
}

func (p *Pod) getValues() (helmify.Values, error) {
	values := helmify.Values{}
	for i, c := range p.Pod.Spec.Containers {
		processed, err := p.processPodContainer(c, &values)
		if err != nil {
			return nil, err
		}
		p.Pod.Spec.Containers[i] = processed
	}
	for _, v := range p.Pod.Spec.Volumes {
		if v.ConfigMap != nil {
			v.ConfigMap.Name = p.AppMeta.TemplatedName(v.ConfigMap.Name)
		}
		if v.Secret != nil {
			v.Secret.SecretName = p.AppMeta.TemplatedName(v.Secret.SecretName)
		}
	}
	p.Pod.Spec.ServiceAccountName = p.AppMeta.TemplatedName(p.Pod.Spec.ServiceAccountName)

	for i, s := range p.Pod.Spec.ImagePullSecrets {
		p.Pod.Spec.ImagePullSecrets[i].Name = p.AppMeta.TemplatedName(s.Name)
	}

	return values, nil
}

func (p *Pod) processPodContainer(c corev1.Container, values *helmify.Values) (corev1.Container, error) {
	index := strings.LastIndex(c.Image, ":")
	if index < 0 {
		return c, errors.New("wrong image format: " + c.Image)
	}
	repo, tag := c.Image[:index], c.Image[index+1:]
	containerName := strcase.ToLowerCamel(c.Name)
	c.Image = fmt.Sprintf("{{ .Values.%[1]s.%[2]s.image.repository }}:{{ .Values.%[1]s.%[2]s.image.tag | default .Chart.AppVersion }}", p.Name, containerName)

	err := unstructured.SetNestedField(*values, repo, p.Name, containerName, "image", "repository")
	if err != nil {
		return c, errors.Wrap(err, "unable to set deployment value field")
	}
	err = unstructured.SetNestedField(*values, tag, p.Name, containerName, "image", "tag")
	if err != nil {
		return c, errors.Wrap(err, "unable to set deployment value field")
	}
	for _, e := range c.Env {
		if e.ValueFrom != nil && e.ValueFrom.SecretKeyRef != nil {
			e.ValueFrom.SecretKeyRef.Name = p.AppMeta.TemplatedName(e.ValueFrom.SecretKeyRef.Name)
		}
		if e.ValueFrom != nil && e.ValueFrom.ConfigMapKeyRef != nil {
			e.ValueFrom.ConfigMapKeyRef.Name = p.AppMeta.TemplatedName(e.ValueFrom.ConfigMapKeyRef.Name)
		}
	}
	for _, e := range c.EnvFrom {
		if e.SecretRef != nil {
			e.SecretRef.Name = p.AppMeta.TemplatedName(e.SecretRef.Name)
		}
		if e.ConfigMapRef != nil {
			e.ConfigMapRef.Name = p.AppMeta.TemplatedName(e.ConfigMapRef.Name)
		}
	}
	for k, v := range c.Resources.Requests {
		err = unstructured.SetNestedField(*values, v.ToUnstructured(), p.Name, containerName, "resources", "requests", k.String())
		if err != nil {
			return c, errors.Wrap(err, "unable to set container resources value")
		}
	}
	for k, v := range c.Resources.Limits {
		err = unstructured.SetNestedField(*values, v.ToUnstructured(), p.Name, containerName, "resources", "limits", k.String())
		if err != nil {
			return c, errors.Wrap(err, "unable to set container resources value")
		}
	}
	if len(c.Args) != 0 {
		err = unstructured.SetNestedStringSlice(*values, c.Args, p.Name, containerName, "args")
		if err != nil {
			return c, errors.Wrap(err, "unable to set container resources value")
		}
	}
	return c, nil
}

func (p *Pod) template(values *helmify.Values) (string, error) {
	// replace PVC to templated name
	for i := 0; i < len(p.Pod.Spec.Volumes); i++ {
		vol := p.Pod.Spec.Volumes[i]
		if vol.PersistentVolumeClaim == nil {
			continue
		}
		tempPVCName := p.AppMeta.TemplatedName(vol.PersistentVolumeClaim.ClaimName)
		p.Pod.Spec.Volumes[i].PersistentVolumeClaim.ClaimName = tempPVCName
	}

	// replace container resources with template to values.
	specMap, err := runtime.DefaultUnstructuredConverter.ToUnstructured(&p.Pod.Spec)
	if err != nil {
		return "", err
	}
	containers, _, err := unstructured.NestedSlice(specMap, "containers")
	if err != nil {
		return "", err
	}
	for i := range containers {
		containerName := strcase.ToLowerCamel((containers[i].(map[string]interface{})["name"]).(string))
		res, exists, err := unstructured.NestedMap(*values, p.Name, containerName, "resources")
		if err != nil {
			return "", err
		}
		if !exists || len(res) == 0 {
			continue
		}
		err = unstructured.SetNestedField(containers[i].(map[string]interface{}), fmt.Sprintf(`{{- toYaml .Values.%s.%s.resources | nindent 10 }}`, p.Name, containerName), "resources")
		if err != nil {
			return "", err
		}
	}
	for i := range containers {
		containerName := strcase.ToLowerCamel((containers[i].(map[string]interface{})["name"]).(string))
		res, exists, err := unstructured.NestedSlice(*values, p.Name, containerName, "args")
		if err != nil {
			return "", err
		}
		if !exists || len(res) == 0 {
			continue
		}
		err = unstructured.SetNestedField(containers[i].(map[string]interface{}), fmt.Sprintf(`{{- toYaml .Values.%s.%s.args | nindent 10 }}`, p.Name, containerName), "args")
		if err != nil {
			return "", err
		}
	}
	err = unstructured.SetNestedSlice(specMap, containers, "containers")
	if err != nil {
		return "", err
	}

	spec, err := yamlformat.Marshal(specMap, 6)
	if err != nil {
		return "", err
	}
	spec = strings.ReplaceAll(spec, "'", "")

	return spec, nil
}
