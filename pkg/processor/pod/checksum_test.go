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
	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/runtime"
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

// deploymentWithSpec builds a Deployment YAML with various ConfigMap/Secret references.
func deploymentWithSpec(envFromCMs, envFromSecrets, envCMKeyRefs, envSecretKeyRefs, volumeCMs, volumeSecrets []string) corev1.PodSpec {
	var envFrom []corev1.EnvFromSource
	for _, name := range envFromCMs {
		envFrom = append(envFrom, corev1.EnvFromSource{
			ConfigMapRef: &corev1.ConfigMapEnvSource{
				LocalObjectReference: corev1.LocalObjectReference{Name: name},
			},
		})
	}
	for _, name := range envFromSecrets {
		envFrom = append(envFrom, corev1.EnvFromSource{
			SecretRef: &corev1.SecretEnvSource{
				LocalObjectReference: corev1.LocalObjectReference{Name: name},
			},
		})
	}

	var env []corev1.EnvVar
	for _, name := range envCMKeyRefs {
		env = append(env, corev1.EnvVar{
			Name: "VAR",
			ValueFrom: &corev1.EnvVarSource{
				ConfigMapKeyRef: &corev1.ConfigMapKeySelector{
					LocalObjectReference: corev1.LocalObjectReference{Name: name},
					Key:                  "key1",
				},
			},
		})
	}
	for _, name := range envSecretKeyRefs {
		env = append(env, corev1.EnvVar{
			Name: "VAR",
			ValueFrom: &corev1.EnvVarSource{
				SecretKeyRef: &corev1.SecretKeySelector{
					LocalObjectReference: corev1.LocalObjectReference{Name: name},
					Key:                  "key1",
				},
			},
		})
	}

	var volumes []corev1.Volume
	for _, name := range volumeCMs {
		volumes = append(volumes, corev1.Volume{
			Name: "vol",
			VolumeSource: corev1.VolumeSource{
				ConfigMap: &corev1.ConfigMapVolumeSource{
					LocalObjectReference: corev1.LocalObjectReference{Name: name},
				},
			},
		})
	}
	for _, name := range volumeSecrets {
		volumes = append(volumes, corev1.Volume{
			Name: "vol",
			VolumeSource: corev1.VolumeSource{
				Secret: &corev1.SecretVolumeSource{SecretName: name},
			},
		})
	}

	return corev1.PodSpec{
		Containers: []corev1.Container{
			{Name: "app", Image: "nginx:latest", EnvFrom: envFrom, Env: env},
		},
		Volumes: volumes,
	}
}

func TestChecksumAnnotations(t *testing.T) {
	t.Run("no references", func(t *testing.T) {
		meta := metadata.New(config.Config{})
		spec := corev1.PodSpec{
			Containers: []corev1.Container{{Name: "app", Image: "nginx:latest"}},
		}
		result := ChecksumAnnotations(meta, spec, nil, nil, 6)
		assert.Equal(t, "", result)
	})

	t.Run("configmap via envFrom", func(t *testing.T) {
		meta := setupChecksumMeta(t, []string{"my-app-config"}, nil)
		spec := deploymentWithSpec([]string{"my-app-config"}, nil, nil, nil, nil, nil)
		cmFiles := map[string]string{"my-app-config": "config.yaml"}
		result := ChecksumAnnotations(meta, spec, cmFiles, nil, 6)
		assert.Contains(t, result, "      annotations:")
		assert.Contains(t, result, `checksum/configmap/config: {{ include (print $.Template.BasePath "/config.yaml") . | sha256sum }}`)
	})

	t.Run("secret via envFrom", func(t *testing.T) {
		meta := setupChecksumMeta(t, nil, []string{"my-app-secret"})
		spec := deploymentWithSpec(nil, []string{"my-app-secret"}, nil, nil, nil, nil)
		secFiles := map[string]string{"my-app-secret": "secret.yaml"}
		result := ChecksumAnnotations(meta, spec, nil, secFiles, 6)
		assert.Contains(t, result, "checksum/secret/secret:")
		assert.Contains(t, result, `/secret.yaml")`)
	})

	t.Run("configmap via env valueFrom", func(t *testing.T) {
		meta := setupChecksumMeta(t, []string{"my-app-config"}, nil)
		spec := deploymentWithSpec(nil, nil, []string{"my-app-config"}, nil, nil, nil)
		cmFiles := map[string]string{"my-app-config": "config.yaml"}
		result := ChecksumAnnotations(meta, spec, cmFiles, nil, 6)
		assert.Contains(t, result, "checksum/configmap/config:")
	})

	t.Run("secret via env valueFrom", func(t *testing.T) {
		meta := setupChecksumMeta(t, nil, []string{"my-app-secret"})
		spec := deploymentWithSpec(nil, nil, nil, []string{"my-app-secret"}, nil, nil)
		secFiles := map[string]string{"my-app-secret": "secret.yaml"}
		result := ChecksumAnnotations(meta, spec, nil, secFiles, 6)
		assert.Contains(t, result, "checksum/secret/secret:")
	})

	t.Run("configmap via volume", func(t *testing.T) {
		meta := setupChecksumMeta(t, []string{"my-app-config"}, nil)
		spec := deploymentWithSpec(nil, nil, nil, nil, []string{"my-app-config"}, nil)
		cmFiles := map[string]string{"my-app-config": "config.yaml"}
		result := ChecksumAnnotations(meta, spec, cmFiles, nil, 6)
		assert.Contains(t, result, "checksum/configmap/config:")
	})

	t.Run("secret via volume", func(t *testing.T) {
		meta := setupChecksumMeta(t, nil, []string{"my-app-secret"})
		spec := deploymentWithSpec(nil, nil, nil, nil, nil, []string{"my-app-secret"})
		secFiles := map[string]string{"my-app-secret": "secret.yaml"}
		result := ChecksumAnnotations(meta, spec, nil, secFiles, 6)
		assert.Contains(t, result, "checksum/secret/secret:")
	})

	t.Run("projected volume", func(t *testing.T) {
		meta := setupChecksumMeta(t, []string{"my-app-config"}, []string{"my-app-secret"})
		spec := corev1.PodSpec{
			Containers: []corev1.Container{{Name: "app", Image: "nginx:latest"}},
			Volumes: []corev1.Volume{
				{
					Name: "projected",
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
		cmFiles := map[string]string{"my-app-config": "config.yaml"}
		secFiles := map[string]string{"my-app-secret": "secret.yaml"}
		result := ChecksumAnnotations(meta, spec, cmFiles, secFiles, 6)
		assert.Contains(t, result, "checksum/configmap/config:")
		assert.Contains(t, result, "checksum/secret/secret:")
	})

	t.Run("external configmap skipped", func(t *testing.T) {
		meta := setupChecksumMeta(t, []string{"my-app-config"}, nil)
		spec := deploymentWithSpec([]string{"external-config", "my-app-config"}, nil, nil, nil, nil, nil)
		cmFiles := map[string]string{"my-app-config": "config.yaml"}
		result := ChecksumAnnotations(meta, spec, cmFiles, nil, 6)
		assert.Contains(t, result, "checksum/configmap/config:")
		assert.NotContains(t, result, "external-config")
	})

	t.Run("multiple configmaps and secrets sorted", func(t *testing.T) {
		meta := setupChecksumMeta(t, []string{"my-app-config-a", "my-app-config-b"}, []string{"my-app-secret-x"})
		spec := deploymentWithSpec(
			[]string{"my-app-config-b", "my-app-config-a"},
			[]string{"my-app-secret-x"},
			nil, nil, nil, nil,
		)
		cmFiles := map[string]string{
			"my-app-config-a": "config-a.yaml",
			"my-app-config-b": "config-b.yaml",
		}
		secFiles := map[string]string{"my-app-secret-x": "secret-x.yaml"}
		result := ChecksumAnnotations(meta, spec, cmFiles, secFiles, 6)
		assert.Contains(t, result, "checksum/configmap/config-a:")
		assert.Contains(t, result, "checksum/configmap/config-b:")
		assert.Contains(t, result, "checksum/secret/secret-x:")
	})

	t.Run("nil metadata maps safe", func(t *testing.T) {
		meta := &metadata.Service{}
		spec := deploymentWithSpec([]string{"some-config"}, nil, nil, nil, nil, nil)
		result := ChecksumAnnotations(meta, spec, nil, nil, 6)
		assert.Equal(t, "", result)
	})

	t.Run("deduplicates same configmap from multiple sources", func(t *testing.T) {
		meta := setupChecksumMeta(t, []string{"my-app-config"}, nil)
		spec := deploymentWithSpec([]string{"my-app-config"}, nil, nil, nil, []string{"my-app-config"}, nil)
		cmFiles := map[string]string{"my-app-config": "config.yaml"}
		result := ChecksumAnnotations(meta, spec, cmFiles, nil, 6)
		assert.Equal(t, 1, strings.Count(result, "checksum/"))
	})

	t.Run("uses actual filename for path", func(t *testing.T) {
		meta := setupChecksumMeta(t, []string{"my-app-config"}, nil)
		spec := deploymentWithSpec([]string{"my-app-config"}, nil, nil, nil, nil, nil)
		cmFiles := map[string]string{"my-app-config": "input.yaml"}
		result := ChecksumAnnotations(meta, spec, cmFiles, nil, 6)
		assert.Contains(t, result, `/input.yaml")`)
	})

	t.Run("initContainers references", func(t *testing.T) {
		meta := setupChecksumMeta(t, []string{"my-app-config"}, nil)
		spec := corev1.PodSpec{
			Containers: []corev1.Container{{Name: "app", Image: "nginx:latest"}},
			InitContainers: []corev1.Container{
				{
					Name: "init", Image: "busybox:latest",
					EnvFrom: []corev1.EnvFromSource{
						{ConfigMapRef: &corev1.ConfigMapEnvSource{
							LocalObjectReference: corev1.LocalObjectReference{Name: "my-app-config"},
						}},
					},
				},
			},
		}
		cmFiles := map[string]string{"my-app-config": "config.yaml"}
		result := ChecksumAnnotations(meta, spec, cmFiles, nil, 6)
		assert.Contains(t, result, "checksum/configmap/config:")
	})

	t.Run("indent parameter controls output", func(t *testing.T) {
		meta := setupChecksumMeta(t, []string{"my-app-config"}, nil)
		spec := deploymentWithSpec([]string{"my-app-config"}, nil, nil, nil, nil, nil)
		cmFiles := map[string]string{"my-app-config": "config.yaml"}

		result6 := ChecksumAnnotations(meta, spec, cmFiles, nil, 6)
		assert.True(t, strings.HasPrefix(result6, "      annotations:"))

		result4 := ChecksumAnnotations(meta, spec, cmFiles, nil, 4)
		assert.True(t, strings.HasPrefix(result4, "    annotations:"))
	})
}

func TestChecksumAnnotations_Integration(t *testing.T) {
	t.Run("deployment processor picks up checksums", func(t *testing.T) {
		deplYaml := `apiVersion: apps/v1
kind: Deployment
metadata:
  name: my-app-web
  namespace: my-app-system
spec:
  selector:
    matchLabels:
      app: web
  template:
    metadata:
      labels:
        app: web
    spec:
      containers:
      - name: web
        image: nginx:latest
        envFrom:
        - configMapRef:
            name: my-app-config`

		meta := metadata.New(config.Config{ChartName: "my-app", AddChecksumAnnotations: true})
		meta.Load(internal.GenerateObj(fmt.Sprintf(checksumConfigMapYaml, "my-app-config")))
		meta.Load(internal.GenerateObj(deplYaml))
		meta.SetConfigMapFiles(map[string]string{"my-app-config": "config.yaml"})

		obj := internal.GenerateObj(deplYaml)
		var depl appsv1.Deployment
		err := runtime.DefaultUnstructuredConverter.FromUnstructured(obj.Object, &depl)
		assert.NoError(t, err)

		checksumAnns := ChecksumAnnotations(meta, depl.Spec.Template.Spec, meta.ConfigMapFiles(), meta.SecretFiles(), 6)
		assert.Contains(t, checksumAnns, "annotations:")
		assert.Contains(t, checksumAnns, "checksum/configmap/config:")
		assert.Contains(t, checksumAnns, `/config.yaml")`)
	})
}

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
