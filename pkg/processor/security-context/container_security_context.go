package security_context

import (
	"fmt"

	"github.com/arttor/helmify/pkg/helmify"
	"github.com/iancoleman/strcase"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

const (
	sc           = "securityContext"
	cscValueName = "containerSecurityContext"
	helmTemplate = "{{- toYaml .Values.%[1]s.%[2]s.containerSecurityContext | nindent 10 }}"
)

// ProcessContainerSecurityContext adds 'securityContext' to the podSpec in specMap, if it doesn't have one already defined.
func ProcessContainerSecurityContext(nameCamel string, specMap map[string]interface{}, values *helmify.Values) {
	if _, defined := specMap["containers"]; defined {
		containers, _, _ := unstructured.NestedSlice(specMap, "containers")
		for _, container := range containers {
			castedContainer := container.(map[string]interface{})
			containerName := strcase.ToLowerCamel(castedContainer["name"].(string))
			if _, defined2 := castedContainer["securityContext"]; defined2 {
				setSecContextValue(nameCamel, containerName, castedContainer, values)
			}
		}
		unstructured.SetNestedSlice(specMap, containers, "containers")
	}
}

func setSecContextValue(resourceName string, containerName string, castedContainer map[string]interface{}, values *helmify.Values) {
	if castedContainer["securityContext"] != nil {
		unstructured.SetNestedField(*values, castedContainer["securityContext"], resourceName, containerName, cscValueName)

		valueString := fmt.Sprintf(helmTemplate, resourceName, containerName)

		unstructured.SetNestedField(castedContainer, valueString, sc)
	}
}
