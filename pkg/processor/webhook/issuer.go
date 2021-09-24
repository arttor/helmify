package webhook

import (
	"bytes"
	"fmt"
	"io"

	"github.com/arttor/helmify/pkg/helmify"
	yamlformat "github.com/arttor/helmify/pkg/yaml"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/yaml"
)

const (
	issuerTempl = `apiVersion: cert-manager.io/v1
kind: Issuer
metadata:
  name: {{ include "%[1]s.fullname" . }}-%[2]s
  labels:
  {{- include "%[1]s.labels" . | nindent 4 }}
spec:
%[3]s`
)

var issuerGVC = schema.GroupVersionKind{
	Group:   "cert-manager.io",
	Version: "v1",
	Kind:    "Issuer",
}

// Issuer creates processor for k8s Issuer resource.
func Issuer() helmify.Processor {
	return &issuer{}
}

type issuer struct{}

// Process k8s Issuer object into template. Returns false if not capable of processing given resource type.
func (i issuer) Process(appMeta helmify.AppMetadata, obj *unstructured.Unstructured) (bool, helmify.Template, error) {
	if obj.GroupVersionKind() != issuerGVC {
		return false, nil, nil
	}
	name := appMeta.TrimName(obj.GetName())
	spec, _ := yaml.Marshal(obj.Object["spec"])
	spec = yamlformat.Indent(spec, 2)
	spec = bytes.TrimRight(spec, "\n ")
	res := fmt.Sprintf(issuerTempl, appMeta.ChartName(), name, string(spec))
	return true, &issResult{
		name: name,
		data: []byte(res),
	}, nil
}

type issResult struct {
	name string
	data []byte
}

func (r *issResult) Filename() string {
	return r.name + ".yaml"
}

func (r *issResult) Values() helmify.Values {
	return helmify.Values{}
}

func (r *issResult) Write(writer io.Writer) error {
	_, err := writer.Write(r.data)
	return err
}
