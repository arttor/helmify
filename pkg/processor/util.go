package processor

import (
	"fmt"
	"strings"

	"github.com/arttor/helmify/pkg/helmify"
	yamlformat "github.com/arttor/helmify/pkg/yaml"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type ReplicaTyped struct {
	Spec ReplicaTypedSpec
}

type ReplicaTypedSpec struct {
	Replicas *int32
}

func ProcessReplicas(name string, r *int32, values *helmify.Values) (string, error) {
	obj := ReplicaTyped{
		Spec: ReplicaTypedSpec{
			Replicas: r,
		},
	}
	if obj.Spec.Replicas == nil {
		return "", nil
	}
	replicasTpl, err := values.Add(int64(*obj.Spec.Replicas), name, "replicas")
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

type SelectorTyped struct {
	Spec SelectorTypedSpec
}

type SelectorTypedSpec struct {
	Selector *metav1.LabelSelector
}

const selectorTempl = `%[1]s
{{- include "%[2]s.selectorLabels" . | nindent 6 }}
%[3]s`

func ProcessSelector(appMeta helmify.AppMetadata, s *metav1.LabelSelector) (string, error) {
	obj := SelectorTyped{
		Spec: SelectorTypedSpec{
			Selector: s,
		},
	}
	matchLabels, err := yamlformat.Marshal(map[string]interface{}{"matchLabels": obj.Spec.Selector.MatchLabels}, 0)
	if err != nil {
		return "", err
	}
	matchExpr := ""
	if obj.Spec.Selector.MatchExpressions != nil {
		matchExpr, err = yamlformat.Marshal(map[string]interface{}{"matchExpressions": obj.Spec.Selector.MatchExpressions}, 0)
		if err != nil {
			return "", err
		}
	}
	selector := fmt.Sprintf(selectorTempl, matchLabels, appMeta.ChartName(), matchExpr)
	selector = strings.Trim(selector, " \n")
	selector = string(yamlformat.Indent([]byte(selector), 4))

	return selector, nil
}
