package pod

import (
	"fmt"
	"sort"
	"strings"

	"github.com/arttor/helmify/pkg/helmify"
	corev1 "k8s.io/api/core/v1"
)

// ChecksumAnnotations scans a PodSpec for references to ConfigMaps and Secrets
// that are part of the chart, and returns checksum annotation lines to be added
// to the pod template metadata. This ensures pods are restarted when referenced
// ConfigMaps or Secrets change.
//
// configMapFiles and secretFiles map original object names to their actual
// template filenames on disk (e.g. "my-app-config" -> "input.yaml").
func ChecksumAnnotations(appMeta helmify.AppMetadata, spec corev1.PodSpec, configMapFiles, secretFiles map[string]string) string {
	configMaps := map[string]struct{}{}
	secrets := map[string]struct{}{}

	collectFromContainers := func(containers []corev1.Container) {
		for _, c := range containers {
			for _, e := range c.EnvFrom {
				if e.ConfigMapRef != nil && appMeta.HasConfigMap(e.ConfigMapRef.Name) {
					configMaps[e.ConfigMapRef.Name] = struct{}{}
				}
				if e.SecretRef != nil && appMeta.HasSecret(e.SecretRef.Name) {
					secrets[e.SecretRef.Name] = struct{}{}
				}
			}
			for _, e := range c.Env {
				if e.ValueFrom == nil {
					continue
				}
				if e.ValueFrom.ConfigMapKeyRef != nil && appMeta.HasConfigMap(e.ValueFrom.ConfigMapKeyRef.Name) {
					configMaps[e.ValueFrom.ConfigMapKeyRef.Name] = struct{}{}
				}
				if e.ValueFrom.SecretKeyRef != nil && appMeta.HasSecret(e.ValueFrom.SecretKeyRef.Name) {
					secrets[e.ValueFrom.SecretKeyRef.Name] = struct{}{}
				}
			}
		}
	}

	collectFromContainers(spec.Containers)
	collectFromContainers(spec.InitContainers)

	for _, v := range spec.Volumes {
		if v.ConfigMap != nil && appMeta.HasConfigMap(v.ConfigMap.Name) {
			configMaps[v.ConfigMap.Name] = struct{}{}
		}
		if v.Secret != nil && appMeta.HasSecret(v.Secret.SecretName) {
			secrets[v.Secret.SecretName] = struct{}{}
		}
		if v.Projected != nil {
			for _, src := range v.Projected.Sources {
				if src.ConfigMap != nil && appMeta.HasConfigMap(src.ConfigMap.Name) {
					configMaps[src.ConfigMap.Name] = struct{}{}
				}
				if src.Secret != nil && appMeta.HasSecret(src.Secret.Name) {
					secrets[src.Secret.Name] = struct{}{}
				}
			}
		}
	}

	if len(configMaps) == 0 && len(secrets) == 0 {
		return ""
	}

	var annotations []string
	for name := range configMaps {
		trimmed := appMeta.TrimName(name)
		filename := configMapFiles[name]
		annotations = append(annotations, checksumAnnotation("configmap", trimmed, filename))
	}
	for name := range secrets {
		trimmed := appMeta.TrimName(name)
		filename := secretFiles[name]
		annotations = append(annotations, checksumAnnotation("secret", trimmed, filename))
	}
	sort.Strings(annotations)

	return strings.Join(annotations, "\n")
}

func checksumAnnotation(kind, trimmedName, filename string) string {
	return fmt.Sprintf(`checksum/%s/%s: {{ include (print $.Template.BasePath "/%s") . | sha256sum }}`, kind, trimmedName, filename)
}
