package app

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_injectAnnotations(t *testing.T) {
	t.Run("injects into deployment template metadata", func(t *testing.T) {
		yaml := `apiVersion: apps/v1
kind: Deployment
spec:
  template:
    metadata:
      labels:
        app: test
    spec:
      containers:
      - name: app`

		result := injectAnnotations(yaml, `checksum/configmap/config: {{ include (print $.Template.BasePath "/config.yaml") . | sha256sum }}`)
		assert.Contains(t, result, "      annotations:")
		assert.Contains(t, result, "        checksum/configmap/config:")
	})

	t.Run("appends to existing annotations block", func(t *testing.T) {
		yaml := `apiVersion: apps/v1
kind: Deployment
spec:
  template:
    metadata:
      annotations:
        existing: value
      labels:
        app: test`

		result := injectAnnotations(yaml, "checksum/configmap/config: hash")
		assert.Contains(t, result, "        existing: value")
		assert.Contains(t, result, "        checksum/configmap/config: hash")
	})

	t.Run("injects multiple annotations", func(t *testing.T) {
		yaml := `spec:
  template:
    metadata:
      labels:
        app: test`

		result := injectAnnotations(yaml, "checksum/configmap/config: hash1\nchecksum/secret/db: hash2")
		assert.Contains(t, result, "        checksum/configmap/config: hash1")
		assert.Contains(t, result, "        checksum/secret/db: hash2")
	})

	t.Run("no injection when no template metadata", func(t *testing.T) {
		yaml := `apiVersion: v1
kind: ConfigMap
metadata:
  name: test`

		result := injectAnnotations(yaml, "checksum/x: hash")
		assert.NotContains(t, result, "annotations:")
	})

	t.Run("cronjob-style deeper nesting", func(t *testing.T) {
		yaml := `spec:
  jobTemplate:
    spec:
      template:
        metadata:
          labels:
            app: test`

		result := injectAnnotations(yaml, "checksum/configmap/config: hash")
		assert.Contains(t, result, "          annotations:")
		assert.Contains(t, result, "            checksum/configmap/config: hash")
	})

	t.Run("does not inject into template: without parent spec:", func(t *testing.T) {
		yaml := `apiVersion: v1
kind: SomeResource
template:
  metadata:
    name: test`

		result := injectAnnotations(yaml, "checksum/x: hash")
		assert.NotContains(t, result, "annotations:")
	})

	t.Run("does not inject into unrelated template: under data:", func(t *testing.T) {
		yaml := `apiVersion: v1
kind: ConfigMap
data:
  template:
    metadata:
      something: else`

		result := injectAnnotations(yaml, "checksum/x: hash")
		assert.NotContains(t, result, "annotations:")
	})

	t.Run("preserves all original lines", func(t *testing.T) {
		yaml := `apiVersion: apps/v1
kind: Deployment
metadata:
  name: my-deploy
spec:
  template:
    metadata:
      labels:
        app: test
    spec:
      containers:
      - name: app
        image: nginx:latest`

		result := injectAnnotations(yaml, "checksum/x: hash")
		assert.Contains(t, result, "kind: Deployment")
		assert.Contains(t, result, "  name: my-deploy")
		assert.Contains(t, result, "        app: test")
		assert.Contains(t, result, "        image: nginx:latest")
		assert.Contains(t, result, "        checksum/x: hash")
	})

	t.Run("only injects once", func(t *testing.T) {
		yaml := `spec:
  template:
    metadata:
      labels:
        app: first
---
spec:
  template:
    metadata:
      labels:
        app: second`

		result := injectAnnotations(yaml, "checksum/x: hash")
		assert.Equal(t, 1, strings.Count(result, "annotations:"))
	})
}
