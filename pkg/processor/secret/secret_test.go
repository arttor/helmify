package secret

import (
	"testing"

	"github.com/arttor/helmify/pkg/metadata"

	"github.com/arttor/helmify/internal"
	"github.com/stretchr/testify/assert"
)

const secretYaml = `apiVersion: v1
data:
  VAR1: bXlfc2VjcmV0X3Zhcl8x
  VAR2: bXlfc2VjcmV0X3Zhcl8y
stringData:
  VAR3: string secret
kind: Secret
metadata:
  name: my-operator-secret-vars
  namespace: my-operator-system
type: opaque`

func Test_secret_Process(t *testing.T) {
	var testInstance secret

	t.Run("processed", func(t *testing.T) {
		obj := internal.GenerateObj(secretYaml)
		processed, _, err := testInstance.Process(&metadata.Service{}, obj)
		assert.NoError(t, err)
		assert.Equal(t, true, processed)
	})
	t.Run("skipped", func(t *testing.T) {
		obj := internal.TestNs
		processed, _, err := testInstance.Process(&metadata.Service{}, obj)
		assert.NoError(t, err)
		assert.Equal(t, false, processed)
	})
}
