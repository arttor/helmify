package imagePullPolicy

import (
	"fmt"

	"github.com/arttor/helmify/pkg/helmify"
	"github.com/iancoleman/strcase"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

const helmTemplate = "{{ .Values.%[1]s.%[2]s.imagePullPolicy }}"

// ProcessSpecMap templates 'imagePullPolicy' to the containers in specMap, if one is defined
func ProcessSpecMap(name string, specMap map[string]interface{}, values *helmify.Values) error {

	cs, _, err := unstructured.NestedSlice(specMap, "containers")

	if err != nil {
		return err
	}

	newContainers := make([]interface{}, len(cs))
	for i, c := range cs {
		castedContainer := c.(map[string]interface{})
		containerName := strcase.ToLowerCamel(castedContainer["name"].(string))
		if castedContainer["imagePullPolicy"] != nil {
			err = unstructured.SetNestedField(*values, castedContainer["imagePullPolicy"], name, containerName, "imagePullPolicy")
			if err != nil {
				return err
			}
			castedContainer["imagePullPolicy"] = fmt.Sprintf(helmTemplate, name, containerName)
		}
		newContainers[i] = castedContainer
	}
	return unstructured.SetNestedSlice(specMap, newContainers, "containers")
}
