package webhook

import (
	"bytes"
	"fmt"
	"io"
	"strings"

	"github.com/arttor/helmify/pkg/helmify"
	"github.com/iancoleman/strcase"
	"github.com/pkg/errors"
	v1 "k8s.io/api/admissionregistration/v1"
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
	nameCamel := strcase.ToLowerCamel(name)

	whConf := v1.MutatingWebhookConfiguration{}
	err := runtime.DefaultUnstructuredConverter.FromUnstructured(obj.Object, &whConf)
	if err != nil {
		return true, nil, errors.Wrap(err, "unable to cast to MutatingWebhookConfiguration")
	}

	values := helmify.Values{}
	err = unstructured.SetNestedMap(values, make(map[string]interface{}), nameCamel)
	if err != nil {
		return true, nil, errors.Wrap(err, fmt.Sprintf("can not set webhook parameter map for %s", name))
	}

	for i, whc := range whConf.Webhooks {
		whcField := strcase.ToLowerCamel(whc.Name)
		whcValues, err := ProcessMWH(appMeta, &whc, fmt.Sprintf(".Values.%s.%s", nameCamel, whcField))
		if err != nil {
			return true, nil, errors.Wrap(err, fmt.Sprintf("unable to Process WebHook config: %s / %s", name, whc.Name))
		}
		err = unstructured.SetNestedField(values, whcValues, nameCamel, whcField)
		if err != nil {
			return true, nil, errors.Wrap(err, fmt.Sprintf("can not set webhook parameters for %s / %s", name, whc.Name))
		}
		whConf.Webhooks[i] = whc
	}
	webhooks, _ := yaml.Marshal(whConf.Webhooks)
	webhooks = bytes.TrimRight(webhooks, "\n ")
	certName, _, err := unstructured.NestedString(obj.Object, "metadata", "annotations", "cert-manager.io/inject-ca-from")
	if err != nil {
		return true, nil, errors.Wrap(err, "unable get webhook certName")
	}
	certName = strings.TrimPrefix(certName, appMeta.Namespace()+"/")
	certName = appMeta.TrimName(certName)
	res := fmt.Sprintf(mwhTempl, appMeta.ChartName(), name, certName, string(webhooks))
	return true, &mwhResult{
		values: values,
		name:   name,
		data:   []byte(res),
	}, nil
}

func ProcessMWH(appMeta helmify.AppMetadata, whc *v1.MutatingWebhook, valuesRoot string) (map[string]interface{}, error) {
	values := make(map[string]interface{})
	failurePolicyField := "failurePolicy"
	failurePolicy := v1.FailurePolicyType(fmt.Sprintf("{{ %s.%s }}", valuesRoot, failurePolicyField))
	values[failurePolicyField] = "Fail"

	whc.ClientConfig.Service.Name = appMeta.TemplatedName(whc.ClientConfig.Service.Name)
	whc.ClientConfig.Service.Namespace = strings.ReplaceAll(whc.ClientConfig.Service.Namespace, appMeta.Namespace(), `{{ .Release.Namespace }}`)
	whc.FailurePolicy = &failurePolicy
	return values, nil
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
