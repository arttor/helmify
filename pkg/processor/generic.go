package processor

import (
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"strings"
)

func GetOperatorName(obj *unstructured.Unstructured) string {
	if strings.HasSuffix(obj.GetNamespace(), "-system") {
		return strings.TrimSuffix(obj.GetNamespace(), "-system")
	}
	return ""
}
