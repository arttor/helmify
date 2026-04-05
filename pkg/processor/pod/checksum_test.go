package pod

import (
	"fmt"
	"strings"
	"testing"

	"github.com/arttor/helmify/internal"
	"github.com/arttor/helmify/pkg/config"
	"github.com/arttor/helmify/pkg/metadata"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
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

func TestChecksumAnnotations(t *testing.T) {
	t.Run("no references", func(t *testing.T) {
		meta := metadata.New(config.Config{})
		spec := corev1.PodSpec{
			Containers: []corev1.Container{
				{Name: "app", Image: "nginx:latest"},
			},
		}
		result := ChecksumAnnotations(meta, spec, nil, nil)
		assert.Equal(t, "", result)
	})

	t.Run("configmap via envFrom", func(t *testing.T) {
		meta := setupChecksumMeta(t, []string{"my-app-config"}, nil)
		cmFiles := map[string]string{"my-app-config": "config.yaml"}
		spec := corev1.PodSpec{
			Containers: []corev1.Container{
				{
					Name:  "app",
					Image: "nginx:latest",
					EnvFrom: []corev1.EnvFromSource{
						{ConfigMapRef: &corev1.ConfigMapEnvSource{
							LocalObjectReference: corev1.LocalObjectReference{Name: "my-app-config"},
						}},
					},
				},
			},
		}
		result := ChecksumAnnotations(meta, spec, cmFiles, nil)
		assert.Contains(t, result, "checksum/configmap/config:")
		assert.Contains(t, result, `{{ include (print $.Template.BasePath "/config.yaml") . | sha256sum }}`)
	})

	t.Run("secret via envFrom", func(t *testing.T) {
		meta := setupChecksumMeta(t, nil, []string{"my-app-secret"})
		secFiles := map[string]string{"my-app-secret": "secret.yaml"}
		spec := corev1.PodSpec{
			Containers: []corev1.Container{
				{
					Name:  "app",
					Image: "nginx:latest",
					EnvFrom: []corev1.EnvFromSource{
						{SecretRef: &corev1.SecretEnvSource{
							LocalObjectReference: corev1.LocalObjectReference{Name: "my-app-secret"},
						}},
					},
				},
			},
		}
		result := ChecksumAnnotations(meta, spec, nil, secFiles)
		assert.Contains(t, result, "checksum/secret/secret:")
		assert.Contains(t, result, `{{ include (print $.Template.BasePath "/secret.yaml") . | sha256sum }}`)
	})

	t.Run("configmap via env valueFrom", func(t *testing.T) {
		meta := setupChecksumMeta(t, []string{"my-app-config"}, nil)
		cmFiles := map[string]string{"my-app-config": "config.yaml"}
		spec := corev1.PodSpec{
			Containers: []corev1.Container{
				{
					Name:  "app",
					Image: "nginx:latest",
					Env: []corev1.EnvVar{
						{
							Name: "MY_VAR",
							ValueFrom: &corev1.EnvVarSource{
								ConfigMapKeyRef: &corev1.ConfigMapKeySelector{
									LocalObjectReference: corev1.LocalObjectReference{Name: "my-app-config"},
									Key:                  "key1",
								},
							},
						},
					},
				},
			},
		}
		result := ChecksumAnnotations(meta, spec, cmFiles, nil)
		assert.Contains(t, result, "checksum/configmap/config:")
	})

	t.Run("secret via env valueFrom", func(t *testing.T) {
		meta := setupChecksumMeta(t, nil, []string{"my-app-secret"})
		secFiles := map[string]string{"my-app-secret": "secret.yaml"}
		spec := corev1.PodSpec{
			Containers: []corev1.Container{
				{
					Name:  "app",
					Image: "nginx:latest",
					Env: []corev1.EnvVar{
						{
							Name: "MY_SECRET",
							ValueFrom: &corev1.EnvVarSource{
								SecretKeyRef: &corev1.SecretKeySelector{
									LocalObjectReference: corev1.LocalObjectReference{Name: "my-app-secret"},
									Key:                  "password",
								},
							},
						},
					},
				},
			},
		}
		result := ChecksumAnnotations(meta, spec, nil, secFiles)
		assert.Contains(t, result, "checksum/secret/secret:")
	})

	t.Run("configmap via volume", func(t *testing.T) {
		meta := setupChecksumMeta(t, []string{"my-app-config"}, nil)
		cmFiles := map[string]string{"my-app-config": "config.yaml"}
		spec := corev1.PodSpec{
			Containers: []corev1.Container{
				{Name: "app", Image: "nginx:latest"},
			},
			Volumes: []corev1.Volume{
				{
					Name: "config-vol",
					VolumeSource: corev1.VolumeSource{
						ConfigMap: &corev1.ConfigMapVolumeSource{
							LocalObjectReference: corev1.LocalObjectReference{Name: "my-app-config"},
						},
					},
				},
			},
		}
		result := ChecksumAnnotations(meta, spec, cmFiles, nil)
		assert.Contains(t, result, "checksum/configmap/config:")
	})

	t.Run("secret via volume", func(t *testing.T) {
		meta := setupChecksumMeta(t, nil, []string{"my-app-secret"})
		secFiles := map[string]string{"my-app-secret": "secret.yaml"}
		spec := corev1.PodSpec{
			Containers: []corev1.Container{
				{Name: "app", Image: "nginx:latest"},
			},
			Volumes: []corev1.Volume{
				{
					Name: "secret-vol",
					VolumeSource: corev1.VolumeSource{
						Secret: &corev1.SecretVolumeSource{
							SecretName: "my-app-secret",
						},
					},
				},
			},
		}
		result := ChecksumAnnotations(meta, spec, nil, secFiles)
		assert.Contains(t, result, "checksum/secret/secret:")
	})

	t.Run("projected volume with configmap and secret", func(t *testing.T) {
		meta := setupChecksumMeta(t, []string{"my-app-config"}, []string{"my-app-secret"})
		cmFiles := map[string]string{"my-app-config": "config.yaml"}
		secFiles := map[string]string{"my-app-secret": "secret.yaml"}
		spec := corev1.PodSpec{
			Containers: []corev1.Container{
				{Name: "app", Image: "nginx:latest"},
			},
			Volumes: []corev1.Volume{
				{
					Name: "projected-vol",
					VolumeSource: corev1.VolumeSource{
						Projected: &corev1.ProjectedVolumeSource{
							Sources: []corev1.VolumeProjection{
								{ConfigMap: &corev1.ConfigMapProjection{
									LocalObjectReference: corev1.LocalObjectReference{Name: "my-app-config"},
								}},
								{Secret: &corev1.SecretProjection{
									LocalObjectReference: corev1.LocalObjectReference{Name: "my-app-secret"},
								}},
							},
						},
					},
				},
			},
		}
		result := ChecksumAnnotations(meta, spec, cmFiles, secFiles)
		assert.Contains(t, result, "checksum/configmap/config:")
		assert.Contains(t, result, "checksum/secret/secret:")
	})

	t.Run("external configmap skipped", func(t *testing.T) {
		meta := setupChecksumMeta(t, []string{"my-app-config"}, nil)
		cmFiles := map[string]string{"my-app-config": "config.yaml"}
		spec := corev1.PodSpec{
			Containers: []corev1.Container{
				{
					Name:  "app",
					Image: "nginx:latest",
					EnvFrom: []corev1.EnvFromSource{
						{ConfigMapRef: &corev1.ConfigMapEnvSource{
							LocalObjectReference: corev1.LocalObjectReference{Name: "external-config"},
						}},
						{ConfigMapRef: &corev1.ConfigMapEnvSource{
							LocalObjectReference: corev1.LocalObjectReference{Name: "my-app-config"},
						}},
					},
				},
			},
		}
		result := ChecksumAnnotations(meta, spec, cmFiles, nil)
		assert.Contains(t, result, "checksum/configmap/config:")
		assert.NotContains(t, result, "external-config")
	})

	t.Run("initContainers references", func(t *testing.T) {
		meta := setupChecksumMeta(t, []string{"my-app-config"}, nil)
		cmFiles := map[string]string{"my-app-config": "config.yaml"}
		spec := corev1.PodSpec{
			Containers: []corev1.Container{
				{Name: "app", Image: "nginx:latest"},
			},
			InitContainers: []corev1.Container{
				{
					Name:  "init",
					Image: "busybox:latest",
					EnvFrom: []corev1.EnvFromSource{
						{ConfigMapRef: &corev1.ConfigMapEnvSource{
							LocalObjectReference: corev1.LocalObjectReference{Name: "my-app-config"},
						}},
					},
				},
			},
		}
		result := ChecksumAnnotations(meta, spec, cmFiles, nil)
		assert.Contains(t, result, "checksum/configmap/config:")
	})

	t.Run("multiple configmaps and secrets sorted", func(t *testing.T) {
		meta := setupChecksumMeta(t, []string{"my-app-config-a", "my-app-config-b"}, []string{"my-app-secret-x"})
		cmFiles := map[string]string{
			"my-app-config-a": "config-a.yaml",
			"my-app-config-b": "config-b.yaml",
		}
		secFiles := map[string]string{"my-app-secret-x": "secret-x.yaml"}
		spec := corev1.PodSpec{
			Containers: []corev1.Container{
				{
					Name:  "app",
					Image: "nginx:latest",
					EnvFrom: []corev1.EnvFromSource{
						{ConfigMapRef: &corev1.ConfigMapEnvSource{
							LocalObjectReference: corev1.LocalObjectReference{Name: "my-app-config-b"},
						}},
						{ConfigMapRef: &corev1.ConfigMapEnvSource{
							LocalObjectReference: corev1.LocalObjectReference{Name: "my-app-config-a"},
						}},
						{SecretRef: &corev1.SecretEnvSource{
							LocalObjectReference: corev1.LocalObjectReference{Name: "my-app-secret-x"},
						}},
					},
				},
			},
		}
		result := ChecksumAnnotations(meta, spec, cmFiles, secFiles)
		assert.Contains(t, result, "checksum/configmap/config-a:")
		assert.Contains(t, result, "checksum/configmap/config-b:")
		assert.Contains(t, result, "checksum/secret/secret-x:")
	})

	t.Run("nil metadata maps safe", func(t *testing.T) {
		meta := &metadata.Service{}
		spec := corev1.PodSpec{
			Containers: []corev1.Container{
				{
					Name:  "app",
					Image: "nginx:latest",
					EnvFrom: []corev1.EnvFromSource{
						{ConfigMapRef: &corev1.ConfigMapEnvSource{
							LocalObjectReference: corev1.LocalObjectReference{Name: "some-config"},
						}},
					},
				},
			},
		}
		result := ChecksumAnnotations(meta, spec, nil, nil)
		assert.Equal(t, "", result)
	})

	t.Run("deduplicates same configmap from multiple sources", func(t *testing.T) {
		meta := setupChecksumMeta(t, []string{"my-app-config"}, nil)
		cmFiles := map[string]string{"my-app-config": "config.yaml"}
		spec := corev1.PodSpec{
			Containers: []corev1.Container{
				{
					Name:  "app",
					Image: "nginx:latest",
					EnvFrom: []corev1.EnvFromSource{
						{ConfigMapRef: &corev1.ConfigMapEnvSource{
							LocalObjectReference: corev1.LocalObjectReference{Name: "my-app-config"},
						}},
					},
				},
			},
			Volumes: []corev1.Volume{
				{
					Name: "config-vol",
					VolumeSource: corev1.VolumeSource{
						ConfigMap: &corev1.ConfigMapVolumeSource{
							LocalObjectReference: corev1.LocalObjectReference{Name: "my-app-config"},
						},
					},
				},
			},
		}
		result := ChecksumAnnotations(meta, spec, cmFiles, nil)
		assert.Equal(t, `checksum/configmap/config: {{ include (print $.Template.BasePath "/config.yaml") . | sha256sum }}`, result)
	})

	t.Run("no collision when configmap and secret have same trimmed name", func(t *testing.T) {
		meta := setupChecksumMeta(t, []string{"my-app-credentials"}, []string{"my-app-credentials"})
		cmFiles := map[string]string{"my-app-credentials": "credentials.yaml"}
		secFiles := map[string]string{"my-app-credentials": "credentials.yaml"}
		spec := corev1.PodSpec{
			Containers: []corev1.Container{
				{
					Name:  "app",
					Image: "nginx:latest",
					EnvFrom: []corev1.EnvFromSource{
						{ConfigMapRef: &corev1.ConfigMapEnvSource{
							LocalObjectReference: corev1.LocalObjectReference{Name: "my-app-credentials"},
						}},
						{SecretRef: &corev1.SecretEnvSource{
							LocalObjectReference: corev1.LocalObjectReference{Name: "my-app-credentials"},
						}},
					},
				},
			},
		}
		result := ChecksumAnnotations(meta, spec, cmFiles, secFiles)
		assert.Contains(t, result, "checksum/configmap/credentials:")
		assert.Contains(t, result, "checksum/secret/credentials:")
		assert.Equal(t, 2, len(strings.Split(result, "\n")))
	})

	t.Run("uses actual filename not trimmed name for path", func(t *testing.T) {
		meta := setupChecksumMeta(t, []string{"my-app-config"}, nil)
		// Simulate all resources in a single input file
		cmFiles := map[string]string{"my-app-config": "input.yaml"}
		spec := corev1.PodSpec{
			Containers: []corev1.Container{
				{
					Name:  "app",
					Image: "nginx:latest",
					EnvFrom: []corev1.EnvFromSource{
						{ConfigMapRef: &corev1.ConfigMapEnvSource{
							LocalObjectReference: corev1.LocalObjectReference{Name: "my-app-config"},
						}},
					},
				},
			},
		}
		result := ChecksumAnnotations(meta, spec, cmFiles, nil)
		assert.Contains(t, result, `/input.yaml")`)
		assert.NotContains(t, result, `/config.yaml")`)
	})
}

// setupChecksumMeta creates a metadata.Service with the given configmaps and secrets loaded,
// plus a deployment to establish a common prefix of "my-app-".
func setupChecksumMeta(t *testing.T, configMaps, secrets []string) *metadata.Service {
	t.Helper()
	meta := metadata.New(config.Config{ChartName: "my-app"})
	// Load a deployment to establish common prefix "my-app-"
	meta.Load(internal.GenerateObj(checksumDeployYaml))
	for _, name := range configMaps {
		meta.Load(internal.GenerateObj(fmt.Sprintf(checksumConfigMapYaml, name)))
	}
	for _, name := range secrets {
		meta.Load(internal.GenerateObj(fmt.Sprintf(checksumSecretYaml, name)))
	}
	return meta
}
