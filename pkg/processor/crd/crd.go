package crd

import (
	"bytes"
	_ "embed"
	"github.com/arttor/helmify/pkg/context"
	yamlformat "github.com/arttor/helmify/pkg/yaml"
	"github.com/pkg/errors"
	"io"
	extensionv1beta1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/yaml"
	"text/template"
)

var (
	crdGVC = schema.GroupVersionKind{
		Group:   "apiextensions.k8s.io",
		Version: "v1",
		Kind:    "CustomResourceDefinition",
	}
	//go:embed crd.templ
	crdYaml string
)

func New() context.Processor {
	return &deployment{}
}

type deployment struct {
}

func (d deployment) Process(obj *unstructured.Unstructured) (bool, context.Template, error) {
	if obj.GroupVersionKind() != crdGVC {
		return false, nil,  nil
	}

	crd := extensionv1beta1.CustomResourceDefinition{}
	err := runtime.DefaultUnstructuredConverter.FromUnstructured(obj.Object, &crd)
	if err != nil {
		return true, nil,errors.Wrap(err, "unable to cast unstructured to crd")
	}

	tmpl, err := template.New("crd").Parse(crdYaml)
	if err != nil {
		return true, nil, errors.Wrap(err, "unable to create crd template")
	}
	versionsUnstr,ok,err:=unstructured.NestedSlice(obj.Object,"spec","versions")
	if err!=nil||!ok{
		return true, nil, errors.Wrap(err, "unable to create crd template")
	}
	var buf bytes.Buffer
	versions, _ := yaml.Marshal(versionsUnstr)
	versions = yamlformat.Indent(versions, 2)
	versions = bytes.TrimRight(versions, "\n ")

	err = tmpl.Execute(&buf, struct{ Versions string }{Versions: string(versions)})
	if err != nil {
		return true, nil, errors.Wrap(err, "unable to execute unstructured template")
	}
	res := bytes.ReplaceAll(buf.Bytes(), []byte("<CRD_NAME>"), []byte(crd.Spec.Names.Singular))
	values := context.Values{}
	err = unstructured.SetNestedField(values, crd.Spec.Group, "crd", crd.Spec.Names.Singular, "group")
	if err != nil {
		return true, nil, errors.Wrap(err, "unable to set crd value field")
	}
	err = unstructured.SetNestedField(values, crd.Spec.Names.Kind, "crd", crd.Spec.Names.Singular, "kind")
	if err != nil {
		return true, nil,  errors.Wrap(err, "unable to set crd value field")
	}
	err = unstructured.SetNestedField(values, crd.Spec.Names.ListKind, "crd", crd.Spec.Names.Singular, "listKind")
	if err != nil {
		return true, nil,  errors.Wrap(err, "unable to set crd value field")
	}
	err = unstructured.SetNestedField(values, crd.Spec.Names.Singular, "crd", crd.Spec.Names.Singular, "singular")
	if err != nil {
		return true, nil,  errors.Wrap(err, "unable to set crd value field")
	}
	err = unstructured.SetNestedField(values, crd.Spec.Names.Plural, "crd", crd.Spec.Names.Singular, "plural")
	if err != nil {
		return true, nil,  errors.Wrap(err, "unable to set crd value field")
	}
	err = unstructured.SetNestedField(values, string(crd.Spec.Scope), "crd", crd.Spec.Names.Singular, "scope")
	if err != nil {
		return true, nil,  errors.Wrap(err, "unable to set crd value field")
	}
	return true, &result{
		name: crd.Spec.Names.Singular+"-crd.yaml",
		values:  values,
		data: res,
	}, nil
}

type result struct {
	name      string
	data      []byte
	values context.Values
	chartName string
}

func (r *result) Filename() string {
	return r.name
}

func (r *result) GVK() schema.GroupVersionKind {
	return crdGVC
}

func (r *result) Values() context.Values {
	return r.values
}

func (r *result) Write(writer io.Writer) error {
	_, err := writer.Write(bytes.ReplaceAll(r.data, []byte("<CHART_NAME>"), []byte(r.chartName)))
	return err
}

func (r *result) PostProcess(data context.Data) {
}

func (r *result) SetChartName(name string) {
	r.chartName = name
}