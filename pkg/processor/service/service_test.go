package service

import (
	"testing"

	"github.com/arttor/helmify/internal"
	"github.com/arttor/helmify/pkg/metadata"
	"github.com/stretchr/testify/assert"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

const svcYaml = `apiVersion: v1
kind: Service
metadata:
  labels:
    control-plane: controller-manager
  name: my-operator-controller-manager-metrics-service
  namespace: my-operator-system
spec:
  ports:
  - name: https
    port: 8443
    targetPort: https
  selector:
    control-plane: controller-manager`

const svcWithIPFamilyYaml = `apiVersion: v1
kind: Service
metadata:
  labels:
    control-plane: controller-manager
  name: my-operator-controller-manager-metrics-service
  namespace: my-operator-system
spec:
  ipFamilyPolicy: PreferDualStack
  ipFamilies:
  - IPv4
  - IPv6
  ports:
  - name: https
    port: 8443
    targetPort: https
  selector:
    control-plane: controller-manager`

func Test_svc_Process(t *testing.T) {
	var testInstance svc

	t.Run("processed", func(t *testing.T) {
		obj := internal.GenerateObj(svcYaml)
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

	t.Run("processed with IP family", func(t *testing.T) {
		obj := internal.GenerateObj(svcWithIPFamilyYaml)
		processed, template, err := testInstance.Process(&metadata.Service{}, obj)
		assert.NoError(t, err)
		assert.Equal(t, true, processed)
		assert.NotNil(t, template)

		values := template.Values()
		ipFamilyPolicy, found, err := unstructured.NestedString(values, "myOperatorControllerManagerMetricsService", "ipFamilyPolicy")
		assert.NoError(t, err)
		assert.True(t, found)
		assert.Equal(t, "PreferDualStack", ipFamilyPolicy)

		ipFamilies, found, err := unstructured.NestedSlice(values, "myOperatorControllerManagerMetricsService", "ipFamilies")
		assert.NoError(t, err)
		assert.True(t, found)
		assert.Len(t, ipFamilies, 2)
	})
}
