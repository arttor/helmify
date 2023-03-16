package rbac

import (
	"fmt"
	"io"
	"strings"
	"text/template"

	"github.com/arttor/helmify/pkg/processor"

	"github.com/arttor/helmify/pkg/helmify"
	yamlformat "github.com/arttor/helmify/pkg/yaml"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

var roleTempl, _ = template.New("clusterRole").Parse(
	`{{ .Meta }}
{{- if .AggregationRule }}
{{ .AggregationRule }}
{{- end}}
{{ .Rules }}`)

var clusterRoleGVC = schema.GroupVersionKind{
	Group:   "rbac.authorization.k8s.io",
	Version: "v1",
	Kind:    "ClusterRole",
}
var roleGVC = schema.GroupVersionKind{
	Group:   "rbac.authorization.k8s.io",
	Version: "v1",
	Kind:    "Role",
}

// Role creates processor for k8s Role and ClusterRole resources.
func Role() helmify.Processor {
	return &role{}
}

type role struct{}

// Process k8s ClusterRole object into template. Returns false if not capable of processing given resource type.
func (r role) Process(appMeta helmify.AppMetadata, obj *unstructured.Unstructured) (bool, helmify.Template, error) {
	var aggregationRule string

	if obj.GroupVersionKind() != clusterRoleGVC && obj.GroupVersionKind() != roleGVC {
		return false, nil, nil
	}

	meta, err := processor.ProcessObjMeta(appMeta, obj)
	if err != nil {
		return true, nil, err
	}

	if existingAggRule := obj.Object["aggregationRule"]; existingAggRule != nil {
		if obj.GroupVersionKind().Kind == "Role" {
			return true, nil, fmt.Errorf("unable to set aggregationRule to the kind Role in %q: unsupported", obj.GetName())
		}

		if existingAggRule.(map[string]interface{})["clusterRoleSelectors"] != nil {
			aggRuleMap := map[string]interface{}{"aggregationRule": existingAggRule}

			aggregationRule, err = yamlformat.Marshal(aggRuleMap, 0)
			if err != nil {
				return true, nil, err
			}
		}
	}

	rules, err := yamlformat.Marshal(map[string]interface{}{"rules": obj.Object["rules"]}, 0)
	if err != nil {
		return true, nil, err
	}

	return true, &crResult{
		name: appMeta.TrimName(obj.GetName()),
		data: struct {
			Meta            string
			AggregationRule string
			Rules           string
		}{Meta: meta, AggregationRule: aggregationRule, Rules: rules},
	}, nil
}

type crResult struct {
	name string
	data struct {
		Meta            string
		AggregationRule string
		Rules           string
	}
}

func (r *crResult) Filename() string {
	return strings.TrimSuffix(r.name, "-role") + "-rbac.yaml"
}

func (r *crResult) GVK() schema.GroupVersionKind {
	return clusterRoleGVC
}

func (r *crResult) Values() helmify.Values {
	return helmify.Values{}
}

func (r *crResult) Write(writer io.Writer) error {
	return roleTempl.Execute(writer, r.data)
}
