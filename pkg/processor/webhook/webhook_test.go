package webhook

import (
	"github.com/arttor/helmify/internal"
	"github.com/arttor/helmify/pkg/helmify"
	"github.com/stretchr/testify/assert"
	"testing"
)

const whYaml = `apiVersion: admissionregistration.k8s.io/v1
kind: ValidatingWebhookConfiguration
metadata:
  annotations:
    cert-manager.io/inject-ca-from: my-operator-system/my-operator-serving-cert
  name: my-operator-validating-webhook-configuration
webhooks:
- admissionReviewVersions:
  - v1
  - v1beta1
  clientConfig:
    service:
      name: my-operator-webhook-service
      namespace: my-operator-system
      path: /validate-ceph-example-com-v1alpha1-volume
  failurePolicy: Fail
  name: vvolume.kb.io
  rules:
  - apiGroups:
    - test.example.com
    apiVersions:
    - v1alpha1
    operations:
    - CREATE
    - UPDATE
    resources:
    - volumes
  sideEffects: None`

func Test_wh_Process(t *testing.T) {
	var testInstance wh

	t.Run("processed", func(t *testing.T) {
		obj := internal.GenerateObj(whYaml)
		processed, _, err := testInstance.Process(helmify.ChartInfo{}, obj)
		assert.NoError(t, err)
		assert.Equal(t, true, processed)
	})
	t.Run("skipped", func(t *testing.T) {
		obj := internal.TestNs
		processed, _, err := testInstance.Process(helmify.ChartInfo{}, obj)
		assert.NoError(t, err)
		assert.Equal(t, false, processed)
	})
}
