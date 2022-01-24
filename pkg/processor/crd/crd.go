package crd

import (
	"bytes"
	"fmt"
	"io"
	"strings"

	"github.com/pkg/errors"
	v1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/yaml"

	"github.com/arttor/helmify/pkg/helmify"
	yamlformat "github.com/arttor/helmify/pkg/yaml"
)

const crdTeml = `apiVersion: apiextensions.k8s.io/v1
kind: CustomResourceDefinition
metadata:
  name: %[1]s
  annotations:
    cert-manager.io/inject-ca-from: {{ .Release.Namespace }}/{{ include "%[2]s.fullname" . }}-%[4]s
    controller-gen.kubebuilder.io/version: v0.7.0
  labels:
  {{- include "%[2]s.labels" . | nindent 4 }}
spec:
%[3]s
status:
  acceptedNames:
    kind: ""
    plural: ""
  conditions: []
  storedVersions: []`

var crdGVC = schema.GroupVersionKind{
	Group:   "apiextensions.k8s.io",
	Version: "v1",
	Kind:    "CustomResourceDefinition",
}

// New creates processor for k8s CustomResourceDefinition resource.
func New() helmify.Processor {
	return &crd{}
}

type crd struct{}

// Process k8s CustomResourceDefinition object into template. Returns false if not capable of processing given resource type.
func (c crd) Process(appMeta helmify.AppMetadata, obj *unstructured.Unstructured) (bool, helmify.Template, error) {
	if obj.GroupVersionKind() != crdGVC {
		return false, nil, nil
	}

	certName, _, err := unstructured.NestedString(obj.Object, "metadata", "annotations", "cert-manager.io/inject-ca-from")
	if err != nil {
		return true, nil, errors.Wrap(err, "unable get crd certName")
	}
	certName = strings.TrimPrefix(certName, appMeta.Namespace()+"/"+appMeta.ChartName()+"-")

	specUnstr, ok, err := unstructured.NestedMap(obj.Object, "spec")
	if err != nil || !ok {
		return true, nil, errors.Wrap(err, "unable to create crd template")
	}

	spec := v1.CustomResourceDefinitionSpec{}
	err = runtime.DefaultUnstructuredConverter.FromUnstructured(specUnstr, &spec)
	if err != nil {
		return true, nil, errors.Wrap(err, "unable to cast to crd spec")
	}

	if spec.Conversion != nil {
		conv := spec.Conversion
		if conv.Strategy == v1.WebhookConverter {
			wh := conv.Webhook
			if wh != nil {
				wh.ClientConfig.Service.Name = appMeta.TemplatedName(wh.ClientConfig.Service.Name)
				wh.ClientConfig.Service.Namespace = strings.ReplaceAll(wh.ClientConfig.Service.Namespace, appMeta.Namespace(), `{{ .Release.Namespace }}`)
			}
		}
	}

	versions, _ := yaml.Marshal(spec)
	versions = yamlformat.Indent(versions, 2)
	versions = bytes.TrimRight(versions, "\n ")

	res := fmt.Sprintf(crdTeml, obj.GetName(), appMeta.ChartName(), string(versions), certName)
	name, _, err := unstructured.NestedString(obj.Object, "spec", "names", "singular")
	if err != nil || !ok {
		return true, nil, errors.Wrap(err, "unable to create crd template")
	}
	return true, &result{
		name: name + "-crd.yaml",
		data: []byte(res),
	}, nil
}

type result struct {
	name string
	data []byte
}

func (r *result) Filename() string {
	return r.name
}

func (r *result) Values() helmify.Values {
	return helmify.Values{}
}

func (r *result) Write(writer io.Writer) error {
	_, err := writer.Write(r.data)
	return err
}
