package crd

import (
	"testing"

	"github.com/arttor/helmify/internal"
	"github.com/arttor/helmify/pkg/helmify"
	"github.com/stretchr/testify/assert"
)

const (
	strCRD = `apiVersion: apiextensions.k8s.io/v1
kind: CustomResourceDefinition
metadata:
  annotations:
    controller-gen.kubebuilder.io/version: v0.4.1
  creationTimestamp: null
  name: cephvolumes.test.example.com
spec:
  group: test.example.com
  names:
    kind: CephVolume
    listKind: CephVolumeList
    plural: cephvolumes
    singular: cephvolume
  scope: Namespaced
`
)

func Test_crd_Process(t *testing.T) {
	var testInstance crd

	t.Run("processed", func(t *testing.T) {
		obj := internal.GenerateObj(strCRD)
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
