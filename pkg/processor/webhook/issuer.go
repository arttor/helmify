package webhook

import (
	"bytes"
	"fmt"
	"io"

	"github.com/arttor/helmify/pkg/helmify"
	"github.com/arttor/helmify/pkg/processor"
	yamlformat "github.com/arttor/helmify/pkg/yaml"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/yaml"
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
	tmpl := ""
	if appMeta.Config().CertManagerAsSubchart {
		annotations := obj.GetAnnotations()
		if annotations == nil {
			annotations = make(map[string]string)
		}
		annotations["helm.sh/hook"] = "post-install,post-upgrade"
		annotations["helm.sh/hook-weight"] = "1"
		obj.SetAnnotations(annotations)
	}
	tmpl, err := processor.ProcessObjMeta(appMeta, obj)
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
	return true, &issResult{
		name: name,
		data: []byte(res),
	}, nil
}

type issResult struct {
	name   string
	data   []byte
	values helmify.Values
}

func (r *issResult) Filename() string {
	return r.name + ".yaml"
}

func (r *issResult) Values() helmify.Values {
	return r.values
}

func (r *issResult) Write(writer io.Writer) error {
	_, err := writer.Write(r.data)
	return err
}
