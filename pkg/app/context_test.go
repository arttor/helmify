package app

import (
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

		annotations := `checksum/configmap/config: {{ include (print $.Template.BasePath "/config.yaml") . | sha256sum }}`
		result := injectAnnotations(yaml, annotations)
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
        app: test
    spec:
      containers:
      - name: app`

		annotations := `checksum/configmap/config: {{ sha256sum }}`
		result := injectAnnotations(yaml, annotations)
		assert.Contains(t, result, "        existing: value")
		assert.Contains(t, result, "        checksum/configmap/config:")
	})

	t.Run("injects multiple annotations", func(t *testing.T) {
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

		annotations := "checksum/configmap/config: hash1\nchecksum/secret/db: hash2"
		result := injectAnnotations(yaml, annotations)
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

	t.Run("handles deeper nesting for cronjob-style templates", func(t *testing.T) {
		// StatefulSet and DaemonSet use "  template:" at 2 spaces indent
		yaml := `spec:
  template:
    metadata:
      labels:
        app: test
    spec:
      containers: []`

		result := injectAnnotations(yaml, "checksum/configmap/config: hash")
		assert.Contains(t, result, "      annotations:")
		assert.Contains(t, result, "        checksum/configmap/config: hash")
	})
}
