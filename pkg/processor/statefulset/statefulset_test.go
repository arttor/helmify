package statefulset

import (
	"testing"

	"github.com/arttor/helmify/internal"
	"github.com/arttor/helmify/pkg/metadata"
	"github.com/stretchr/testify/assert"
)

const strStatefulSet = `apiVersion: apps/v1
kind: StatefulSet
metadata:
  name: my-app-web
  namespace: my-app-system
spec:
  serviceName: my-app-web
  replicas: 3
  selector:
    matchLabels:
      app: my-app-web
  template:
    metadata:
      labels:
        app: my-app-web
    spec:
      containers:
      - name: web
        image: my-app/web:v1.0.0
        envFrom:
        - configMapRef:
            name: my-app-web-config
        env:
        - name: SECRET_KEY
          valueFrom:
            secretKeyRef:
              name: my-app-web-secret
              key: secret-key
`

func Test_statefulset_Process(t *testing.T) {
	var testInstance statefulset

	t.Run("processed", func(t *testing.T) {
		obj := internal.GenerateObj(strStatefulSet)
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
