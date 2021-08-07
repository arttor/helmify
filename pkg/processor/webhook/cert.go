package webhook

import (
	"bytes"
	"fmt"
	"github.com/arttor/helmify/pkg/helmify"
	yamlformat "github.com/arttor/helmify/pkg/yaml"
	"github.com/pkg/errors"
	"io"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/yaml"
	"strings"
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

var (
	certGVC = schema.GroupVersionKind{
		Group:   "cert-manager.io",
		Version: "v1",
		Kind:    "Certificate",
	}
)

func Certificate() helmify.Processor {
	return &cert{}
}

type cert struct {
}

func (c cert) Process(info helmify.ChartInfo, obj *unstructured.Unstructured) (bool, helmify.Template, error) {
	if obj.GroupVersionKind() != certGVC {
		return false, nil, nil
	}
	name := strings.TrimPrefix(obj.GetName(), info.OperatorName+"-")
	fullnameTempl := fmt.Sprintf(`{{ include "%s.fullname" . }}`, info.ChartName)

	dnsNames, _, err := unstructured.NestedSlice(obj.Object, "spec", "dnsNames")
	if err != nil {
		return true, nil, errors.Wrap(err, "unable get cert dnsNames")
	}
	for i, dns := range dnsNames {
		dns = strings.ReplaceAll(dns.(string), info.OperatorNamespace, "{{ .Release.Namespace }}")
		dns = strings.ReplaceAll(dns.(string), info.OperatorName, fullnameTempl)
		dnsNames[i] = dns
	}
	err = unstructured.SetNestedSlice(obj.Object, dnsNames, "spec", "dnsNames")
	if err != nil {
		return true, nil, errors.Wrap(err, "unable set cert dnsNames")
	}

	issName, _, err := unstructured.NestedString(obj.Object, "spec", "issuerRef", "name")
	if err != nil {
		return true, nil, errors.Wrap(err, "unable get cert issuerRef")
	}
	issName = strings.ReplaceAll(issName, info.OperatorName, fullnameTempl)
	err = unstructured.SetNestedField(obj.Object, issName, "spec", "issuerRef", "name")

	spec, _ := yaml.Marshal(obj.Object["spec"])
	spec = yamlformat.Indent(spec, 2)
	spec = bytes.TrimRight(spec, "\n ")
	res := fmt.Sprintf(certTempl, info.ChartName, name, string(spec))
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
