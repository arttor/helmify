package service

import (
	"github.com/arttor/helmify/internal"
	"github.com/arttor/helmify/pkg/helmify"
	"github.com/stretchr/testify/assert"
	"testing"
)

const svcYaml = `apiVersion: v1
kind: Service
metadata:
  labels:
    control-plane: controller-manager
  name: my-operator-controller-manager-metrics-service
  namespace: my-operator-system
spec:
  ports:
  - name: https
    port: 8443
    targetPort: https
  selector:
    control-plane: controller-manager`

func Test_svc_Process(t *testing.T) {
	var testInstance svc

	t.Run("processed", func(t *testing.T) {
		obj := internal.GenerateObj(svcYaml)
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
