package job

import (
	"fmt"
	"io"
	"strings"
	"text/template"

	"github.com/arttor/helmify/pkg/helmify"
	"github.com/arttor/helmify/pkg/processor"
	"github.com/arttor/helmify/pkg/processor/pod"
	yamlformat "github.com/arttor/helmify/pkg/yaml"
	"github.com/iancoleman/strcase"
	batchv1 "k8s.io/api/batch/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

var jobTempl, _ = template.New("job").Parse(
	`{{ .Meta }}
{{ .Spec }}`)

var jobGVC = schema.GroupVersionKind{
	Group:   "batch",
	Version: "v1",
	Kind:    "Job",
}

// NewJob creates processor for k8s Job resource.
func NewJob() helmify.Processor {
	return &job{}
}

type job struct{}

// Process k8s Job object into template. Returns false if not capable of processing given resource type.
func (p job) Process(appMeta helmify.AppMetadata, obj *unstructured.Unstructured) (bool, helmify.Template, error) {
	if obj.GroupVersionKind() != jobGVC {
		return false, nil, nil
	}
	meta, err := processor.ProcessObjMeta(appMeta, obj)
	if err != nil {
		return true, nil, err
	}
	name := appMeta.TrimName(obj.GetName())
	nameCamelCase := strcase.ToLowerCamel(name)

	jobObj := batchv1.Job{}
	err = runtime.DefaultUnstructuredConverter.FromUnstructured(obj.Object, &jobObj)
	if err != nil {
		return true, nil, fmt.Errorf("%w: unable to cast to Job", err)
	}
	spec := jobObj.Spec
	specMap, exists, err := unstructured.NestedMap(obj.Object, "spec")
	if err != nil {
		return true, nil, fmt.Errorf("%w: unable to get job spec", err)
	}
	if !exists {
		return true, nil, fmt.Errorf("no job spec presented")
	}

	values := helmify.Values{}

	// process job spec params:
	if spec.BackoffLimit != nil {
		err := templateSpecVal(*spec.BackoffLimit, &values, specMap, nameCamelCase, "backoffLimit")
		if err != nil {
			return true, nil, err
		}
	}

	if spec.ActiveDeadlineSeconds != nil {
		err := templateSpecVal(*spec.ActiveDeadlineSeconds, &values, specMap, nameCamelCase, "activeDeadlineSeconds")
		if err != nil {
			return true, nil, err
		}
	}

	if spec.Completions != nil {
		err := templateSpecVal(*spec.Completions, &values, specMap, nameCamelCase, "completions")
		if err != nil {
			return true, nil, err
		}
	}

	if spec.Parallelism != nil {
		err := templateSpecVal(*spec.Parallelism, &values, specMap, nameCamelCase, "parallelism")
		if err != nil {
			return true, nil, err
		}
	}

	if spec.Suspend != nil {
		err := templateSpecVal(*spec.Suspend, &values, specMap, nameCamelCase, "suspend")
		if err != nil {
			return true, nil, err
		}
	}

	if spec.ActiveDeadlineSeconds != nil {
		err := templateSpecVal(*spec.ActiveDeadlineSeconds, &values, specMap, nameCamelCase, "activeDeadlineSeconds")
		if err != nil {
			return true, nil, err
		}
	}
	// process job pod template:
	podSpecMap, podValues, err := pod.ProcessSpec(nameCamelCase, appMeta, jobObj.Spec.Template.Spec, jobObj.TypeMeta.Kind)
	if err != nil {
		return true, nil, err
	}
	err = values.Merge(podValues)
	if err != nil {
		return true, nil, err
	}

	err = unstructured.SetNestedMap(specMap, podSpecMap, "template", "spec")
	if err != nil {
		return true, nil, fmt.Errorf("%w: unable to template job spec", err)
	}

	specStr, err := yamlformat.Marshal(map[string]interface{}{"spec": specMap}, 0)
	if err != nil {
		return true, nil, err
	}
	specStr = strings.ReplaceAll(specStr, "'", "")

	return true, &result{
		name: name + ".yaml",
		data: struct {
			Meta string
			Spec string
		}{Meta: meta, Spec: specStr},
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
	return jobTempl.Execute(writer, r.data)
}

func templateSpecVal(val any, values *helmify.Values, specMap map[string]interface{}, objName string, fieldName ...string) error {
	valName := []string{objName}
	valName = append(valName, fieldName...)
	templatedVal, err := values.Add(val, valName...)
	if err != nil {
		return fmt.Errorf("%w: unable to set %q to values", err, strings.Join(valName, "."))
	}

	err = unstructured.SetNestedField(specMap, templatedVal, fieldName...)
	if err != nil {
		return fmt.Errorf("%w: unable to template job %q", err, strings.Join(valName, "."))
	}
	return nil
}
