package rbac

import (
	"github.com/arttor/helmify/internal"
	"github.com/arttor/helmify/pkg/helmify"
	"github.com/stretchr/testify/assert"
	"testing"
)

const roleYaml = `apiVersion: rbac.authorization.k8s.io/v1
kind: Role
metadata:
  name: my-operator-leader-election-role
  namespace: my-operator-system
rules:
- apiGroups:
  - ""
  resources:
  - configmaps
  verbs:
  - get
  - list
  - watch`

func Test_role_Process(t *testing.T) {
	var testInstance role

	t.Run("processed", func(t *testing.T) {
		obj := internal.GenerateObj(roleYaml)
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
