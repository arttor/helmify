package webhook

import (
	"bytes"
	"fmt"
	"github.com/arttor/helmify/pkg/helmify"
	"github.com/pkg/errors"
	"io"
	v1 "k8s.io/api/admissionregistration/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/yaml"
	"strings"
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

var (
	whGVK = schema.GroupVersionKind{
		Group:   "admissionregistration.k8s.io",
		Version: "v1",
		Kind:    "ValidatingWebhookConfiguration",
	}
)

func Webhook() helmify.Processor {
	return &wh{}
}

type wh struct {
}

func (w wh) Process(info helmify.ChartInfo, obj *unstructured.Unstructured) (bool, helmify.Template, error) {
	if obj.GroupVersionKind() != whGVK {
		return false, nil, nil
	}
	name := strings.TrimPrefix(obj.GetName(), info.OperatorName+"-")

	whConf := v1.ValidatingWebhookConfiguration{}
	err := runtime.DefaultUnstructuredConverter.FromUnstructured(obj.Object, &whConf)
	if err != nil {
		return true, nil, errors.Wrap(err, "unable to cast to ValidatingWebhookConfiguration")
	}
	for i, whc := range whConf.Webhooks {
		whc.ClientConfig.Service.Name = strings.ReplaceAll(whc.ClientConfig.Service.Name, info.OperatorName, fmt.Sprintf(`{{ include "%s.fullname" . }}`, info.ChartName))
		whc.ClientConfig.Service.Namespace = strings.ReplaceAll(whc.ClientConfig.Service.Namespace, info.OperatorNamespace, `{{ .Release.Namespace }}`)
		whConf.Webhooks[i] = whc
	}
	webhooks, _ := yaml.Marshal(whConf.Webhooks)
	webhooks = bytes.TrimRight(webhooks, "\n ")
	certName, _, err := unstructured.NestedString(obj.Object, "metadata", "annotations", "cert-manager.io/inject-ca-from")
	if err != nil {
		return true, nil, errors.Wrap(err, "unable get webhook certName")
	}
	certName = strings.TrimPrefix(certName, info.OperatorNamespace+"/"+info.OperatorName+"-")
	res := fmt.Sprintf(whTempl, info.ChartName, name, certName, string(webhooks))
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
