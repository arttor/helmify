package imagePullSecrets

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_imagePullSecrets_RespectExistingSpec(t *testing.T) {
	spec := make(map[string]interface{})

	type ipsReference struct {
		Name string
	}

	spec["imagePullSecrets"] = []ipsReference{
		{Name: "ips"},
	}

	ProcessSpecMap(spec)

	assert.Equal(t, "ips", spec["imagePullSecrets"].([]ipsReference)[0].Name)

}

func Test_imagePullSecrets_ProvideDefault(t *testing.T) {
	spec := make(map[string]interface{})

	ProcessSpecMap(spec)

	ips, found := spec["imagePullSecrets"]
	assert.True(t, found)

	assert.Equal(t, ips, helmExpression)

}
