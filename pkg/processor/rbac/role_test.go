package rbac

import (
	"testing"

	"github.com/arttor/helmify/internal"
	"github.com/arttor/helmify/pkg/helmify"
	"github.com/stretchr/testify/assert"
)

const clusterRoleYaml = `apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRole
metadata:
  creationTimestamp: null
  name: my-operator-manager-role
rules:
- apiGroups:
  - ""
  resources:
  - pods
  verbs:
  - get
  - list`

func Test_clusterRole_Process(t *testing.T) {
	var testInstance role

	t.Run("processed", func(t *testing.T) {
		obj := internal.GenerateObj(clusterRoleYaml)
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
