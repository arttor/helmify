package pod

import (
	"fmt"
	"sort"
	"strings"

	"github.com/arttor/helmify/pkg/helmify"
	corev1 "k8s.io/api/core/v1"
)

// ChecksumAnnotations scans a PodSpec for references to chart-local ConfigMaps
// and Secrets and returns a formatted annotations YAML string ready for inclusion
// in a pod template. configMapFiles and secretFiles map original object names to
// their template filenames (e.g. "my-app-config" -> "config.yaml").
//
// The indent parameter controls the base indentation of the "annotations:" key.
// Returns empty string if no chart-local config references are found.
func ChecksumAnnotations(appMeta helmify.AppMetadata, spec corev1.PodSpec, configMapFiles, secretFiles map[string]string, indent int) string {
	configMaps, secrets := collectConfigRefs(appMeta, spec)
	if len(configMaps) == 0 && len(secrets) == 0 {
		return ""
	}

	var lines []string
	for name := range configMaps {
		trimmed := appMeta.TrimName(name)
		lines = append(lines, checksumAnnotation("configmap", trimmed, configMapFiles[name]))
	}
	for name := range secrets {
		trimmed := appMeta.TrimName(name)
		lines = append(lines, checksumAnnotation("secret", trimmed, secretFiles[name]))
	}
	sort.Strings(lines)

	valueIndent := strings.Repeat(" ", indent+2)
	var result []string
	for _, line := range lines {
		result = append(result, valueIndent+line)
	}
	return strings.Repeat(" ", indent) + "annotations:\n" + strings.Join(result, "\n")
}

// collectConfigRefs scans a PodSpec for references to ConfigMaps and Secrets
// that are part of the chart.
func collectConfigRefs(appMeta helmify.AppMetadata, spec corev1.PodSpec) (configMaps, secrets map[string]struct{}) {
	configMaps = map[string]struct{}{}
	secrets = map[string]struct{}{}

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

	return configMaps, secrets
}

func checksumAnnotation(kind, trimmedName, filename string) string {
	return fmt.Sprintf(`checksum/%s/%s: {{ include (print $.Template.BasePath "/%s") . | sha256sum }}`, kind, trimmedName, filename)
}
