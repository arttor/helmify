package pod

import (
	"fmt"
	"strings"
	"testing"

	"github.com/arttor/helmify/internal"
	"github.com/arttor/helmify/pkg/config"
	"github.com/arttor/helmify/pkg/metadata"
	"github.com/stretchr/testify/assert"
)

const checksumConfigMapYaml = `apiVersion: v1
kind: ConfigMap
metadata:
  name: %s
  namespace: my-app-system`

const checksumSecretYaml = `apiVersion: v1
kind: Secret
metadata:
  name: %s
  namespace: my-app-system`

const checksumDeployYaml = `apiVersion: apps/v1
kind: Deployment
metadata:
  name: my-app-deploy
  namespace: my-app-system`

// deploymentWith builds a Deployment YAML that references the given configmaps/secrets.
func deploymentWith(envFromCMs, envFromSecrets, envCMKeyRefs, envSecretKeyRefs, volumeCMs, volumeSecrets []string) string {
	var envFromParts, envParts, volumeParts []string

	for _, name := range envFromCMs {
		envFromParts = append(envFromParts, fmt.Sprintf(`        - configMapRef:
            name: %s`, name))
	}
	for _, name := range envFromSecrets {
		envFromParts = append(envFromParts, fmt.Sprintf(`        - secretRef:
            name: %s`, name))
	}
	for _, name := range envCMKeyRefs {
		envParts = append(envParts, fmt.Sprintf(`        - name: VAR
          valueFrom:
            configMapKeyRef:
              name: %s
              key: key1`, name))
	}
	for _, name := range envSecretKeyRefs {
		envParts = append(envParts, fmt.Sprintf(`        - name: VAR
          valueFrom:
            secretKeyRef:
              name: %s
              key: key1`, name))
	}
	for _, name := range volumeCMs {
		volumeParts = append(volumeParts, fmt.Sprintf(`      - name: vol
        configMap:
          name: %s`, name))
	}
	for _, name := range volumeSecrets {
		volumeParts = append(volumeParts, fmt.Sprintf(`      - name: vol
        secret:
          secretName: %s`, name))
	}

	y := `apiVersion: apps/v1
kind: Deployment
metadata:
  name: my-app-deploy
  namespace: my-app-system
spec:
  selector:
    matchLabels:
      app: test
  template:
    metadata:
      labels:
        app: test
    spec:
      containers:
      - name: app
        image: nginx:latest`

	if len(envFromParts) > 0 {
		y += "\n        envFrom:\n" + strings.Join(envFromParts, "\n")
	}
	if len(envParts) > 0 {
		y += "\n        env:\n" + strings.Join(envParts, "\n")
	}
	if len(volumeParts) > 0 {
		y += "\n      volumes:\n" + strings.Join(volumeParts, "\n")
	}
	return y
}

func TestChecksumAnnotations(t *testing.T) {
	t.Run("no references", func(t *testing.T) {
		meta := metadata.New(config.Config{})
		obj := internal.GenerateObj(deploymentWith(nil, nil, nil, nil, nil, nil))
		result := ChecksumAnnotations(meta, obj, nil, nil)
		assert.Equal(t, "", result)
	})

	t.Run("non-workload returns empty", func(t *testing.T) {
		meta := setupChecksumMeta(t, []string{"my-app-config"}, nil)
		obj := internal.GenerateObj(fmt.Sprintf(checksumConfigMapYaml, "my-app-config"))
		result := ChecksumAnnotations(meta, obj, map[string]string{"my-app-config": "config.yaml"}, nil)
		assert.Equal(t, "", result)
	})

	t.Run("configmap via envFrom", func(t *testing.T) {
		meta := setupChecksumMeta(t, []string{"my-app-config"}, nil)
		obj := internal.GenerateObj(deploymentWith([]string{"my-app-config"}, nil, nil, nil, nil, nil))
		cmFiles := map[string]string{"my-app-config": "config.yaml"}
		result := ChecksumAnnotations(meta, obj, cmFiles, nil)
		assert.Contains(t, result, "checksum/configmap/config:")
		assert.Contains(t, result, `{{ include (print $.Template.BasePath "/config.yaml") . | sha256sum }}`)
	})

	t.Run("secret via envFrom", func(t *testing.T) {
		meta := setupChecksumMeta(t, nil, []string{"my-app-secret"})
		obj := internal.GenerateObj(deploymentWith(nil, []string{"my-app-secret"}, nil, nil, nil, nil))
		secFiles := map[string]string{"my-app-secret": "secret.yaml"}
		result := ChecksumAnnotations(meta, obj, nil, secFiles)
		assert.Contains(t, result, "checksum/secret/secret:")
		assert.Contains(t, result, `{{ include (print $.Template.BasePath "/secret.yaml") . | sha256sum }}`)
	})

	t.Run("configmap via env valueFrom", func(t *testing.T) {
		meta := setupChecksumMeta(t, []string{"my-app-config"}, nil)
		obj := internal.GenerateObj(deploymentWith(nil, nil, []string{"my-app-config"}, nil, nil, nil))
		cmFiles := map[string]string{"my-app-config": "config.yaml"}
		result := ChecksumAnnotations(meta, obj, cmFiles, nil)
		assert.Contains(t, result, "checksum/configmap/config:")
	})

	t.Run("secret via env valueFrom", func(t *testing.T) {
		meta := setupChecksumMeta(t, nil, []string{"my-app-secret"})
		obj := internal.GenerateObj(deploymentWith(nil, nil, nil, []string{"my-app-secret"}, nil, nil))
		secFiles := map[string]string{"my-app-secret": "secret.yaml"}
		result := ChecksumAnnotations(meta, obj, nil, secFiles)
		assert.Contains(t, result, "checksum/secret/secret:")
	})

	t.Run("configmap via volume", func(t *testing.T) {
		meta := setupChecksumMeta(t, []string{"my-app-config"}, nil)
		obj := internal.GenerateObj(deploymentWith(nil, nil, nil, nil, []string{"my-app-config"}, nil))
		cmFiles := map[string]string{"my-app-config": "config.yaml"}
		result := ChecksumAnnotations(meta, obj, cmFiles, nil)
		assert.Contains(t, result, "checksum/configmap/config:")
	})

	t.Run("secret via volume", func(t *testing.T) {
		meta := setupChecksumMeta(t, nil, []string{"my-app-secret"})
		obj := internal.GenerateObj(deploymentWith(nil, nil, nil, nil, nil, []string{"my-app-secret"}))
		secFiles := map[string]string{"my-app-secret": "secret.yaml"}
		result := ChecksumAnnotations(meta, obj, nil, secFiles)
		assert.Contains(t, result, "checksum/secret/secret:")
	})

	t.Run("external configmap skipped", func(t *testing.T) {
		meta := setupChecksumMeta(t, []string{"my-app-config"}, nil)
		obj := internal.GenerateObj(deploymentWith([]string{"external-config", "my-app-config"}, nil, nil, nil, nil, nil))
		cmFiles := map[string]string{"my-app-config": "config.yaml"}
		result := ChecksumAnnotations(meta, obj, cmFiles, nil)
		assert.Contains(t, result, "checksum/configmap/config:")
		assert.NotContains(t, result, "external-config")
	})

	t.Run("multiple configmaps and secrets sorted", func(t *testing.T) {
		meta := setupChecksumMeta(t, []string{"my-app-config-a", "my-app-config-b"}, []string{"my-app-secret-x"})
		obj := internal.GenerateObj(deploymentWith(
			[]string{"my-app-config-b", "my-app-config-a"},
			[]string{"my-app-secret-x"},
			nil, nil, nil, nil,
		))
		cmFiles := map[string]string{
			"my-app-config-a": "config-a.yaml",
			"my-app-config-b": "config-b.yaml",
		}
		secFiles := map[string]string{"my-app-secret-x": "secret-x.yaml"}
		result := ChecksumAnnotations(meta, obj, cmFiles, secFiles)
		assert.Contains(t, result, "checksum/configmap/config-a:")
		assert.Contains(t, result, "checksum/configmap/config-b:")
		assert.Contains(t, result, "checksum/secret/secret-x:")
	})

	t.Run("nil metadata maps safe", func(t *testing.T) {
		meta := &metadata.Service{}
		obj := internal.GenerateObj(deploymentWith([]string{"some-config"}, nil, nil, nil, nil, nil))
		result := ChecksumAnnotations(meta, obj, nil, nil)
		assert.Equal(t, "", result)
	})

	t.Run("deduplicates same configmap from multiple sources", func(t *testing.T) {
		meta := setupChecksumMeta(t, []string{"my-app-config"}, nil)
		obj := internal.GenerateObj(deploymentWith([]string{"my-app-config"}, nil, nil, nil, []string{"my-app-config"}, nil))
		cmFiles := map[string]string{"my-app-config": "config.yaml"}
		result := ChecksumAnnotations(meta, obj, cmFiles, nil)
		assert.Equal(t, `checksum/configmap/config: {{ include (print $.Template.BasePath "/config.yaml") . | sha256sum }}`, result)
	})

	t.Run("no collision when configmap and secret have same trimmed name", func(t *testing.T) {
		meta := setupChecksumMeta(t, []string{"my-app-credentials"}, []string{"my-app-credentials"})
		obj := internal.GenerateObj(deploymentWith([]string{"my-app-credentials"}, []string{"my-app-credentials"}, nil, nil, nil, nil))
		cmFiles := map[string]string{"my-app-credentials": "credentials.yaml"}
		secFiles := map[string]string{"my-app-credentials": "credentials.yaml"}
		result := ChecksumAnnotations(meta, obj, cmFiles, secFiles)
		assert.Contains(t, result, "checksum/configmap/credentials:")
		assert.Contains(t, result, "checksum/secret/credentials:")
		assert.Equal(t, 2, len(strings.Split(result, "\n")))
	})

	t.Run("uses actual filename not trimmed name for path", func(t *testing.T) {
		meta := setupChecksumMeta(t, []string{"my-app-config"}, nil)
		obj := internal.GenerateObj(deploymentWith([]string{"my-app-config"}, nil, nil, nil, nil, nil))
		// Simulate all resources in a single input file
		cmFiles := map[string]string{"my-app-config": "input.yaml"}
		result := ChecksumAnnotations(meta, obj, cmFiles, nil)
		assert.Contains(t, result, `/input.yaml")`)
		assert.NotContains(t, result, `/config.yaml")`)
	})
}

// setupChecksumMeta creates a metadata.Service with the given configmaps and secrets loaded,
// plus a deployment to establish a common prefix of "my-app-".
func setupChecksumMeta(t *testing.T, configMaps, secrets []string) *metadata.Service {
	t.Helper()
	meta := metadata.New(config.Config{ChartName: "my-app"})
	meta.Load(internal.GenerateObj(checksumDeployYaml))
	for _, name := range configMaps {
		meta.Load(internal.GenerateObj(fmt.Sprintf(checksumConfigMapYaml, name)))
	}
	for _, name := range secrets {
		meta.Load(internal.GenerateObj(fmt.Sprintf(checksumSecretYaml, name)))
	}
	return meta
}
