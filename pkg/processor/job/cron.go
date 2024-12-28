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

var cronTempl, _ = template.New("cron").Parse(
	`{{ .Meta }}
{{ .Spec }}`)

var cronGVC = schema.GroupVersionKind{
	Group:   "batch",
	Version: "v1",
	Kind:    "CronJob",
}

// NewCron creates processor for k8s CronJob resource.
func NewCron() helmify.Processor {
	return &cron{}
}

type cron struct{}

// Process k8s CronJob object into template. Returns false if not capable of processing given resource type.
func (p cron) Process(appMeta helmify.AppMetadata, obj *unstructured.Unstructured) (bool, helmify.Template, error) {
	if obj.GroupVersionKind() != cronGVC {
		return false, nil, nil
	}
	meta, err := processor.ProcessObjMeta(appMeta, obj)
	if err != nil {
		return true, nil, err
	}
	name := appMeta.TrimName(obj.GetName())
	nameCamelCase := strcase.ToLowerCamel(name)

	jobObj := batchv1.CronJob{}
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
	if spec.Schedule != "" {
		err := templateSpecVal(spec.Schedule, &values, specMap, nameCamelCase, "schedule")
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

	if spec.FailedJobsHistoryLimit != nil {
		err := templateSpecVal(*spec.FailedJobsHistoryLimit, &values, specMap, nameCamelCase, "failedJobsHistoryLimit")
		if err != nil {
			return true, nil, err
		}
	}

	if spec.StartingDeadlineSeconds != nil {
		err := templateSpecVal(*spec.StartingDeadlineSeconds, &values, specMap, nameCamelCase, "startingDeadlineSeconds")
		if err != nil {
			return true, nil, err
		}
	}

	if spec.TimeZone != nil {
		err := templateSpecVal(*spec.TimeZone, &values, specMap, nameCamelCase, "timeZone")
		if err != nil {
			return true, nil, err
		}
	}

	if spec.SuccessfulJobsHistoryLimit != nil {
		err := templateSpecVal(*spec.SuccessfulJobsHistoryLimit, &values, specMap, nameCamelCase, "successfulJobsHistoryLimit")
		if err != nil {
			return true, nil, err
		}
	}

	// process job pod template:
	podSpecMap, podValues, err := pod.ProcessSpec(nameCamelCase, appMeta, jobObj.Spec.JobTemplate.Spec.Template.Spec, jobObj.TypeMeta.Kind)
	if err != nil {
		return true, nil, err
	}
	err = values.Merge(podValues)
	if err != nil {
		return true, nil, err
	}

	err = unstructured.SetNestedMap(specMap, podSpecMap, "jobTemplate", "spec", "template", "spec")
	if err != nil {
		return true, nil, fmt.Errorf("%w: unable to template job spec", err)
	}

	specStr, err := yamlformat.Marshal(map[string]interface{}{"spec": specMap}, 0)
	if err != nil {
		return true, nil, err
	}
	specStr = strings.ReplaceAll(specStr, "'", "")

	return true, &resultCron{
		name: name + ".yaml",
		data: struct {
			Meta string
			Spec string
		}{Meta: meta, Spec: specStr},
		values: values,
	}, nil
}

type resultCron struct {
	name string
	data struct {
		Meta string
		Spec string
	}
	values helmify.Values
}

func (r *resultCron) Filename() string {
	return r.name
}

func (r *resultCron) Values() helmify.Values {
	return r.values
}

func (r *resultCron) Write(writer io.Writer) error {
	return cronTempl.Execute(writer, r.data)
}
