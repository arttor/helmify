package webhook

import (
	"testing"

	"github.com/arttor/helmify/pkg/metadata"

	"github.com/arttor/helmify/internal"
	"github.com/stretchr/testify/assert"
)

const mwhYaml = `apiVersion: admissionregistration.k8s.io/v1
kind: MutatingWebhookConfiguration
metadata:
  annotations:
    cert-manager.io/inject-ca-from: my-operator-system/my-operator-serving-cert
  name: my-operator-mutating-webhook-configuration
webhooks:
- admissionReviewVersions:
  - v1
  - v1beta1
  clientConfig:
    service:
      name: my-operator-webhook-service
      namespace: my-operator-system
      path: /mutate-ceph-example-com-v1alpha1-volume
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

func Test_mwh_Process(t *testing.T) {
	var testInstance mwh

	t.Run("processed", func(t *testing.T) {
		obj := internal.GenerateObj(mwhYaml)
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
