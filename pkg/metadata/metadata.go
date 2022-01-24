package metadata

import (
	"fmt"
	"strings"

	"github.com/arttor/helmify/pkg/helmify"
	"github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

const nameTeml = `{{ include "%s.fullname" . }}-%s`

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

func New(chartName string) *Service {
	return &Service{chartName: chartName, names: make(map[string]struct{})}
}

type Service struct {
	commonPrefix string
	namespace    string
	chartName    string
	names        map[string]struct{}
}

// TrimName - tries to trim app common prefix for object name if detected.
// If no common prefix - returns name as it is.
// It is better to trim common prefix because Helm also adds release name as common prefix.
func (a *Service) TrimName(objName string) string {
	trimmed := strings.TrimPrefix(objName, a.commonPrefix)
	trimmed = strings.TrimLeft(trimmed, "-./_ ")
	if trimmed == "" {
		return objName
	}
	return trimmed
}

var _ helmify.AppMetadata = &Service{}

// Load processed objects one-by-one before actual processing to define app namespace, name common prefix and
// other app meta information.
func (a *Service) Load(obj *unstructured.Unstructured) {
	a.names[obj.GetName()] = struct{}{}
	a.commonPrefix = detectCommonPrefix(obj, a.commonPrefix)
	objNs := extractAppNamespace(obj)
	if objNs == "" {
		return
	}
	if a.namespace != "" && a.namespace != objNs {
		logrus.Warnf("Two different namespaces for app detected: %s and %s. Resulted char will have single namespace.", objNs, a.namespace)
	}
	a.namespace = objNs
}

// Namespace returns detected app namespace.
func (a *Service) Namespace() string {
	return a.namespace
}

// ChartName returns ChartName.
func (a *Service) ChartName() string {
	return a.chartName
}

// TemplatedName - converts object name to its Helm templated representation.
// Adds chart fullname prefix from _helpers.tpl
func (a *Service) TemplatedName(name string) string {
	_, contains := a.names[name]
	if !contains {
		// template only app objects
		return name
	}
	name = a.TrimName(name)
	return fmt.Sprintf(nameTeml, a.chartName, name)
}

func (a *Service) TemplatedString(str string) string {
	name := a.TrimName(str)
	return fmt.Sprintf(nameTeml, a.chartName, name)
}

func extractAppNamespace(obj *unstructured.Unstructured) string {
	if obj.GroupVersionKind() == nsGVK {
		return obj.GetName()
	}
	return obj.GetNamespace()
}

func detectCommonPrefix(obj *unstructured.Unstructured, prevName string) string {
	if obj.GroupVersionKind() == crdGVK || obj.GroupVersionKind() == nsGVK {
		return prevName
	}
	if prevName == "" {
		return obj.GetName()
	}
	return commonPrefix(obj.GetName(), prevName)
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
