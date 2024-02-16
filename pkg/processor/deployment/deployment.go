package deployment

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

var deploymentGVC = schema.GroupVersionKind{
	Group:   "apps",
	Version: "v1",
	Kind:    "Deployment",
}

var deploymentTempl, _ = template.New("deployment").Parse(
	`{{- .Meta }}
spec:
{{- if .Replicas }}
{{ .Replicas }}
{{- end }}
{{- if .RevisionHistoryLimit }}
{{ .RevisionHistoryLimit }}
{{- end }}
  selector:
{{ .Selector }}
  template:
    metadata:
      labels:
{{ .PodLabels }}
{{- .PodAnnotations }}
    spec:
{{ .Spec }}`)

const selectorTempl = `%[1]s
{{- include "%[2]s.selectorLabels" . | nindent 6 }}
%[3]s`

// New creates processor for k8s Deployment resource.
func New() helmify.Processor {
	return &deployment{}
}

type deployment struct{}

// Process k8s Deployment object into template. Returns false if not capable of processing given resource type.
func (d deployment) Process(appMeta helmify.AppMetadata, obj *unstructured.Unstructured) (bool, helmify.Template, error) {
	if obj.GroupVersionKind() != deploymentGVC {
		return false, nil, nil
	}
	depl := appsv1.Deployment{}

	err := runtime.DefaultUnstructuredConverter.FromUnstructured(obj.Object, &depl)
	if err != nil {
		return true, nil, fmt.Errorf("%w: unable to cast to deployment", err)
	}
	meta, err := processor.ProcessObjMeta(appMeta, obj)
	if err != nil {
		return true, nil, err
	}

	values := helmify.Values{}

	name := appMeta.TrimName(obj.GetName())
	replicas, err := processReplicas(name, &depl, &values)
	if err != nil {
		return true, nil, err
	}

	revisionHistoryLimit, err := processRevisionHistoryLimit(name, &depl, &values)
	if err != nil {
		return true, nil, err
	}

	matchLabels, err := yamlformat.Marshal(map[string]interface{}{"matchLabels": depl.Spec.Selector.MatchLabels}, 0)
	if err != nil {
		return true, nil, err
	}
	matchExpr := ""
	if depl.Spec.Selector.MatchExpressions != nil {
		matchExpr, err = yamlformat.Marshal(map[string]interface{}{"matchExpressions": depl.Spec.Selector.MatchExpressions}, 0)
		if err != nil {
			return true, nil, err
		}
	}
	selector := fmt.Sprintf(selectorTempl, matchLabels, appMeta.ChartName(), matchExpr)
	selector = strings.Trim(selector, " \n")
	selector = string(yamlformat.Indent([]byte(selector), 4))

	podLabels, err := yamlformat.Marshal(depl.Spec.Template.ObjectMeta.Labels, 8)
	if err != nil {
		return true, nil, err
	}
	podLabels += fmt.Sprintf("\n      {{- include \"%s.selectorLabels\" . | nindent 8 }}", appMeta.ChartName())

	podAnnotations := ""
	if len(depl.Spec.Template.ObjectMeta.Annotations) != 0 {
		podAnnotations, err = yamlformat.Marshal(map[string]interface{}{"annotations": depl.Spec.Template.ObjectMeta.Annotations}, 6)
		if err != nil {
			return true, nil, err
		}

		podAnnotations = "\n" + podAnnotations
	}

	nameCamel := strcase.ToLowerCamel(name)
	specMap, podValues, err := pod.ProcessSpec(nameCamel, appMeta, depl.Spec.Template.Spec, depl.TypeMeta.Kind)
	if err != nil {
		return true, nil, err
	}
	err = values.Merge(podValues)
	if err != nil {
		return true, nil, err
	}

	spec, err := yamlformat.Marshal(specMap, 6)
	if err != nil {
		return true, nil, err
	}

	spec = strings.ReplaceAll(spec, "'", "")

	return true, &result{
		values: values,
		data: struct {
			Meta                 string
			Replicas             string
			RevisionHistoryLimit string
			Selector             string
			PodLabels            string
			PodAnnotations       string
			Spec                 string
		}{
			Meta:                 meta,
			Replicas:             replicas,
			RevisionHistoryLimit: revisionHistoryLimit,
			Selector:             selector,
			PodLabels:            podLabels,
			PodAnnotations:       podAnnotations,
			Spec:                 spec,
		},
	}, nil
}

func processReplicas(name string, deployment *appsv1.Deployment, values *helmify.Values) (string, error) {
	if deployment.Spec.Replicas == nil {
		return "", nil
	}
	replicasTpl, err := values.Add(int64(*deployment.Spec.Replicas), name, "replicas")
	if err != nil {
		return "", err
	}
	replicas, err := yamlformat.Marshal(map[string]interface{}{"replicas": replicasTpl}, 2)
	if err != nil {
		return "", err
	}
	replicas = strings.ReplaceAll(replicas, "'", "")
	return replicas, nil
}

func processRevisionHistoryLimit(name string, deployment *appsv1.Deployment, values *helmify.Values) (string, error) {
	if deployment.Spec.RevisionHistoryLimit == nil {
		return "", nil
	}
	revisionHistoryLimitTpl, err := values.Add(int64(*deployment.Spec.RevisionHistoryLimit), name, "revisionHistoryLimit")
	if err != nil {
		return "", err
	}
	revisionHistoryLimit, err := yamlformat.Marshal(map[string]interface{}{"revisionHistoryLimit": revisionHistoryLimitTpl}, 2)
	if err != nil {
		return "", err
	}
	revisionHistoryLimit = strings.ReplaceAll(revisionHistoryLimit, "'", "")
	return revisionHistoryLimit, nil
}

type result struct {
	data struct {
		Meta                 string
		Replicas             string
		RevisionHistoryLimit string
		Selector             string
		PodLabels            string
		PodAnnotations       string
		Spec                 string
	}
	values helmify.Values
}

func (r *result) Filename() string {
	return "deployment.yaml"
}

func (r *result) Values() helmify.Values {
	return r.values
}

func (r *result) Write(writer io.Writer) error {
	return deploymentTempl.Execute(writer, r.data)
}
