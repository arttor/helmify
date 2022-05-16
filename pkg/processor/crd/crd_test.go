package crd

import (
	"testing"

	"github.com/arttor/helmify/pkg/metadata"

	"github.com/arttor/helmify/internal"
	"github.com/stretchr/testify/assert"
)

const (
	strCRD = `apiVersion: apiextensions.k8s.io/v1
kind: CustomResourceDefinition
metadata:
  annotations:
    cert-manager.io/inject-ca-from: my-operator-system/my-operator-serving-cert
  creationTimestamp: null
  name: cephvolumes.test.example.com
  labels:
    example: true
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
