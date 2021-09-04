package configmap

import (
	"github.com/arttor/helmify/pkg/metadata"
	"testing"

	"github.com/arttor/helmify/internal"
	"github.com/stretchr/testify/assert"
)

const (
	strConfigmap = `apiVersion: v1
kind: ConfigMap
metadata:
  name: my-operator-manager-config
  namespace: my-operator-system
data:
  dummyconfigmapkey: dummyconfigmapvalue
  controller_manager_config.yaml: |
    apiVersion: controller-runtime.sigs.k8s.io/v1alpha1
    kind: ControllerManagerConfig
    health:
      healthProbeBindAddress: :8081`
)

func Test_configMap_Process(t *testing.T) {
	var testInstance configMap

	t.Run("processed", func(t *testing.T) {
		obj := internal.GenerateObj(strConfigmap)
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
