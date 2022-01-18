package webhook

import (
	"bytes"
	"fmt"
	"io"
	"strings"

	"github.com/arttor/helmify/pkg/helmify"
	"github.com/pkg/errors"
	v1 "k8s.io/api/admissionregistration/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/yaml"
)

const (
	whTempl = `apiVersion: admissionregistration.k8s.io/v1
kind: ValidatingWebhookConfiguration
metadata:
  name: {{ include "%[1]s.fullname" . }}-%[2]s
  annotations:
    cert-manager.io/inject-ca-from: {{ .Release.Namespace }}/{{ include "%[1]s.fullname" . }}-%[3]s
  labels:
  {{- include "%[1]s.labels" . | nindent 4 }}
webhooks:
%[4]s`
)

var whGVK = schema.GroupVersionKind{
	Group:   "admissionregistration.k8s.io",
	Version: "v1",
	Kind:    "ValidatingWebhookConfiguration",
}

// Webhook creates processor for k8s ValidatingWebhookConfiguration resource.
func Webhook() helmify.Processor {
	return &wh{}
}

type wh struct{}

// Process k8s ValidatingWebhookConfiguration object into template. Returns false if not capable of processing given resource type.
func (w wh) Process(appMeta helmify.AppMetadata, obj *unstructured.Unstructured) (bool, helmify.Template, error) {
	if obj.GroupVersionKind() != whGVK {
		return false, nil, nil
	}
	name := appMeta.TrimName(obj.GetName())

	whConf := v1.ValidatingWebhookConfiguration{}
	err := runtime.DefaultUnstructuredConverter.FromUnstructured(obj.Object, &whConf)
	if err != nil {
		return true, nil, errors.Wrap(err, "unable to cast to ValidatingWebhookConfiguration")
	}
	for i, whc := range whConf.Webhooks {
		whc.ClientConfig.Service.Name = appMeta.TemplatedName(whc.ClientConfig.Service.Name)
		whc.ClientConfig.Service.Namespace = strings.ReplaceAll(whc.ClientConfig.Service.Namespace, appMeta.Namespace(), `{{ .Release.Namespace }}`)
		whConf.Webhooks[i] = whc
	}
	webhooks, _ := yaml.Marshal(whConf.Webhooks)
	webhooks = bytes.TrimRight(webhooks, "\n ")
	certName, _, err := unstructured.NestedString(obj.Object, "metadata", "annotations", "cert-manager.io/inject-ca-from")
	if err != nil {
		return true, nil, errors.Wrap(err, "unable get webhook certName")
	}
	certName = strings.TrimPrefix(certName, appMeta.Namespace()+"/"+appMeta.Namespace()+"-")
	res := fmt.Sprintf(whTempl, appMeta.ChartName(), name, certName, string(webhooks))
	return true, &whResult{
		name: name,
		data: []byte(res),
	}, nil
}

type whResult struct {
	name string
	data []byte
}

func (r *whResult) Filename() string {
	return r.name + ".yaml"
}

func (r *whResult) Values() helmify.Values {
	return helmify.Values{}
}

func (r *whResult) Write(writer io.Writer) error {
	_, err := writer.Write(r.data)
	return err
}
