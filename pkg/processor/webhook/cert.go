package webhook

import (
	"bytes"
	"fmt"
	"io"
	"strings"

	"github.com/arttor/helmify/pkg/helmify"
	yamlformat "github.com/arttor/helmify/pkg/yaml"
	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/yaml"
)

const (
	certTempl = `apiVersion: cert-manager.io/v1
kind: Certificate
metadata:
  name: {{ include "%[1]s.fullname" . }}-%[2]s
  labels:
  {{- include "%[1]s.labels" . | nindent 4 }}
spec:
%[3]s`
)

var certGVC = schema.GroupVersionKind{
	Group:   "cert-manager.io",
	Version: "v1",
	Kind:    "Certificate",
}

// Certificate creates processor for k8s Certificate resource.
func Certificate() helmify.Processor {
	return &cert{}
}

type cert struct{}

// Process k8s Certificate object into template. Returns false if not capable of processing given resource type.
func (c cert) Process(appMeta helmify.AppMetadata, obj *unstructured.Unstructured) (bool, helmify.Template, error) {
	if obj.GroupVersionKind() != certGVC {
		return false, nil, nil
	}
	name := appMeta.TrimName(obj.GetName())

	dnsNames, _, err := unstructured.NestedSlice(obj.Object, "spec", "dnsNames")
	if err != nil {
		return true, nil, errors.Wrap(err, "unable get cert dnsNames")
	}

	processedDnsNames := []interface{}{}
	for _, dnsName := range dnsNames {
		dns := dnsName.(string)
		templatedDns := appMeta.TemplatedString(dns)
		processedDns := strings.ReplaceAll(templatedDns, appMeta.Namespace(), "{{ .Release.Namespace }}")
		processedDnsNames = append(processedDnsNames, processedDns)
	}
	err = unstructured.SetNestedSlice(obj.Object, processedDnsNames, "spec", "dnsNames")
	if err != nil {
		return true, nil, errors.Wrap(err, "unable set cert dnsNames")
	}

	issName, _, err := unstructured.NestedString(obj.Object, "spec", "issuerRef", "name")
	if err != nil {
		return true, nil, errors.Wrap(err, "unable get cert issuerRef")
	}
	issName = appMeta.TemplatedName(issName)
	err = unstructured.SetNestedField(obj.Object, issName, "spec", "issuerRef", "name")
	if err != nil {
		return true, nil, errors.Wrap(err, "unable set cert issuerRef")
	}
	spec, _ := yaml.Marshal(obj.Object["spec"])
	spec = yamlformat.Indent(spec, 2)
	spec = bytes.TrimRight(spec, "\n ")
	res := fmt.Sprintf(certTempl, appMeta.ChartName(), name, string(spec))
	return true, &certResult{
		name: name,
		data: []byte(res),
	}, nil
}

type certResult struct {
	name string
	data []byte
}

func (r *certResult) Filename() string {
	return r.name + ".yaml"
}

func (r *certResult) Values() helmify.Values {
	return helmify.Values{}
}

func (r *certResult) Write(writer io.Writer) error {
	_, err := writer.Write(r.data)
	return err
}
