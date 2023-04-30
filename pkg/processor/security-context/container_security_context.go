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
func ProcessContainerSecurityContext(nameCamel string, specMap map[string]interface{}, values *helmify.Values) error {
	err := processSecurityContext(nameCamel, "containers", specMap, values)
	if err != nil {
		return err
	}
	
	err = processSecurityContext(nameCamel, "initContainers", specMap, values)
	if err != nil {
		return err
	}
	
	return nil
}

func processSecurityContext(nameCamel string, containerType string, specMap map[string]interface{}, values *helmify.Values) error {
	if containers, defined := specMap[containerType]; defined {
		for _, container := range containers.([]interface{}) {
			castedContainer := container.(map[string]interface{})
			containerName := strcase.ToLowerCamel(castedContainer["name"].(string))
			if _, defined2 := castedContainer["securityContext"]; defined2 {
				err := setSecContextValue(nameCamel, containerName, castedContainer, values)
				if err != nil {
					return err
				}
			}
		}
		err := unstructured.SetNestedField(specMap, containers, containerType)
		if err != nil {
			return err
		}
	}
	return nil
}

func setSecContextValue(resourceName string, containerName string, castedContainer map[string]interface{}, values *helmify.Values) error {
	if castedContainer["securityContext"] != nil {
		err := unstructured.SetNestedField(*values, castedContainer["securityContext"], resourceName, containerName, cscValueName)
		if err != nil {
			return err
		}
		
		valueString := fmt.Sprintf(helmTemplate, resourceName, containerName)
		
		err = unstructured.SetNestedField(castedContainer, valueString, sc)
		if err != nil {
			return err
		}
	}
	return nil
}
