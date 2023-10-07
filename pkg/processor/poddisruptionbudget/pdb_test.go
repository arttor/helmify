package poddisruptionbudget

import (
	"os"
	"testing"

	"github.com/arttor/helmify/pkg/metadata"

	"github.com/arttor/helmify/internal"
	"github.com/stretchr/testify/assert"
)

const pdbYaml = `apiVersion: policy/v1
kind: PodDisruptionBudget
metadata:
  labels:
    control-plane: controller-manager
  name: my-operator-controller-manager-pdb
  namespace: my-operator-system
spec:
  minAvailable: 2
  selector:
    matchLabels:
      control-plane: controller-manager`

func Test_pdb_Process(t *testing.T) {
	var testInstance pdb

	t.Run("processed", func(t *testing.T) {
		obj := internal.GenerateObj(pdbYaml)
		processed, tt, err := testInstance.Process(&metadata.Service{}, obj)
		_ = tt.Write(os.Stdout)
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
