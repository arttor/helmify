package service

import (
	"bytes"
	"fmt"
	"github.com/arttor/helmify/pkg/helmify"
	yamlformat "github.com/arttor/helmify/pkg/yaml"
	"github.com/iancoleman/strcase"
	"github.com/pkg/errors"
	"io"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/intstr"
	"sigs.k8s.io/yaml"
	"strings"
)

const (
	svcTemplMeta = `apiVersion: v1
kind: Service
metadata:
  name: {{ include "%[1]s.fullname" . }}-%[2]s
  labels:
  {{- include "%[1]s.labels" . | nindent 4 }}
`
	svcTempSpec = `
spec:
  type: {{ .Values.%s.type }}
  selector:
%s
  {{- include "chart.selectorLabels" . | nindent 4 }}
  ports:
	{{- .Values.%s.ports | toYaml | nindent 2 -}}
`
)

var (
	svcGVC = schema.GroupVersionKind{
		Group:   "",
		Version: "v1",
		Kind:    "Service",
	}
)

func New() helmify.Processor {
	return &svc{}
}

type svc struct {
}

func (r svc) Process(info helmify.ChartInfo, obj *unstructured.Unstructured) (bool, helmify.Template, error) {
	if obj.GroupVersionKind() != svcGVC {
		return false, nil, nil
	}
	service := corev1.Service{}
	err := runtime.DefaultUnstructuredConverter.FromUnstructured(obj.Object, &service)
	if err != nil {
		return true, nil, errors.Wrap(err, "unable to cast to service")
	}

	name := strings.TrimPrefix(obj.GetName(), info.OperatorName+"-")
	shortName := strings.TrimPrefix(name, "controller-manager-")
	shortNameCamel := strcase.ToLowerCamel(shortName)
	res := fmt.Sprintf(svcTemplMeta, info.ChartName, name)
	if len(obj.GetLabels()) > 0 {
		labels, _ := yaml.Marshal(obj.GetLabels())
		labels = yamlformat.Indent(labels, 4)
		labels = bytes.TrimRight(labels, "\n ")
		res = res + string(labels)
	}

	selector, _ := yaml.Marshal(service.Spec.Selector)
	selector = yamlformat.Indent(selector, 4)
	selector = bytes.TrimRight(selector, "\n ")

	values := helmify.Values{}
	svcType := service.Spec.Type
	if svcType == "" {
		svcType = corev1.ServiceTypeClusterIP
	}
	_ = unstructured.SetNestedField(values, string(svcType), shortNameCamel, "type")
	var ports []interface{}
	for _, p := range service.Spec.Ports {
		pMap := map[string]interface{}{
			"port": int64(p.Port),
		}
		if p.Name != "" {
			pMap["name"] = p.Name
		}
		if p.NodePort != 0 {
			pMap["nodePort"] = int64(p.NodePort)
		}
		if p.Protocol != "" {
			pMap["protocol"] = string(p.Protocol)
		}
		if p.TargetPort.Type == intstr.Int {
			pMap["targetPort"] = int64(p.TargetPort.IntVal)
		} else {
			pMap["targetPort"] = p.TargetPort.StrVal
		}
		ports = append(ports, pMap)
	}
	_ = unstructured.SetNestedSlice(values, ports, shortNameCamel, "ports")
	res = res + fmt.Sprintf(svcTempSpec, shortNameCamel, selector, shortNameCamel)
	return true, &result{
		name:   shortName,
		data:   res,
		values: values,
	}, nil
}

type result struct {
	name      string
	data      string
	values    helmify.Values
	chartName string
}

func (r *result) Filename() string {
	return r.name + ".yaml"
}

func (r *result) Values() helmify.Values {
	return r.values
}

func (r *result) Write(writer io.Writer) error {
	_, err := writer.Write([]byte(r.data))
	return err
}

func (r *result) PostProcess(values helmify.Values) {
}
