package webhook

import (
	"testing"

	"github.com/arttor/helmify/pkg/metadata"

	"github.com/arttor/helmify/internal"
	"github.com/stretchr/testify/assert"
)

const vwhYaml = `apiVersion: admissionregistration.k8s.io/v1
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
  namespaceSelector:
    matchExpressions:
    - key: kubernetes.io/metadata.name
      operator: NotIn
      values:
      - namespace-1
      - my-operator-system
      - namespace-3
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

func Test_vwh_Process(t *testing.T) {
	var testInstance vwh

	t.Run("processed", func(t *testing.T) {
		obj := internal.GenerateObj(vwhYaml)
		processed, _, err := testInstance.Process(testAppMetaWithNamespace(), obj)
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
