package daemonset

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
	"k8s.io/apimachinery/pkg/util/intstr"
)

var daemonsetGVC = schema.GroupVersionKind{
	Group:   "apps",
	Version: "v1",
	Kind:    "DaemonSet",
}

var daemonsetTempl, _ = template.New("daemonset").Parse(
	`{{- .Meta }}
spec:
{{- if .UpdateStrategy }}
{{ .UpdateStrategy }}
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

// New creates processor for k8s Daemonset resource.
func New() helmify.Processor {
	return &daemonset{}
}

type daemonset struct{}

// Process k8s Daemonset object into template. Returns false if not capable of processing given resource type.
func (d daemonset) Process(appMeta helmify.AppMetadata, obj *unstructured.Unstructured) (bool, helmify.Template, error) {
	if obj.GroupVersionKind() != daemonsetGVC {
		return false, nil, nil
	}
	dae := appsv1.DaemonSet{}
	err := runtime.DefaultUnstructuredConverter.FromUnstructured(obj.Object, &dae)
	if err != nil {
		return true, nil, fmt.Errorf("%w: unable to cast to daemonset", err)
	}
	meta, err := processor.ProcessObjMeta(appMeta, obj)
	if err != nil {
		return true, nil, err
	}

	values := helmify.Values{}

	name := appMeta.TrimName(obj.GetName())
	updateStrategy, err := processUpdateStrategy(name, &dae, &values)
	if err != nil {
		return true, nil, err
	}

	matchLabels, err := yamlformat.Marshal(map[string]interface{}{"matchLabels": dae.Spec.Selector.MatchLabels}, 0)
	if err != nil {
		return true, nil, err
	}
	matchExpr := ""
	if dae.Spec.Selector.MatchExpressions != nil {
		matchExpr, err = yamlformat.Marshal(map[string]interface{}{"matchExpressions": dae.Spec.Selector.MatchExpressions}, 0)
		if err != nil {
			return true, nil, err
		}
	}
	selector := fmt.Sprintf(selectorTempl, matchLabels, appMeta.ChartName(), matchExpr)
	selector = strings.Trim(selector, " \n")
	selector = string(yamlformat.Indent([]byte(selector), 4))

	podLabels, err := yamlformat.Marshal(dae.Spec.Template.ObjectMeta.Labels, 8)
	if err != nil {
		return true, nil, err
	}
	podLabels += fmt.Sprintf("\n      {{- include \"%s.selectorLabels\" . | nindent 8 }}", appMeta.ChartName())

	podAnnotations := ""
	if len(dae.Spec.Template.ObjectMeta.Annotations) != 0 {
		podAnnotations, err = yamlformat.Marshal(map[string]interface{}{"annotations": dae.Spec.Template.ObjectMeta.Annotations}, 6)
		if err != nil {
			return true, nil, err
		}

		podAnnotations = "\n" + podAnnotations
	}

	nameCamel := strcase.ToLowerCamel(name)
	specMap, podValues, err := pod.ProcessSpec(nameCamel, appMeta, dae.Spec.Template.Spec, 0)
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
			Meta           string
			UpdateStrategy string
			Selector       string
			PodLabels      string
			PodAnnotations string
			Spec           string
		}{
			Meta:           meta,
			UpdateStrategy: updateStrategy,
			Selector:       selector,
			PodLabels:      podLabels,
			PodAnnotations: podAnnotations,
			Spec:           spec,
		},
	}, nil
}

func processUpdateStrategy(name string, dae *appsv1.DaemonSet, values *helmify.Values) (string, error) {
	if dae.Spec.UpdateStrategy.Type == "" && dae.Spec.UpdateStrategy.RollingUpdate == nil {
		return "", nil
	}
	if dae.Spec.UpdateStrategy.Type == "" {
		dae.Spec.UpdateStrategy.Type = appsv1.RollingUpdateDaemonSetStrategyType
	}
	allowedStrategyTypes := map[appsv1.DaemonSetUpdateStrategyType]bool{
		appsv1.RollingUpdateDaemonSetStrategyType: true,
		appsv1.OnDeleteDaemonSetStrategyType:      true,
	}
	if !allowedStrategyTypes[dae.Spec.UpdateStrategy.Type] {
		return "", fmt.Errorf("invalid daemonset update strategy type: %s", dae.Spec.UpdateStrategy.Type)
	}
	strategyTypeTpl, err := values.Add(string(dae.Spec.UpdateStrategy.Type), name, "updateStrategy", "type")
	if err != nil {
		return "", err
	}
	strategyMap := map[string]interface{}{
		"type": strategyTypeTpl,
	}
	if dae.Spec.UpdateStrategy.Type == appsv1.RollingUpdateDaemonSetStrategyType {
		if rollingUpdate := dae.Spec.UpdateStrategy.RollingUpdate; rollingUpdate != nil {
			rollingUpdateMap := map[string]interface{}{}
			setRollingUpdateField := func(value *intstr.IntOrString, fieldName string) error {
				var tpl string
				var err error
				if value.Type == intstr.Int {
					tpl, err = values.Add(value.IntValue(), name, "updateStrategy", "rollingUpdate", fieldName)
				} else {
					tpl, err = values.Add(value.String(), name, "updateStrategy", "rollingUpdate", fieldName)
				}
				if err != nil {
					return err
				}
				rollingUpdateMap[fieldName] = tpl
				return nil
			}
			if rollingUpdate.MaxSurge != nil {
				if err := setRollingUpdateField(rollingUpdate.MaxSurge, "maxSurge"); err != nil {
					return "", err
				}
			}
			if rollingUpdate.MaxUnavailable != nil {
				if err := setRollingUpdateField(rollingUpdate.MaxUnavailable, "maxUnavailable"); err != nil {
					return "", err
				}
			}
			strategyMap["rollingUpdate"] = rollingUpdateMap
		}
	}
	strategy, err := yamlformat.Marshal(map[string]interface{}{"updateStrategy": strategyMap}, 2)
	if err != nil {
		return "", err
	}
	strategy = strings.ReplaceAll(strategy, "'", "")
	return strategy, nil
}

type result struct {
	data struct {
		Meta           string
		UpdateStrategy string
		Selector       string
		PodLabels      string
		PodAnnotations string
		Spec           string
	}
	values helmify.Values
}

func (r *result) Filename() string {
	return "daemonset.yaml"
}

func (r *result) Values() helmify.Values {
	return r.values
}

func (r *result) Write(writer io.Writer) error {
	return daemonsetTempl.Execute(writer, r.data)
}
