package processor

import (
	"github.com/arttor/helmify/pkg/config"
	"testing"

	"github.com/arttor/helmify/internal"
	"github.com/arttor/helmify/pkg/metadata"
	"github.com/stretchr/testify/assert"
)

const pvcYaml = `apiVersion: v1
kind: PersistentVolumeClaim
metadata:
  name: my-operator-pvc-lim
spec:
  accessModes:
    - ReadWriteOnce
  resources:
    requests:
      storage: 2Gi
  storageClassName: cust1-mypool-lim`

func Test_dft_Process(t *testing.T) {

	t.Run("skip namespace", func(t *testing.T) {
		testMeta := metadata.New(config.Config{ChartName: "chart-name"})
		testMeta.Load(internal.TestNs)
		testProcessor := Default()
		processed, templ, err := testProcessor.Process(testMeta, internal.TestNs)
		assert.NoError(t, err)
		assert.True(t, processed)
		assert.Nil(t, templ)
	})
	t.Run("process", func(t *testing.T) {
		obj := internal.GenerateObj(pvcYaml)
		testMeta := metadata.New(config.Config{ChartName: "chart-name"})
		testMeta.Load(obj)
		testProcessor := Default()
		processed, templ, err := testProcessor.Process(testMeta, obj)
		assert.NoError(t, err)
		assert.True(t, processed)
		assert.NotNil(t, templ)
	})
}
