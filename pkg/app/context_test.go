package app

import (
	"testing"

	"github.com/arttor/helmify/internal"
	"github.com/arttor/helmify/pkg/config"
	"github.com/stretchr/testify/assert"
)

func Test_precomputeConfigFileNames(t *testing.T) {
	t.Run("maps configmaps and secrets with input filenames", func(t *testing.T) {
		conf := config.Config{ChartName: "my-app", AddChecksumAnnotations: true}
		ctx := New(conf, nil)

		cmObj := internal.GenerateObj(`apiVersion: v1
kind: ConfigMap
metadata:
  name: my-app-config`)
		secObj := internal.GenerateObj(`apiVersion: v1
kind: Secret
metadata:
  name: my-app-secret`)
		deplObj := internal.GenerateObj(`apiVersion: apps/v1
kind: Deployment
metadata:
  name: my-app-web`)

		ctx.Add(cmObj, "configmap.yaml")
		ctx.Add(secObj, "secret.yaml")
		ctx.Add(deplObj, "deployment.yaml")

		ctx.precomputeConfigFileNames()

		assert.Equal(t, "configmap.yaml", ctx.appMeta.ConfigMapFiles()["my-app-config"])
		assert.Equal(t, "secret.yaml", ctx.appMeta.SecretFiles()["my-app-secret"])
		assert.Empty(t, ctx.appMeta.ConfigMapFiles()["my-app-web"])
	})

	t.Run("uses trimmed name for stdin input", func(t *testing.T) {
		conf := config.Config{ChartName: "my-app", AddChecksumAnnotations: true}
		ctx := New(conf, nil)

		cmObj := internal.GenerateObj(`apiVersion: v1
kind: ConfigMap
metadata:
  name: my-app-config`)
		secObj := internal.GenerateObj(`apiVersion: v1
kind: Secret
metadata:
  name: my-app-secret`)

		ctx.Add(cmObj, "") // empty filename = stdin
		ctx.Add(secObj, "")

		ctx.precomputeConfigFileNames()

		assert.Equal(t, "config.yaml", ctx.appMeta.ConfigMapFiles()["my-app-config"])
		assert.Equal(t, "secret.yaml", ctx.appMeta.SecretFiles()["my-app-secret"])
	})
}
