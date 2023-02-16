package imagePullSecrets

import "github.com/arttor/helmify/pkg/helmify"

const helmExpression = "{{ .Values.imagePullSecrets | default list | toJson }}"

// ProcessSpecMap adds 'imagePullSecrets' to the podSpec in specMap, if it doesn't
// already has one defined.
func ProcessSpecMap(specMap map[string]interface{}, values *helmify.Values) {

	if _, defined := specMap["imagePullSecrets"]; !defined {
		specMap["imagePullSecrets"] = helmExpression
		(*values)["imagePullSecrets"] = []string{}
	}
}
