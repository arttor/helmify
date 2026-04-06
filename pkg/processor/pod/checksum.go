package pod

import (
	"fmt"
	"sort"
	"strings"

	"github.com/arttor/helmify/pkg/helmify"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

// WorkloadGVKs lists the resource kinds whose pod templates can get checksum annotations.
var WorkloadGVKs = map[schema.GroupVersionKind]bool{
	{Group: "apps", Version: "v1", Kind: "Deployment"}:  true,
	{Group: "apps", Version: "v1", Kind: "DaemonSet"}:   true,
	{Group: "apps", Version: "v1", Kind: "StatefulSet"}:  true,
}

// ChecksumAnnotations extracts the PodSpec from a workload object, scans it for
// references to chart-local ConfigMaps and Secrets, and returns checksum annotation
// lines. configMapFiles and secretFiles map original object names to their actual
// template filenames on disk (e.g. "my-app-config" -> "input.yaml").
//
// Returns empty string if the object is not a supported workload or has no
// chart-local config references.
func ChecksumAnnotations(appMeta helmify.AppMetadata, obj *unstructured.Unstructured, configMapFiles, secretFiles map[string]string) string {
	podSpec := extractPodSpec(obj)
	if podSpec == nil {
		return ""
	}

	configMaps, secrets := collectConfigRefs(appMeta, *podSpec)
	if len(configMaps) == 0 && len(secrets) == 0 {
		return ""
	}

	var annotations []string
	for name := range configMaps {
		trimmed := appMeta.TrimName(name)
		annotations = append(annotations, checksumAnnotation("configmap", trimmed, configMapFiles[name]))
	}
	for name := range secrets {
		trimmed := appMeta.TrimName(name)
		annotations = append(annotations, checksumAnnotation("secret", trimmed, secretFiles[name]))
	}
	sort.Strings(annotations)

	return strings.Join(annotations, "\n")
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

// extractPodSpec extracts the PodSpec from a supported workload object.
// Returns nil if the object is not a supported workload type.
func extractPodSpec(obj *unstructured.Unstructured) *corev1.PodSpec {
	switch obj.GroupVersionKind() {
	case schema.GroupVersionKind{Group: "apps", Version: "v1", Kind: "Deployment"}:
		var d appsv1.Deployment
		if err := runtime.DefaultUnstructuredConverter.FromUnstructured(obj.Object, &d); err != nil {
			return nil
		}
		return &d.Spec.Template.Spec
	case schema.GroupVersionKind{Group: "apps", Version: "v1", Kind: "DaemonSet"}:
		var d appsv1.DaemonSet
		if err := runtime.DefaultUnstructuredConverter.FromUnstructured(obj.Object, &d); err != nil {
			return nil
		}
		return &d.Spec.Template.Spec
	case schema.GroupVersionKind{Group: "apps", Version: "v1", Kind: "StatefulSet"}:
		var s appsv1.StatefulSet
		if err := runtime.DefaultUnstructuredConverter.FromUnstructured(obj.Object, &s); err != nil {
			return nil
		}
		return &s.Spec.Template.Spec
	}
	return nil
}

func checksumAnnotation(kind, trimmedName, filename string) string {
	return fmt.Sprintf(`checksum/%s/%s: {{ include (print $.Template.BasePath "/%s") . | sha256sum }}`, kind, trimmedName, filename)
}
