package imagePullPolicy

import (
	"fmt"
	"testing"

	"github.com/arttor/helmify/pkg/helmify"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/yaml"
)

func Test_imagePullSecrets_RespectExistingSpec(t *testing.T) {
	spec := make(map[string]interface{}, 1)

	spec["containers"] = []interface{}{
		map[string]interface{}{
			"name":            "mycontainer",
			"imagePullPolicy": string(corev1.PullAlways),
		},
	}

	values := &helmify.Values{}
	err := ProcessSpecMap("", spec, values)
	assert.Nil(t, err)
	b, err := yaml.Marshal(spec)
	assert.Contains(t, string(b), fmt.Sprintf(helmTemplate, "", "mycontainer"))
	castedValue := (*values)[""].(map[string]interface{})["mycontainer"]
	assert.Equal(t, string(corev1.PullAlways), castedValue.(map[string]interface{})["imagePullPolicy"])

}
