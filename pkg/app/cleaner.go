package app

import (
	"strings"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

// cleanKomposeMetadata deeply iterates through a K8s object and removes any 
// label, annotation, or selector key starting with "io.kompose."
func cleanKomposeMetadata(obj *unstructured.Unstructured) {
	cleanMap(obj.Object)
}

func cleanMap(obj interface{}) {
	switch m := obj.(type) {
	case map[string]interface{}:
		// Clean the common metadata/selector containers if they exist at this level
		cleanKeys(m, "labels")
		cleanKeys(m, "annotations")
		cleanKeys(m, "selector")
		cleanKeys(m, "matchLabels")
		
		// Recurse into all maps
		for _, v := range m {
			cleanMap(v)
		}
	case []interface{}:
		for _, v := range m {
			cleanMap(v)
		}
	}
}

func cleanKeys(m map[string]interface{}, containerKey string) {
	if container, ok := m[containerKey].(map[string]interface{}); ok {
		for k := range container {
			if strings.HasPrefix(k, "io.kompose.") || strings.HasPrefix(k, "kompose.cmd") {
				delete(container, k)
			}
		}
		// If the container is now empty, we delete the container entirely
		// to keep the yaml clean, unless it's an empty selector which some apps might require?
		// Typically an empty label map is fine to be deleted so it doesn't render "labels: {}"
		if len(container) == 0 {
			delete(m, containerKey)
		}
	}
}
