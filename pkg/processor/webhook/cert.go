package webhook

import (
	"bytes"
	"fmt"
	"io"
	"strings"

	"github.com/arttor/helmify/pkg/cluster"
	"github.com/arttor/helmify/pkg/helmify"
	"github.com/arttor/helmify/pkg/processor"
	yamlformat "github.com/arttor/helmify/pkg/yaml"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/yaml"
)

const (
	WebhookHeader = `{{- if .Values.webhook.enabled }}`
	WebhookFooter = `{{- end }}`
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
		return true, nil, fmt.Errorf("%w: unable get cert dnsNames", err)
	}

	processedDnsNames := []interface{}{}
	for _, dnsName := range dnsNames {
		dns := dnsName.(string)
		templatedDns := appMeta.TemplatedString(dns)
		processedDns := strings.ReplaceAll(templatedDns, appMeta.Namespace(), "{{ .Release.Namespace }}")
		processedDns = strings.ReplaceAll(processedDns, cluster.DefaultDomain, fmt.Sprintf("{{ .Values.%s }}", cluster.DomainKey))
		processedDnsNames = append(processedDnsNames, processedDns)
	}
	err = unstructured.SetNestedSlice(obj.Object, processedDnsNames, "spec", "dnsNames")
	if err != nil {
		return true, nil, fmt.Errorf("%w: unable set cert dnsNames", err)
	}

	issName, _, err := unstructured.NestedString(obj.Object, "spec", "issuerRef", "name")
	if err != nil {
		return true, nil, fmt.Errorf("%w: unable get cert issuerRef", err)
	}
	issName = appMeta.TemplatedName(issName)
	err = unstructured.SetNestedField(obj.Object, issName, "spec", "issuerRef", "name")
	if err != nil {
		return true, nil, fmt.Errorf("%w: unable set cert issuerRef", err)
	}
	spec, _ := yaml.Marshal(obj.Object["spec"])
	spec = yamlformat.Indent(spec, 2)
	spec = bytes.TrimRight(spec, "\n ")
	tmpl := ""

	if appMeta.Config().CertManagerAsSubchart {
		annotations := obj.GetAnnotations()
		if annotations == nil {
			annotations = make(map[string]string)
		}
		annotations["helm.sh/hook"] = "post-install,post-upgrade"
		annotations["helm.sh/hook-weight"] = "2"
		obj.SetAnnotations(annotations)
	}

	tmpl, err = processor.ProcessObjMeta(appMeta, obj)
	if err != nil {
		return true, nil, err
	}

	values := helmify.Values{}
	if appMeta.Config().AddWebhookOption {
		// Add webhook.enabled value to values.yaml
		_, _ = values.Add(true, "webhook", "enabled")

		tmpl = fmt.Sprintf("%s\n%s\n%s", WebhookHeader, tmpl, WebhookFooter)
	}
	res := tmpl + "\nspec:\n" + string(spec)
	return true, &certResult{
		name:   name,
		data:   []byte(res),
		values: values,
	}, nil
}

type certResult struct {
	name   string
	data   []byte
	values helmify.Values
}

func (r *certResult) Filename() string {
	return r.name + ".yaml"
}

func (r *certResult) Values() helmify.Values {
	return r.values
}

func (r *certResult) Write(writer io.Writer) error {
	_, err := writer.Write(r.data)
	return err
}
