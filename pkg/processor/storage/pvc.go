package storage

import (
	"github.com/arttor/helmify/pkg/helmify"
	"github.com/arttor/helmify/pkg/processor"
	yamlformat "github.com/arttor/helmify/pkg/yaml"
	"github.com/iancoleman/strcase"
	"github.com/pkg/errors"
	"io"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"strings"
	"text/template"
)

var pvcTempl, _ = template.New("pvc").Parse(
	`{{ .Meta }}
{{ .Spec }}`)

var pvcGVC = schema.GroupVersionKind{
	Group:   "",
	Version: "v1",
	Kind:    "PersistentVolumeClaim",
}

// New creates processor for k8s PVC resource.
func New() helmify.Processor {
	return &pvc{}
}

type pvc struct{}

// Process k8s PVC object into template. Returns false if not capable of processing given resource type.
func (p pvc) Process(appMeta helmify.AppMetadata, obj *unstructured.Unstructured) (bool, helmify.Template, error) {
	if obj.GroupVersionKind() != pvcGVC {
		return false, nil, nil
	}
	meta, err := processor.ProcessObjMeta(appMeta, obj)
	if err != nil {
		return true, nil, err
	}

	name := appMeta.TrimName(obj.GetName())
	nameCamelCase := strcase.ToLowerCamel(name)
	values := helmify.Values{}

	claim := corev1.PersistentVolumeClaim{}
	err = runtime.DefaultUnstructuredConverter.FromUnstructured(obj.Object, &claim)
	if err != nil {
		return true, nil, errors.Wrap(err, "unable to cast to PVC")
	}

	// template storage class name
	if claim.Spec.StorageClassName != nil {
		templatedSC, err := values.Add(*claim.Spec.StorageClassName, "pvc", nameCamelCase, "storageClass")
		if err != nil {
			return true, nil, err
		}
		claim.Spec.StorageClassName = &templatedSC
	}

	// template resources
	specMap, err := runtime.DefaultUnstructuredConverter.ToUnstructured(&claim.Spec)
	if err != nil {
		return true, nil, err
	}

	storageReq, ok, _ := unstructured.NestedString(specMap, "resources", "requests", "storage")
	if ok {
		templatedStorageReq, err := values.Add(storageReq, "pvc", nameCamelCase, "storageRequest")
		if err != nil {
			return true, nil, err
		}
		err = unstructured.SetNestedField(specMap, templatedStorageReq, "resources", "requests", "storage")
		if err != nil {
			return true, nil, err
		}
	}

	storageLim, ok, _ := unstructured.NestedString(specMap, "resources", "limits", "storage")
	if ok {
		templatedStorageLim, err := values.Add(storageLim, "pvc", nameCamelCase, "storageLimit")
		if err != nil {
			return true, nil, err
		}
		err = unstructured.SetNestedField(specMap, templatedStorageLim, "resources", "limits", "storage")
		if err != nil {
			return true, nil, err
		}
	}

	spec, err := yamlformat.Marshal(map[string]interface{}{"spec": specMap}, 0)
	if err != nil {
		return true, nil, err
	}
	spec = strings.ReplaceAll(spec, "'", "")

	return true, &result{
		name: name + ".yaml",
		data: struct {
			Meta string
			Spec string
		}{Meta: meta, Spec: spec},
		values: values,
	}, nil
}

type result struct {
	name string
	data struct {
		Meta string
		Spec string
	}
	values helmify.Values
}

func (r *result) Filename() string {
	return r.name
}

func (r *result) Values() helmify.Values {
	return r.values
}

func (r *result) Write(writer io.Writer) error {
	return pvcTempl.Execute(writer, r.data)
}
