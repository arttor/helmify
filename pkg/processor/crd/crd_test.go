package crd

import (
	"strings"
	"testing"

	"github.com/arttor/helmify/pkg/config"
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

	t.Run("wrapped with condition", func(t *testing.T) {
		obj := internal.GenerateObj(strCRD)

		conditions := []string{"crds.create", "crds.enabled", "installCRDs"}

		for _, cond := range conditions {
			t.Run(cond, func(t *testing.T) {
				meta := metadata.New(config.Config{WrapCRDs: true, WrapCRDsCondition: cond})
				processed, tmpl, err := testInstance.Process(meta, obj)
				assert.NoError(t, err)
				assert.True(t, processed)
				assert.NotNil(t, tmpl)

				data := string(tmpl.(*result).data)

				assert.Contains(t, data, "{{- if .Values."+cond+" }}", "template should start with conditional")
				assert.Contains(t, data, "{{- end }}", "template should end with conditional")

				values := tmpl.(*result).values
				val, ok := getValue(values, cond)
				assert.True(t, ok, "expected key crds."+cond+" in values")
				assert.Equal(t, true, val)
			})
		}
	})
}

func getValue(values map[string]any, path string) (any, bool) {
	parts := strings.Split(path, ".")
	current := any(values)

	for _, part := range parts {
		m, ok := current.(map[string]any)
		if !ok {
			return nil, false
		}
		current, ok = m[part]
		if !ok {
			return nil, false
		}
	}

	return current, true
}
