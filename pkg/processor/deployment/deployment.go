package deployment

import (
	"fmt"
	"io"
	"regexp"
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
{{- if .Strategy }}
{{ .Strategy }}
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

	strategy, err := processStrategy(name, &depl, &values)
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
	specMap, podValues, err := pod.ProcessSpec(nameCamel, appMeta, depl.Spec.Template.Spec, 0)
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
	if appMeta.Config().AddWebhookOption {
		spec = addWebhookOption(spec)
	}

	spec = replaceSingleQuotes(spec)

	return true, &result{
		values: values,
		data: struct {
			Meta                 string
			Replicas             string
			RevisionHistoryLimit string
			Strategy             string
			Selector             string
			PodLabels            string
			PodAnnotations       string
			Spec                 string
		}{
			Meta:                 meta,
			Replicas:             replicas,
			RevisionHistoryLimit: revisionHistoryLimit,
			Strategy:             strategy,
			Selector:             selector,
			PodLabels:            podLabels,
			PodAnnotations:       podAnnotations,
			Spec:                 spec,
		},
	}, nil
}

func replaceSingleQuotes(s string) string {
	r := regexp.MustCompile(`'({{((.*|.*\n.*))}}.*)'`)
	return r.ReplaceAllString(s, "${1}")
}

func addWebhookOption(manifest string) string {
	webhookOptionHeader := "      {{- if .Values.webhook.enabled }}"
	webhookOptionFooter := "      {{- end }}"
	volumes := `      - name: cert
        secret:
          defaultMode: 420
          secretName: webhook-server-cert`
	volumeMounts := `        - mountPath: /tmp/k8s-webhook-server/serving-certs
          name: cert
          readOnly: true`
	manifest = strings.ReplaceAll(manifest, volumes, fmt.Sprintf("%s\n%s\n%s",
		webhookOptionHeader, volumes, webhookOptionFooter))
	manifest = strings.ReplaceAll(manifest, volumeMounts, fmt.Sprintf("%s\n%s\n%s",
		webhookOptionHeader, volumeMounts, webhookOptionFooter))

	re := regexp.MustCompile(`        - containerPort: \d+
          name: webhook-server
          protocol: TCP`)

	manifest = re.ReplaceAllString(manifest, fmt.Sprintf("%s\n%s\n%s", webhookOptionHeader,
		re.FindString(manifest), webhookOptionFooter))
	return manifest
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

func processStrategy(name string, deployment *appsv1.Deployment, values *helmify.Values) (string, error) {
	if deployment.Spec.Strategy.Type == "" {
		return "", nil
	}
	allowedStrategyTypes := map[appsv1.DeploymentStrategyType]bool{
		appsv1.RecreateDeploymentStrategyType:      true,
		appsv1.RollingUpdateDeploymentStrategyType: true,
	}
	if !allowedStrategyTypes[deployment.Spec.Strategy.Type] {
		return "", fmt.Errorf("invalid deployment strategy type: %s", deployment.Spec.Strategy.Type)
	}
	strategyTypeTpl, err := values.Add(string(deployment.Spec.Strategy.Type), name, "strategy", "type")
	if err != nil {
		return "", err
	}
	strategyMap := map[string]interface{}{
		"type": strategyTypeTpl,
	}
	if deployment.Spec.Strategy.Type == appsv1.RollingUpdateDeploymentStrategyType {
		if rollingUpdate := deployment.Spec.Strategy.RollingUpdate; rollingUpdate != nil {
			rollingUpdateMap := map[string]interface{}{}
			setRollingUpdateField := func(value *intstr.IntOrString, fieldName string) error {
				var tpl string
				var err error
				if value.Type == intstr.Int {
					tpl, err = values.Add(value.IntValue(), name, "strategy", "rollingUpdate", fieldName)
				} else {
					tpl, err = values.Add(value.String(), name, "strategy", "rollingUpdate", fieldName)
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
	strategy, err := yamlformat.Marshal(map[string]interface{}{"strategy": strategyMap}, 2)
	if err != nil {
		return "", err
	}
	strategy = strings.ReplaceAll(strategy, "'", "")
	return strategy, nil
}

type result struct {
	data struct {
		Meta                 string
		Replicas             string
		RevisionHistoryLimit string
		Strategy             string
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
