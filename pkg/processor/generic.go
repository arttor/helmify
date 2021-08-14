package processor

import (
	"github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"strings"
)

var nsGVK = schema.GroupVersionKind{
	Group:   "",
	Version: "v1",
	Kind:    "Namespace",
}

var crdGVK = schema.GroupVersionKind{
	Group:   "apiextensions.k8s.io",
	Version: "v1",
	Kind:    "CustomResourceDefinition",
}

// ExtractOperatorName - tries to define operator name as the longest common prefix across all objects names
// except CRDs and namespaces
func ExtractOperatorName(obj *unstructured.Unstructured, prevName string) string {
	if obj.GroupVersionKind() == crdGVK || obj.GroupVersionKind() == nsGVK {
		return prevName
	}
	if prevName == "" {
		return obj.GetName()
	}
	common := commonPrefix(obj.GetName(), prevName)
	if common == "" {
		logrus.WithFields(logrus.Fields{
			"prev": prevName,
			"curr": obj.GetName(),
		}).Error("unable to define operator name as common object name prefix")
		return prevName
	}
	return strings.TrimSuffix(common, "-")
}

func commonPrefix(one, two string) string {
	runes1 := []rune(one)
	runes2 := []rune(two)
	min := len(runes1)
	if min > len(runes2) {
		min = len(runes2)
	}
	for i := 0; i < min; i++ {
		if runes1[i] != runes2[i] {
			return string(runes1[:i])
		}
	}
	return string(runes1[:min])
}

// ExtractOperatorNamespace returns name if given object is a namespace
func ExtractOperatorNamespace(obj *unstructured.Unstructured) string {
	if obj.GroupVersionKind() != nsGVK {
		return ""
	}
	return obj.GetName()
}
