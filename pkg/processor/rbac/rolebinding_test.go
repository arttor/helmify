package rbac

import (
	"github.com/arttor/helmify/internal"
	"github.com/arttor/helmify/pkg/helmify"
	"github.com/stretchr/testify/assert"
	"testing"
)

const roleBindingYaml = `apiVersion: rbac.authorization.k8s.io/v1
kind: RoleBinding
metadata:
  name: my-operator-leader-election-rolebinding
  namespace: my-operator-system
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: Role
  name: my-operator-leader-election-role
subjects:
- kind: ServiceAccount
  name: my-operator-controller-manager
  namespace: my-operator-system`

func Test_roleBinding_Process(t *testing.T) {
	var testInstance roleBinding

	t.Run("processed", func(t *testing.T) {
		obj := internal.GenerateObj(roleBindingYaml)
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
