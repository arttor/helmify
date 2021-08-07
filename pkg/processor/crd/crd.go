package crd

import (
	"bytes"
	"fmt"
	"github.com/arttor/helmify/pkg/helmify"
	yamlformat "github.com/arttor/helmify/pkg/yaml"
	"github.com/pkg/errors"
	"io"
	extensionv1beta1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/yaml"
)

var crdTeml = `apiVersion: apiextensions.k8s.io/v1
kind: CustomResourceDefinition
metadata:
  name: {{ .Values.crd.%[1]s.plural }}.{{ .Values.crd.%[1]s.group }}
  annotations:
    controller-gen.kubebuilder.io/version: v0.4.1
  labels:
  {{- include "%[2]s.labels" . | nindent 4 }}
spec:
  group: {{ .Values.crd.%[1]s.group }}
  names:
    kind: {{ .Values.crd.%[1]s.kind }}
    listKind: {{ .Values.crd.%[1]s.listKind }}
    plural: {{ .Values.crd.%[1]s.plural }}
    singular: {{ .Values.crd.%[1]s.singular }}
  scope: {{ .Values.crd.%[1]s.scope }}
  versions:
%[3]s
status:
  acceptedNames:
    kind: ""
    plural: ""
  conditions: []
  storedVersions: []
`

var crdGVC = schema.GroupVersionKind{
	Group:   "apiextensions.k8s.io",
	Version: "v1",
	Kind:    "CustomResourceDefinition",
}

func New() helmify.Processor {
	return &deployment{}
}

type deployment struct {
}

func (d deployment) Process(info helmify.ChartInfo, obj *unstructured.Unstructured) (bool, helmify.Template, error) {
	if obj.GroupVersionKind() != crdGVC {
		return false, nil, nil
	}

	crd := extensionv1beta1.CustomResourceDefinition{}
	err := runtime.DefaultUnstructuredConverter.FromUnstructured(obj.Object, &crd)
	if err != nil {
		return true, nil, errors.Wrap(err, "unable to cast unstructured to crd")
	}

	versionsUnstr, ok, err := unstructured.NestedSlice(obj.Object, "spec", "versions")
	if err != nil || !ok {
		return true, nil, errors.Wrap(err, "unable to create crd template")
	}
	versions, _ := yaml.Marshal(versionsUnstr)
	versions = yamlformat.Indent(versions, 2)
	versions = bytes.TrimRight(versions, "\n ")

	res := fmt.Sprintf(crdTeml, crd.Spec.Names.Singular, info.ChartName, string(versions))

	values := helmify.Values{}
	err = unstructured.SetNestedField(values, crd.Spec.Group, "crd", crd.Spec.Names.Singular, "group")
	if err != nil {
		return true, nil, errors.Wrap(err, "unable to set crd value field")
	}
	err = unstructured.SetNestedField(values, crd.Spec.Names.Kind, "crd", crd.Spec.Names.Singular, "kind")
	if err != nil {
		return true, nil, errors.Wrap(err, "unable to set crd value field")
	}
	err = unstructured.SetNestedField(values, crd.Spec.Names.ListKind, "crd", crd.Spec.Names.Singular, "listKind")
	if err != nil {
		return true, nil, errors.Wrap(err, "unable to set crd value field")
	}
	err = unstructured.SetNestedField(values, crd.Spec.Names.Singular, "crd", crd.Spec.Names.Singular, "singular")
	if err != nil {
		return true, nil, errors.Wrap(err, "unable to set crd value field")
	}
	err = unstructured.SetNestedField(values, crd.Spec.Names.Plural, "crd", crd.Spec.Names.Singular, "plural")
	if err != nil {
		return true, nil, errors.Wrap(err, "unable to set crd value field")
	}
	err = unstructured.SetNestedField(values, string(crd.Spec.Scope), "crd", crd.Spec.Names.Singular, "scope")
	if err != nil {
		return true, nil, errors.Wrap(err, "unable to set crd value field")
	}
	return true, &result{
		name:   crd.Spec.Names.Singular + "-crd.yaml",
		values: values,
		data:   []byte(res),
	}, nil
}

type result struct {
	name   string
	data   []byte
	values helmify.Values
}

func (r *result) Filename() string {
	return r.name
}

func (r *result) Values() helmify.Values {
	return r.values
}

func (r *result) Write(writer io.Writer) error {
	_, err := writer.Write(r.data)
	return err
}

func (r *result) PostProcess(helmify.Values) {
}
