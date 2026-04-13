package pod

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/arttor/helmify/pkg/cluster"
	"github.com/arttor/helmify/pkg/helmify"
	securityContext "github.com/arttor/helmify/pkg/processor/security-context"
	"github.com/iancoleman/strcase"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
)

const imagePullPolicyTemplate = "{{ .Values.%[1]s.%[2]s.imagePullPolicy }}"
const envValue = "{{ quote .Values.%[1]s.%[2]s.%[3]s.%[4]s }}"
const baseIndent = 8

func ProcessSpec(objName string, appMeta helmify.AppMetadata, spec corev1.PodSpec, addIndent int) (map[string]interface{}, helmify.Values, error) {
	nindent := baseIndent + addIndent

	values, err := processPodSpec(objName, appMeta, &spec)
	if err != nil {
		return nil, nil, err
	}

	// replace PVC to templated name
	for i := 0; i < len(spec.Volumes); i++ {
		vol := spec.Volumes[i]
		if vol.PersistentVolumeClaim == nil {
			continue
		}
		tempPVCName := appMeta.TemplatedName(vol.PersistentVolumeClaim.ClaimName)

		spec.Volumes[i].PersistentVolumeClaim.ClaimName = tempPVCName
	}

	// replace container resources with template to values.
	specMap, err := runtime.DefaultUnstructuredConverter.ToUnstructured(&spec)
	if err != nil {
		return nil, nil, fmt.Errorf("%w: unable to convert podSpec to map", err)
	}

	specMap, values, err = processNestedContainers(specMap, objName, values, "containers", nindent)
	if err != nil {
		return nil, nil, err
	}

	specMap, values, err = processNestedContainers(specMap, objName, values, "initContainers", nindent)
	if err != nil {
		return nil, nil, err
	}

	if appMeta.Config().ImagePullSecrets {
		if _, defined := specMap["imagePullSecrets"]; !defined {
			specMap["imagePullSecrets"] = "{{ .Values.imagePullSecrets | default list | toJson }}"
			values["imagePullSecrets"] = []string{}
		}
	}

	err = securityContext.ProcessContainerSecurityContext(objName, specMap, &values, nindent)
	if err != nil {
		return nil, nil, err
	}
	if spec.SecurityContext != nil {
		securityContextMap, err := runtime.DefaultUnstructuredConverter.ToUnstructured(&spec.SecurityContext)
		if err != nil {
			return nil, nil, err
		}
		if len(securityContextMap) > 0 {
			err = unstructured.SetNestedField(specMap, fmt.Sprintf(`{{- toYaml .Values.%[1]s.podSecurityContext | nindent %d }}`, objName, nindent), "securityContext")
			if err != nil {
				return nil, nil, err
			}

			err = unstructured.SetNestedField(values, securityContextMap, objName, "podSecurityContext")
			if err != nil {
				return nil, nil, fmt.Errorf("%w: unable to set deployment value field", err)
			}
		}
	}

	// process nodeSelector if presented:
	err = unstructured.SetNestedField(specMap, fmt.Sprintf(`{{- toYaml .Values.%s.nodeSelector | nindent %d }}`, objName, nindent), "nodeSelector")
	if err != nil {
		return nil, nil, err
	}
	if spec.NodeSelector != nil {
		err = unstructured.SetNestedStringMap(values, spec.NodeSelector, objName, "nodeSelector")
		if err != nil {
			return nil, nil, err
		}
	} else {
		err = unstructured.SetNestedField(values, map[string]interface{}{}, objName, "nodeSelector")
		if err != nil {
			return nil, nil, err
		}
	}

	// process tolerations if presented:
	err = unstructured.SetNestedField(specMap, fmt.Sprintf(`{{- toYaml .Values.%s.tolerations | nindent %d }}`, objName, nindent), "tolerations")
	if err != nil {
		return nil, nil, err
	}
	if spec.Tolerations != nil {
		tolerations := make([]any, len(spec.Tolerations))
		inrec, err := json.Marshal(spec.Tolerations)
		if err != nil {
			return nil, nil, err
		}
		err = json.Unmarshal(inrec, &tolerations)
		if err != nil {
			return nil, nil, err
		}
		err = unstructured.SetNestedSlice(values, tolerations, objName, "tolerations")
		if err != nil {
			return nil, nil, err
		}
	} else {
		err = unstructured.SetNestedSlice(values, []any{}, objName, "tolerations")
		if err != nil {
			return nil, nil, err
		}
	}

	// process topologySpreadConstraints if presented:
	err = unstructured.SetNestedField(specMap, fmt.Sprintf(`{{- toYaml .Values.%s.topologySpreadConstraints | nindent %d }}`, objName, nindent), "topologySpreadConstraints")
	if err != nil {
		return nil, nil, err
	}
	if spec.TopologySpreadConstraints != nil {
		topologySpreadConstraints := make([]any, len(spec.TopologySpreadConstraints))
		inrec, err := json.Marshal(spec.TopologySpreadConstraints)
		if err != nil {
			return nil, nil, err
		}
		err = json.Unmarshal(inrec, &topologySpreadConstraints)
		if err != nil {
			return nil, nil, err
		}
		err = unstructured.SetNestedSlice(values, topologySpreadConstraints, objName, "topologySpreadConstraints")
		if err != nil {
			return nil, nil, err
		}
	} else {
		err = unstructured.SetNestedSlice(values, []any{}, objName, "topologySpreadConstraints")
		if err != nil {
			return nil, nil, err
		}
	}

	return specMap, values, nil
}

func processNestedContainers(specMap map[string]interface{}, objName string, values map[string]interface{}, containerKey string, nindent int) (map[string]interface{}, map[string]interface{}, error) {
	containers, _, err := unstructured.NestedSlice(specMap, containerKey)
	if err != nil {
		return nil, nil, err
	}

	if len(containers) > 0 {
		containers, values, err = processContainers(objName, values, containerKey, containers, nindent)
		if err != nil {
			return nil, nil, err
		}

		err = unstructured.SetNestedSlice(specMap, containers, containerKey)
		if err != nil {
			return nil, nil, err
		}
	}

	return specMap, values, nil
}

func processContainers(objName string, values helmify.Values, containerType string, containers []interface{}, nindent int) ([]interface{}, helmify.Values, error) {
	for i := range containers {
		containerName := strcase.ToLowerCamel((containers[i].(map[string]interface{})["name"]).(string))
		var valuePath []string
		if containerName == objName || containerName == "" {
			valuePath = []string{objName}
		} else {
			valuePath = []string{objName, containerName}
		}
		valuePathStr := strings.Join(valuePath, ".")

		res, exists, err := unstructured.NestedMap(values, append(valuePath, "resources")...)
		if err != nil {
			return nil, nil, err
		}
		if exists && len(res) > 0 {
			err = unstructured.SetNestedField(containers[i].(map[string]interface{}), fmt.Sprintf(`{{- toYaml .Values.%s.resources | nindent %d }}`, valuePathStr, nindent+2), "resources")
			if err != nil {
				return nil, nil, err
			}
		}

		args, exists, err := unstructured.NestedStringSlice(containers[i].(map[string]interface{}), "args")
		if err != nil {
			return nil, nil, err
		}
		if exists && len(args) > 0 {
			err = unstructured.SetNestedField(containers[i].(map[string]interface{}), fmt.Sprintf(`{{- toYaml .Values.%s.args | nindent %d }}`, valuePathStr, nindent), "args")
			if err != nil {
				return nil, nil, err
			}

			err = unstructured.SetNestedStringSlice(values, args, append(valuePath, "args")...)
			if err != nil {
				return nil, nil, fmt.Errorf("%w: unable to set deployment value field", err)
			}
		}
	}
	return containers, values, nil
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

	for i, c := range pod.InitContainers {
		processed, err := processPodContainer(name, appMeta, c, &values)
		if err != nil {
			return nil, err
		}
		pod.InitContainers[i] = processed
	}

	for _, v := range pod.Volumes {
		if v.ConfigMap != nil {
			v.ConfigMap.Name = appMeta.TemplatedName(v.ConfigMap.Name)
		}
		if v.Secret != nil {
			v.Secret.SecretName = appMeta.TemplatedName(v.Secret.SecretName)
		}
	}
	pod.ServiceAccountName = fmt.Sprintf(`{{ include "%s.serviceAccountName" . }}`, appMeta.ChartName())

	for i, s := range pod.ImagePullSecrets {
		pod.ImagePullSecrets[i].Name = appMeta.TemplatedName(s.Name)
	}

	return values, nil
}

func processPodContainer(name string, appMeta helmify.AppMetadata, c corev1.Container, values *helmify.Values) (corev1.Container, error) {
	index := strings.LastIndex(c.Image, ":")
	if strings.Contains(c.Image, "@") && strings.Count(c.Image, ":") >= 2 {
		last := strings.LastIndex(c.Image, ":")
		index = strings.LastIndex(c.Image[:last], ":")
	}
	if index < 0 {
		return c, fmt.Errorf("wrong image format: %q", c.Image)
	}
	repo, tag := c.Image[:index], c.Image[index+1:]
	containerName := strcase.ToLowerCamel(c.Name)
	
	var valuePath []string
	if containerName == name || containerName == "" {
		valuePath = []string{name}
	} else {
		valuePath = []string{name, containerName}
	}
	valuePathStr := strings.Join(valuePath, ".")

	c.Image = fmt.Sprintf("{{ .Values.%[1]s.image.repository }}:{{ .Values.%[1]s.image.tag | default .Chart.AppVersion }}", valuePathStr)

	err := unstructured.SetNestedField(*values, repo, append(valuePath, "image", "repository")...)
	if err != nil {
		return c, fmt.Errorf("%w: unable to set deployment value field", err)
	}
	err = unstructured.SetNestedField(*values, tag, append(valuePath, "image", "tag")...)
	if err != nil {
		return c, fmt.Errorf("%w: unable to set deployment value field", err)
	}

	c, err = processEnv(name, containerName, appMeta, c, values)
	if err != nil {
		return c, err
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
		Value: fmt.Sprintf("{{ quote .Values.%s }}", cluster.DomainKey),
	})
	for k, v := range c.Resources.Requests {
		err = unstructured.SetNestedField(*values, v.ToUnstructured(), append(valuePath, "resources", "requests", k.String())...)
		if err != nil {
			return c, fmt.Errorf("%w: unable to set container resources value", err)
		}
	}
	for k, v := range c.Resources.Limits {
		err = unstructured.SetNestedField(*values, v.ToUnstructured(), append(valuePath, "resources", "limits", k.String())...)
		if err != nil {
			return c, fmt.Errorf("%w: unable to set container resources value", err)
		}
	}

	if c.ImagePullPolicy != "" {
		err = unstructured.SetNestedField(*values, string(c.ImagePullPolicy), append(valuePath, "imagePullPolicy")...)
		if err != nil {
			return c, fmt.Errorf("%w: unable to set container imagePullPolicy", err)
		}
		c.ImagePullPolicy = corev1.PullPolicy(fmt.Sprintf("{{ .Values.%s.imagePullPolicy }}", valuePathStr))
	}
	return c, nil
}

func processEnv(name string, containerName string, appMeta helmify.AppMetadata, c corev1.Container, values *helmify.Values) (corev1.Container, error) {
	var valuePath []string
	if containerName == name || containerName == "" {
		valuePath = []string{name}
	} else {
		valuePath = []string{name, containerName}
	}
	valuePathStr := strings.Join(valuePath, ".")
	for i := 0; i < len(c.Env); i++ {
		if c.Env[i].ValueFrom != nil {
			switch {
			case c.Env[i].ValueFrom.SecretKeyRef != nil:
				c.Env[i].ValueFrom.SecretKeyRef.Name = appMeta.TemplatedName(c.Env[i].ValueFrom.SecretKeyRef.Name)
			case c.Env[i].ValueFrom.ConfigMapKeyRef != nil:
				c.Env[i].ValueFrom.ConfigMapKeyRef.Name = appMeta.TemplatedName(c.Env[i].ValueFrom.ConfigMapKeyRef.Name)
			case c.Env[i].ValueFrom.FieldRef != nil, c.Env[i].ValueFrom.ResourceFieldRef != nil:
				// nothing to change here, keep the original value
			}
			continue
		}

		err := unstructured.SetNestedField(*values, c.Env[i].Value, append(valuePath, "env", strcase.ToLowerCamel(strings.ToLower(c.Env[i].Name)))...)
		if err != nil {
			return c, fmt.Errorf("%w: unable to set deployment value field", err)
		}
		c.Env[i].Value = fmt.Sprintf("{{ .Values.%s.env.%s }}", valuePathStr, strcase.ToLowerCamel(strings.ToLower(c.Env[i].Name)))
	}
	return c, nil
}

// AddReloadingAnnotations scans the PodSpec for ConfigMap and Secret references, and injects Helm checksum
// annotations into the provided map so that pods restart when configurations change.
func AddReloadingAnnotations(appMeta helmify.AppMetadata, annotations map[string]string, spec *corev1.PodSpec) map[string]string {
	if annotations == nil {
		annotations = make(map[string]string)
	}

	configMaps := make(map[string]struct{})
	secrets := make(map[string]struct{})

	for _, v := range spec.Volumes {
		if v.ConfigMap != nil {
			configMaps[v.ConfigMap.Name] = struct{}{}
		}
		if v.Secret != nil {
			secrets[v.Secret.SecretName] = struct{}{}
		}
	}

	scanContainerRef := func(c corev1.Container) {
		for _, e := range c.EnvFrom {
			if e.ConfigMapRef != nil {
				configMaps[e.ConfigMapRef.Name] = struct{}{}
			}
			if e.SecretRef != nil {
				secrets[e.SecretRef.Name] = struct{}{}
			}
		}
		for _, e := range c.Env {
			if e.ValueFrom != nil {
				if e.ValueFrom.ConfigMapKeyRef != nil {
					configMaps[e.ValueFrom.ConfigMapKeyRef.Name] = struct{}{}
				}
				if e.ValueFrom.SecretKeyRef != nil {
					secrets[e.ValueFrom.SecretKeyRef.Name] = struct{}{}
				}
			}
		}
	}

	for _, c := range spec.Containers {
		scanContainerRef(c)
	}
	for _, c := range spec.InitContainers {
		scanContainerRef(c)
	}

	for cm := range configMaps {
		trimmed := appMeta.TrimName(cm)
		annotations["checksum/config-"+trimmed] = fmt.Sprintf(`{{ include (print $.Template.BasePath "/%s.yaml") . | sha256sum }}`, trimmed)
	}
	for sec := range secrets {
		trimmed := appMeta.TrimName(sec)
		annotations["checksum/secret-"+trimmed] = fmt.Sprintf(`{{ include (print $.Template.BasePath "/%s.yaml") . | sha256sum }}`, trimmed)
	}

	return annotations
}
