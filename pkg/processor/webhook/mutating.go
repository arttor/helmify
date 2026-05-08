package webhook

import (
	"bytes"
	"fmt"
	"io"
	"strings"

	"github.com/arttor/helmify/pkg/helmify"
	v1 "k8s.io/api/admissionregistration/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/yaml"
)

const (
	mwhTempl = `apiVersion: admissionregistration.k8s.io/v1
kind: MutatingWebhookConfiguration
metadata:
  name: {{ include "%[1]s.fullname" . }}-%[2]s
  annotations:
    cert-manager.io/inject-ca-from: {{ .Release.Namespace }}/{{ include "%[1]s.fullname" . }}-%[3]s
  labels:
  {{- include "%[1]s.labels" . | nindent 4 }}
webhooks:
%[4]s`
)

var mwhGVK = schema.GroupVersionKind{
	Group:   "admissionregistration.k8s.io",
	Version: "v1",
	Kind:    "MutatingWebhookConfiguration",
}

// MutatingWebhook creates processor for k8s MutatingWebhookConfiguration resource.
func MutatingWebhook() helmify.Processor {
	return &mwh{}
}

type mwh struct{}

// Process k8s MutatingWebhookConfiguration object into template. Returns false if not capable of processing given resource type.
func (w mwh) Process(appMeta helmify.AppMetadata, obj *unstructured.Unstructured) (bool, helmify.Template, error) {
	if obj.GroupVersionKind() != mwhGVK {
		return false, nil, nil
	}
	name := appMeta.TrimName(obj.GetName())

	whConf := v1.MutatingWebhookConfiguration{}
	err := runtime.DefaultUnstructuredConverter.FromUnstructured(obj.Object, &whConf)
	if err != nil {
		return true, nil, fmt.Errorf("%w: unable to cast to MutatingWebhookConfiguration", err)
	}
	for i, whc := range whConf.Webhooks {
		whc.ClientConfig.Service.Name = appMeta.TemplatedName(whc.ClientConfig.Service.Name)
		whc.ClientConfig.Service.Namespace = strings.ReplaceAll(whc.ClientConfig.Service.Namespace, appMeta.Namespace(), `{{ .Release.Namespace }}`)
		mutateNamespaceSelector(appMeta, whc.NamespaceSelector)
		whConf.Webhooks[i] = whc
	}
	webhooks, _ := yaml.Marshal(whConf.Webhooks)
	webhooks = bytes.TrimRight(webhooks, "\n ")
	certName, _, err := unstructured.NestedString(obj.Object, "metadata", "annotations", "cert-manager.io/inject-ca-from")
	if err != nil {
		return true, nil, fmt.Errorf("%w: unable get webhook certName", err)
	}
	certName = strings.TrimPrefix(certName, appMeta.Namespace()+"/")
	certName = appMeta.TrimName(certName)
	tmpl := mwhTempl
	values := helmify.Values{}
	if appMeta.Config().AddWebhookOption {
		// Add webhook.enabled value to values.yaml
		_, _ = values.Add(true, "webhook", "enabled")

		tmpl = fmt.Sprintf("%s\n%s\n%s", WebhookHeader, mwhTempl, WebhookFooter)
	}
	res := fmt.Sprintf(tmpl, appMeta.ChartName(), name, certName, string(webhooks))
	return true, &mwhResult{
		name: name,
		data: []byte(res),
	}, nil
}

type mwhResult struct {
	name   string
	data   []byte
	values helmify.Values
}

func (r *mwhResult) Filename() string {
	return r.name + ".yaml"
}

func (r *mwhResult) Values() helmify.Values {
	return r.values
}

func (r *mwhResult) Write(writer io.Writer) error {
	_, err := writer.Write(r.data)
	return err
}

const (
	nameLabel         = "kubernetes.io/metadata.name"
	namespaceTemplate = "{{ .Release.Namespace }}"
)

// Replace the relase namespace in a namespace selector
func mutateNamespaceSelector(appMeta helmify.AppMetadata, sel *metav1.LabelSelector) {
	if appMeta.Config().PreserveNs || sel == nil {
		return
	}
	origNamespace := appMeta.Namespace()
	for i, me := range sel.MatchExpressions {
		if me.Key == nameLabel {
			for vi, v := range me.Values {
				if v == origNamespace {
					me.Values[vi] = namespaceTemplate
				}
			}
			sel.MatchExpressions[i] = me
		}
	}
}
