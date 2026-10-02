package daemonset

import (
	"bytes"
	"testing"

	"github.com/arttor/helmify/pkg/helmify"
	"github.com/arttor/helmify/pkg/metadata"

	"github.com/arttor/helmify/internal"
	"github.com/stretchr/testify/assert"
)

const (
	strDepl = `apiVersion: apps/v1
kind: DaemonSet
metadata:
  name: fluentd-elasticsearch
  namespace: kube-system
  labels:
    k8s-app: fluentd-logging
spec:
  selector:
    matchLabels:
      name: fluentd-elasticsearch
  template:
    metadata:
      labels:
        name: fluentd-elasticsearch
    spec:
      tolerations:
      # this toleration is to have the daemonset runnable on master nodes
      # remove it if your masters can't run pods
      - key: node-role.kubernetes.io/master
        operator: Exists
        effect: NoSchedule
      containers:
      - name: fluentd-elasticsearch
        image: quay.io/fluentd_elasticsearch/fluentd:v2.5.2
        resources:
          limits:
            memory: 200Mi
          requests:
            cpu: 100m
            memory: 200Mi
        volumeMounts:
        - name: varlog
          mountPath: /var/log
        - name: varlibdockercontainers
          mountPath: /var/lib/docker/containers
          readOnly: true
      terminationGracePeriodSeconds: 30
      volumes:
      - name: varlog
        hostPath:
          path: /var/log
      - name: varlibdockercontainers
        hostPath:
          path: /var/lib/docker/containers
`
	strDaeWithRollingUpdate = `apiVersion: apps/v1
kind: DaemonSet
metadata:
  name: fluentd-elasticsearch
  namespace: kube-system
  labels:
    k8s-app: fluentd-logging
spec:
  updateStrategy:
    type: RollingUpdate
    rollingUpdate:
      maxUnavailable: 20%
      maxSurge: 1
  selector:
    matchLabels:
      name: fluentd-elasticsearch
  template:
    metadata:
      labels:
        name: fluentd-elasticsearch
    spec:
      containers:
      - name: fluentd-elasticsearch
        image: quay.io/fluentd_elasticsearch/fluentd:v2.5.2
`
	strDaeWithOnDelete = `apiVersion: apps/v1
kind: DaemonSet
metadata:
  name: fluentd-elasticsearch
  namespace: kube-system
  labels:
    k8s-app: fluentd-logging
spec:
  updateStrategy:
    type: OnDelete
  selector:
    matchLabels:
      name: fluentd-elasticsearch
  template:
    metadata:
      labels:
        name: fluentd-elasticsearch
    spec:
      containers:
      - name: fluentd-elasticsearch
        image: quay.io/fluentd_elasticsearch/fluentd:v2.5.2
`
)

func Test_daemonset_Process(t *testing.T) {
	var testInstance daemonset

	t.Run("processed", func(t *testing.T) {
		obj := internal.GenerateObj(strDepl)
		processed, res, err := testInstance.Process(&metadata.Service{}, obj)
		assert.NoError(t, err)
		assert.Equal(t, true, processed)

		var buf bytes.Buffer
		err = res.Write(&buf)
		assert.NoError(t, err)
		assert.NotContains(t, buf.String(), "updateStrategy")
	})
	t.Run("processed_with_rolling_update_strategy", func(t *testing.T) {
		obj := internal.GenerateObj(strDaeWithRollingUpdate)
		processed, res, err := testInstance.Process(&metadata.Service{}, obj)
		assert.NoError(t, err)
		assert.Equal(t, true, processed)

		var buf bytes.Buffer
		err = res.Write(&buf)
		assert.NoError(t, err)

		expectedTemplate := `spec:
  updateStrategy:
    rollingUpdate:
      maxSurge: {{ .Values.fluentdElasticsearch.updateStrategy.rollingUpdate.maxSurge
        }}
      maxUnavailable: {{ .Values.fluentdElasticsearch.updateStrategy.rollingUpdate.maxUnavailable
        | quote }}
    type: {{ .Values.fluentdElasticsearch.updateStrategy.type | quote }}
  selector:`
		assert.Contains(t, buf.String(), expectedTemplate)

		expectedValues := helmify.Values{
			"fluentdElasticsearch": map[string]interface{}{
				"updateStrategy": map[string]interface{}{
					"type": "RollingUpdate",
					"rollingUpdate": map[string]interface{}{
						"maxUnavailable": "20%",
						"maxSurge":       int64(1),
					},
				},
			},
		}
		values := res.Values()
		assert.Equal(t, "RollingUpdate", values["fluentdElasticsearch"].(map[string]interface{})["updateStrategy"].(map[string]interface{})["type"])
		ru := values["fluentdElasticsearch"].(map[string]interface{})["updateStrategy"].(map[string]interface{})["rollingUpdate"].(map[string]interface{})
		assert.Equal(t, "20%", ru["maxUnavailable"])
		assert.Equal(t, int64(1), ru["maxSurge"])
		_ = expectedValues
	})
	t.Run("processed_with_on_delete_strategy", func(t *testing.T) {
		obj := internal.GenerateObj(strDaeWithOnDelete)
		processed, res, err := testInstance.Process(&metadata.Service{}, obj)
		assert.NoError(t, err)
		assert.Equal(t, true, processed)

		var buf bytes.Buffer
		err = res.Write(&buf)
		assert.NoError(t, err)

		expectedTemplate := `spec:
  updateStrategy:
    type: {{ .Values.fluentdElasticsearch.updateStrategy.type | quote }}
  selector:`
		assert.Contains(t, buf.String(), expectedTemplate)

		values := res.Values()
		assert.Equal(t, "OnDelete", values["fluentdElasticsearch"].(map[string]interface{})["updateStrategy"].(map[string]interface{})["type"])
	})
	t.Run("skipped", func(t *testing.T) {
		obj := internal.TestNs
		processed, _, err := testInstance.Process(&metadata.Service{}, obj)
		assert.NoError(t, err)
		assert.Equal(t, false, processed)
	})
}
