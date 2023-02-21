package constraints

import (
	"github.com/arttor/helmify/pkg/helmify"
	yamlformat "github.com/arttor/helmify/pkg/yaml"
)

const tolerations = "tolerations"
const topology = "topologySpreadConstraints"
const nodeSelector = "nodeSelector"

const topologyExpression = "\n{{- if .Values.topologySpreadConstraints }}\n" +
	"      topologySpreadConstraints: {{- include \"tplvalues.render\" (dict \"value\" .Values.topologySpreadConstraints \"context\" $) | nindent 8 }}\n" +
	"{{- end }}\n"

const nodeSelectorExpression = "{{- if .Values.nodeSelector }}\n" +
	"      nodeSelector: {{- include \"tplvalues.render\" ( dict \"value\" .Values.nodeSelector \"context\" $) | nindent 8 }}\n" +
	"{{- end }}\n"

const tolerationsExpression = "{{- if .Values.tolerations }}\n" +
	"      tolerations: {{- include \"tplvalues.render\" (dict \"value\" .Values.tolerations \"context\" .) | nindent 8 }}\n" +
	"{{- end }}\n"

// ProcessSpecMap adds 'topologyConstraints' to the podSpec in specMap, if it doesn't
// already has one defined.
func ProcessSpecMap(name string, specMap map[string]interface{}, values *helmify.Values) string {

	mapConstraint(name, specMap, topology, []interface{}{}, values)
	mapConstraint(name, specMap, tolerations, []interface{}{}, values)
	mapConstraint(name, specMap, nodeSelector, map[string]string{}, values)

	spec, err := yamlformat.Marshal(specMap, 6)
	if err != nil {
		return ""
	}
	return spec + topologyExpression + nodeSelectorExpression + tolerationsExpression
}

func mapConstraint(name string, specMap map[string]interface{}, constraint string, override interface{}, values *helmify.Values) {
	if specMap[constraint] != nil {
		(*values)[name].(map[string]interface{})[constraint] = specMap[constraint].(interface{})
	} else {
		(*values)[name].(map[string]interface{})[constraint] = override
	}
	delete(specMap, constraint)
}
