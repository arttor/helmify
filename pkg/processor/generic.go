package processor

import (
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"strings"
)

// ExtractOperatorName - tries to define operator name by removing "-system" suffix from namespace
// Based on idea that operator-SDK adds namespace: <operator-name>-system to all generated objects.
func ExtractOperatorName(obj *unstructured.Unstructured) string {
	if strings.HasSuffix(obj.GetNamespace(), "-system") {
		return strings.TrimSuffix(obj.GetNamespace(), "-system")
	}
	return ""
}
