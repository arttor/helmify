package statefulset

import (
	"fmt"
	"io"
	"strings"
	"text/template"

	"github.com/arttor/helmify/pkg/processor/pod"

	"github.com/arttor/helmify/pkg/helmify"
	"github.com/arttor/helmify/pkg/processor"
	yamlformat "github.com/arttor/helmify/pkg/yaml"
	"github.com/iancoleman/strcase"
	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

var statefulsetGVC = schema.GroupVersionKind{
	Group:   "apps",
	Version: "v1",
	Kind:    "StatefulSet",
}

var statefulsetTempl, _ = template.New("statefulset").Parse(
	`{{- .Meta }}
spec:
{{ .Spec }}`)

// New creates processor for k8s StatefulSet resource.
func New() helmify.Processor {
	return &statefulset{}
}

type statefulset struct{}

// Process k8s StatefulSet object into template. Returns false if not capable of processing given resource type.
func (d statefulset) Process(appMeta helmify.AppMetadata, obj *unstructured.Unstructured) (bool, helmify.Template, error) {
	if obj.GroupVersionKind() != statefulsetGVC {
		return false, nil, nil
	}
	ss := appsv1.StatefulSet{}
	err := runtime.DefaultUnstructuredConverter.FromUnstructured(obj.Object, &ss)
	if err != nil {
		return true, nil, fmt.Errorf("%w: unable to cast to StatefulSet", err)
	}
	meta, err := processor.ProcessObjMeta(appMeta, obj)
	if err != nil {
		return true, nil, err
	}

	ssSpec := ss.Spec
	ssSpecMap, err := runtime.DefaultUnstructuredConverter.ToUnstructured(&ssSpec)
	if err != nil {
		return true, nil, err
	}
	delete((ssSpecMap["template"].(map[string]interface{}))["metadata"].(map[string]interface{}), "creationTimestamp")

	values := helmify.Values{}

	name := appMeta.TrimName(obj.GetName())
	nameCamel := strcase.ToLowerCamel(name)

	if ssSpec.ServiceName != "" {
		servName := appMeta.TemplatedName(ssSpec.ServiceName)
		ssSpecMap["serviceName"] = servName
	}

	if ssSpec.Replicas != nil {
		repl, err := values.Add(*ssSpec.Replicas, nameCamel, "replicas")
		if err != nil {
			return true, nil, err
		}
		ssSpecMap["replicas"] = repl
	}

	for i, claim := range ssSpec.VolumeClaimTemplates {
		volName := claim.ObjectMeta.Name
		delete(((ssSpecMap["volumeClaimTemplates"].([]interface{}))[i]).(map[string]interface{}), "status")
		if claim.Spec.StorageClassName != nil {
			scName := appMeta.TemplatedName(*claim.Spec.StorageClassName)
			err = unstructured.SetNestedField(((ssSpecMap["volumeClaimTemplates"].([]interface{}))[i]).(map[string]interface{}), scName, "spec", "storageClassName")
			if err != nil {
				return true, nil, err
			}
		}
		if claim.Spec.VolumeName != "" {
			vName := appMeta.TemplatedName(claim.Spec.VolumeName)
			err = unstructured.SetNestedField(((ssSpecMap["volumeClaimTemplates"].([]interface{}))[i]).(map[string]interface{}), vName, "spec", "volumeName")
			if err != nil {
				return true, nil, err
			}
		}

		resMap, err := runtime.DefaultUnstructuredConverter.ToUnstructured(&claim.Spec.Resources)
		if err != nil {
			return true, nil, err
		}
		resName, err := values.AddYaml(resMap, 8, true, nameCamel, "volumeClaims", volName)
		if err != nil {
			return true, nil, err
		}
		err = unstructured.SetNestedField(((ssSpecMap["volumeClaimTemplates"].([]interface{}))[i]).(map[string]interface{}), resName, "spec", "resources")
		if err != nil {
			return true, nil, err
		}
	}

	// process pod spec:
	podSpecMap, podValues, err := pod.ProcessSpec(nameCamel, appMeta, ssSpec.Template.Spec, ss.TypeMeta.Kind)
	if err != nil {
		return true, nil, err
	}
	err = values.Merge(podValues)
	if err != nil {
		return true, nil, err
	}
	err = unstructured.SetNestedMap(ssSpecMap, podSpecMap, "template", "spec")
	if err != nil {
		return true, nil, err
	}

	spec, err := yamlformat.Marshal(ssSpecMap, 2)
	if err != nil {
		return true, nil, err
	}
	spec = strings.ReplaceAll(spec, "'", "")

	return true, &result{
		values: values,
		data: struct {
			Meta string
			Spec string
		}{
			Meta: meta,
			Spec: spec,
		},
	}, nil
}

type result struct {
	data struct {
		Meta string
		Spec string
	}
	values helmify.Values
}

func (r *result) Filename() string {
	return "statefulset.yaml"
}

func (r *result) Values() helmify.Values {
	return r.values
}

func (r *result) Write(writer io.Writer) error {
	return statefulsetTempl.Execute(writer, r.data)
}
