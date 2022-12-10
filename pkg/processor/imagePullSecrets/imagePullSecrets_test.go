package imagePullSecrets

import (
	"testing"

	"github.com/arttor/helmify/pkg/helmify"
	"github.com/stretchr/testify/assert"
)

func Test_imagePullSecrets_RespectExistingSpec(t *testing.T) {
	Enabled = true
	spec := make(map[string]interface{})

	type ipsReference struct {
		Name string
	}

	spec["imagePullSecrets"] = []ipsReference{
		{Name: "ips"},
	}

	values := &helmify.Values{}
	ProcessSpecMap(spec, values)

	assert.Equal(t, "ips", spec["imagePullSecrets"].([]ipsReference)[0].Name)
	assert.Equal(t, 0, len(*values))

}

func Test_imagePullSecrets_ProvideDefault(t *testing.T) {
	Enabled = true
	spec := make(map[string]interface{})

	values := &helmify.Values{}
	ProcessSpecMap(spec, values)

	ips, found := spec["imagePullSecrets"]
	assert.True(t, found)

	assert.Equal(t, ips, helmExpression)
	assert.Equal(t, 1, len(*values))
}

func Test_imagePullSecrets_DoNothingIfNotEnabled(t *testing.T) {
	Enabled = false
	spec := make(map[string]interface{})

	values := &helmify.Values{}
	ProcessSpecMap(spec, values)

	_, found := spec["imagePullSecrets"]
	assert.False(t, found)

	assert.Equal(t, 0, len(*values))
}
