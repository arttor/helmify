package deployment

import (
	"bytes"
	"testing"

	"github.com/arttor/helmify/pkg/metadata"

	"github.com/arttor/helmify/internal"
	"github.com/stretchr/testify/assert"
)

const (
	strDepl = `apiVersion: apps/v1
kind: Deployment
metadata:
  labels:
    control-plane: controller-manager
  name: my-operator-controller-manager
  namespace: my-operator-system
spec:
  revisionHistoryLimit: 5
  replicas: 1
  strategy:
    type: Recreate
  selector:
    matchLabels:
      control-plane: controller-manager
  template:
    metadata:
      labels:
        control-plane: controller-manager
    spec:
      containers:
      - args:
        - --secure-listen-address=0.0.0.0:8443
        - --upstream=http://127.0.0.1:8080/
        - --logtostderr=true
        - --v=10
        image: gcr.io/kubebuilder/kube-rbac-proxy:v0.8.0
        name: kube-rbac-proxy
        ports:
        - containerPort: 8443
          name: https
      - args:
        - --health-probe-bind-address=:8081
        - --metrics-bind-address=127.0.0.1:8080
        - --leader-elect
        command:
        - /manager
        volumeMounts:
        - mountPath: /controller_manager_config.yaml
          name: manager-config
          subPath: controller_manager_config.yaml
        - name: secret-volume
          mountPath: /my.ca
        - name: sample-pv-storage
          mountPath: "/usr/share/nginx/html"
        env:
        - name: VAR1
          valueFrom:
            secretKeyRef:
              name: my-operator-secret-vars
              key: VAR1
        - name: VAR2
          valueFrom:
            configMapKeyRef:
              name: my-operator-configmap-vars
              key: VAR2
        - name: VAR3
          valueFrom:
            fieldRef:
              fieldPath: metadata.namespace
        - name: VAR4
          valueFrom:
            resourceFieldRef:
              resource: limits.cpu
        - name: VAR5
          value: "123"
        - name: VAR6
          valueFrom:
            fieldRef:
              fieldPath: metadata.labels['app.kubernetes.io/something']
        image: controller:latest
        livenessProbe:
          httpGet:
            path: /healthz
            port: 8081
          initialDelaySeconds: 15
          periodSeconds: 20
        name: manager
        readinessProbe:
          httpGet:
            path: /readyz
            port: 8081
          initialDelaySeconds: 5
          periodSeconds: 10
        resources:
          limits:
            cpu: 100m
            memory: 30Mi
          requests:
            cpu: 100m
            memory: 20Mi
        securityContext:
          allowPrivilegeEscalation: false
      securityContext:
        runAsNonRoot: true
      serviceAccountName: my-operator-controller-manager
      terminationGracePeriodSeconds: 10
      volumes:
      - configMap:
          name: my-operator-manager-config
        name: manager-config
      - name: secret-volume
        secret:
          secretName: my-operator-secret-ca
      - name: sample-pv-storage
        persistentVolumeClaim:
          claimName: my-sample-pv-claim
`
)

func Test_deployment_Process(t *testing.T) {
	var testInstance deployment

	t.Run("processed", func(t *testing.T) {
		obj := internal.GenerateObj(strDepl)
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

const (
	// strDeplNoAnnotations has no pod template annotations — tests that podAnnotations is
	// still seeded in values and the values-driven block is present in the template.
	strDeplNoAnnotations = `apiVersion: apps/v1
kind: Deployment
metadata:
  labels:
    control-plane: controller-manager
  name: my-operator-controller-manager
  namespace: my-operator-system
spec:
  replicas: 1
  selector:
    matchLabels:
      control-plane: controller-manager
  template:
    metadata:
      labels:
        control-plane: controller-manager
    spec:
      containers:
      - name: manager
        image: controller:latest
`
	// strDeplWithAnnotations has static pod template annotations — tests that static
	// annotations are preserved and the values-driven block is appended after them.
	strDeplWithAnnotations = `apiVersion: apps/v1
kind: Deployment
metadata:
  labels:
    control-plane: controller-manager
  name: my-operator-controller-manager
  namespace: my-operator-system
spec:
  replicas: 1
  selector:
    matchLabels:
      control-plane: controller-manager
  template:
    metadata:
      labels:
        control-plane: controller-manager
      annotations:
        kubectl.kubernetes.io/default-container: manager
    spec:
      containers:
      - name: manager
        image: controller:latest
`
)

func Test_deployment_podAnnotations(t *testing.T) {
	var testInstance deployment

	t.Run("no static annotations - values seeded and template block present", func(t *testing.T) {
		obj := internal.GenerateObj(strDeplNoAnnotations)
		processed, tmpl, err := testInstance.Process(&metadata.Service{}, obj)
		assert.NoError(t, err)
		assert.True(t, processed)

		// podAnnotations should be seeded as empty map in values
		// the deployment name "my-operator-controller-manager" trims to "controller-manager"
		// which becomes "controllerManager" in lowerCamel
		vals := tmpl.Values()
		controllerManager, ok := vals["myOperatorControllerManager"].(map[string]interface{})
		assert.True(t, ok, "expected myOperatorControllerManager key in values")
		podAnnotations, ok := controllerManager["podAnnotations"]
		assert.True(t, ok, "expected podAnnotations key in values")
		assert.Equal(t, map[string]interface{}{}, podAnnotations)

		// Template output must contain the values-driven annotations block
		var buf bytes.Buffer
		assert.NoError(t, tmpl.Write(&buf))
		output := buf.String()
		assert.Contains(t, output, "{{- with .Values.myOperatorControllerManager.podAnnotations }}")
		assert.Contains(t, output, "{{- toYaml . | nindent 8 }}")
		assert.Contains(t, output, "annotations:")
	})

	t.Run("static annotations preserved and values block appended", func(t *testing.T) {
		obj := internal.GenerateObj(strDeplWithAnnotations)
		processed, tmpl, err := testInstance.Process(&metadata.Service{}, obj)
		assert.NoError(t, err)
		assert.True(t, processed)

		var buf bytes.Buffer
		assert.NoError(t, tmpl.Write(&buf))
		output := buf.String()

		// Static annotation must be in the output
		assert.Contains(t, output, "kubectl.kubernetes.io/default-container: manager")
		// Values-driven block must also be present
		assert.Contains(t, output, "{{- with .Values.myOperatorControllerManager.podAnnotations }}")
	})
}

func Test_deployment_podLabels(t *testing.T) {
	var testInstance deployment

	t.Run("podLabels seeded in values and template block present", func(t *testing.T) {
		obj := internal.GenerateObj(strDeplNoAnnotations)
		processed, tmpl, err := testInstance.Process(&metadata.Service{}, obj)
		assert.NoError(t, err)
		assert.True(t, processed)

		vals := tmpl.Values()
		controllerManager, ok := vals["myOperatorControllerManager"].(map[string]interface{})
		assert.True(t, ok, "expected myOperatorControllerManager key in values")
		podLabels, ok := controllerManager["podLabels"]
		assert.True(t, ok, "expected podLabels key in values")
		assert.Equal(t, map[string]interface{}{}, podLabels)

		var buf bytes.Buffer
		assert.NoError(t, tmpl.Write(&buf))
		output := buf.String()
		assert.Contains(t, output, "{{- if  .Values.myOperatorControllerManager.podLabels }}")
		assert.Contains(t, output, "{{- toYaml .Values.myOperatorControllerManager.podLabels | nindent 8 }}")
	})
}

var singleQuotesTest = []struct {
	input    string
	expected string
}{
	{
		"{{ .Values.x }}",
		"{{ .Values.x }}",
	},
	{
		"'{{ .Values.x }}'",
		"{{ .Values.x }}",
	},
	{
		"'{{ .Values.x }}:{{ .Values.y }}'",
		"{{ .Values.x }}:{{ .Values.y }}",
	},
	{
		"'{{ .Values.x }}:{{ .Values.y \n\t| default .Chart.AppVersion}}'",
		"{{ .Values.x }}:{{ .Values.y \n\t| default .Chart.AppVersion}}",
	},
	{
		"echo 'x'",
		"echo 'x'",
	},
	{
		"abcd: x.y['x/y']",
		"abcd: x.y['x/y']",
	},
	{
		"abcd: x.y[\"'{{}}'\"]",
		"abcd: x.y[\"{{}}\"]",
	},
	{
		"image: '{{ .Values.x }}'",
		"image: {{ .Values.x }}",
	},
	{
		"'{{ .Values.x }} y'",
		"{{ .Values.x }} y",
	},
	{
		"\t\t- mountPath: './x.y'",
		"\t\t- mountPath: './x.y'",
	},
	{
		"'{{}}'",
		"{{}}",
	},
	{
		"'{{ {nested} }}'",
		"{{ {nested} }}",
	},
	{
		"'{{ '{{nested}}' }}'",
		"{{ '{{nested}}' }}",
	},
	{
		"'{{ unbalanced }'",
		"'{{ unbalanced }'",
	},
	{
		"'{{\nincomplete content'",
		"'{{\nincomplete content'",
	},
	{
		"'{{ @#$%^&*() }}'",
		"{{ @#$%^&*() }}",
	},
}

func Test_replaceSingleQuotes(t *testing.T) {
	for _, tt := range singleQuotesTest {
		t.Run(tt.input, func(t *testing.T) {
			s := replaceSingleQuotes(tt.input)
			if s != tt.expected {
				t.Errorf("got %q, want %q", s, tt.expected)
			}
		})
	}
}
