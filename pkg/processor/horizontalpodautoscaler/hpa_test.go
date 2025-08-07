package horizontalpodautoscaler

import (
	"os"
	"testing"

	"github.com/arttor/helmify/pkg/metadata"

	"github.com/arttor/helmify/internal"
	"github.com/stretchr/testify/assert"
)

const hpaYaml = `apiVersion: autoscaling/v1
kind: HorizontalPodAutoscaler
metadata:
  name: myapp-hpa
spec:
  scaleTargetRef:
    apiVersion: apps/v1
    kind: Deployment
    name: myapp
  minReplicas: 2
  maxReplicas: 10
  targetCPUUtilizationPercentage: 80`

func Test_hpa_Process(t *testing.T) {
	var testInstance hpa

	t.Run("processed", func(t *testing.T) {
		obj := internal.GenerateObj(hpaYaml)
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
