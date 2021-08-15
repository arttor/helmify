package internal

import (
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/serializer/yaml"
)

const (
	nsYaml = `apiVersion: v1
kind: Namespace
metadata:
  labels:
    control-plane: controller-manager
  name: my-operator-system`
	TestNsName = "my-operator-system"
)

var TestNs = GenerateObj(nsYaml)

func GenerateObj(objYaml string) *unstructured.Unstructured {
	obj := unstructured.Unstructured{}
	dec := yaml.NewDecodingSerializer(unstructured.UnstructuredJSONScheme)
	_, _, err := dec.Decode([]byte(objYaml), nil, &obj)
	if err != nil {
		panic(err)
	}
	return &obj
}
