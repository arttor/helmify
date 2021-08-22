package rbac

import (
	"testing"

	"github.com/arttor/helmify/internal"
	"github.com/arttor/helmify/pkg/helmify"
	"github.com/stretchr/testify/assert"
)

const serviceAccountYaml = `apiVersion: v1
kind: ServiceAccount
metadata:
  name: my-operator-controller-manager
  namespace: my-operator-system`

func Test_serviceAccount_Process(t *testing.T) {
	var testInstance serviceAccount

	t.Run("processed", func(t *testing.T) {
		obj := internal.GenerateObj(serviceAccountYaml)
		processed, _, err := testInstance.Process(helmify.ChartInfo{}, obj)
		assert.NoError(t, err)
		assert.Equal(t, true, processed)
	})
	t.Run("skipped", func(t *testing.T) {
		obj := internal.TestNs
		processed, _, err := testInstance.Process(helmify.ChartInfo{}, obj)
		assert.NoError(t, err)
		assert.Equal(t, false, processed)
	})
}
