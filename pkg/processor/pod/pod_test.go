package pod

import (
	"testing"

	"github.com/arttor/helmify/pkg/config"
	"github.com/arttor/helmify/pkg/helmify"
	"github.com/arttor/helmify/pkg/metadata"
	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/runtime"

	"github.com/arttor/helmify/internal"
	"github.com/stretchr/testify/assert"
)

const (
	strDeployment = `
apiVersion: apps/v1
kind: Deployment
metadata:
  name: nginx-deployment
  labels:
    app: nginx
spec:
  replicas: 3
  selector:
    matchLabels:
      app: nginx
  template:
    metadata:
      labels:
        app: nginx
    spec:
      containers:
      - name: nginx
        image: nginx:1.14.2
        args:
        - --test
        - --arg
        ports:
        - containerPort: 80
`

	strDeploymentWithTagAndDigest = `
apiVersion: apps/v1
kind: Deployment
metadata:
  name: nginx-deployment
  labels:
    app: nginx
spec:
  replicas: 3
  selector:
    matchLabels:
      app: nginx
  template:
    metadata:
      labels:
        app: nginx
    spec:
      containers:
      - name: nginx
        image: nginx:1.14.2@sha256:cb5c1bddd1b5665e1867a7fa1b5fa843a47ee433bbb75d4293888b71def53229
        ports:
        - containerPort: 80
`

	strDeploymentWithNoArgs = `
apiVersion: apps/v1
kind: Deployment
metadata:
  name: nginx-deployment
  labels:
    app: nginx
spec:
  replicas: 3
  selector:
    matchLabels:
      app: nginx
  template:
    metadata:
      labels:
        app: nginx
    spec:
      containers:
      - name: nginx
        image: nginx:1.14.2
        ports:
        - containerPort: 80
`

	strDeploymentWithPort = `
apiVersion: apps/v1
kind: Deployment
metadata:
  name: nginx-deployment
  labels:
    app: nginx
spec:
  replicas: 3
  selector:
    matchLabels:
      app: nginx
  template:
    metadata:
      labels:
        app: nginx
    spec:
      containers:
      - name: nginx
        image: localhost:6001/my_project:latest
        ports:
        - containerPort: 80
`
	strDeploymentWithPodSecurityContext = `
apiVersion: apps/v1
kind: Deployment
metadata:
  name: nginx-deployment
  labels:
    app: nginx
spec:
  replicas: 3
  selector:
    matchLabels:
      app: nginx
  template:
    metadata:
      labels:
        app: nginx
    spec:
      containers:
      - name: nginx
        image: localhost:6001/my_project:latest
      securityContext:
        fsGroup: 20000
        runAsGroup: 30000
        runAsNonRoot: true
        runAsUser: 65532

`
	strDeploymentWithImagePullSecrets = `
apiVersion: apps/v1
kind: Deployment
metadata:
  name: nginx-deployment
  labels:
    app: nginx
spec:
  replicas: 3
  selector:
    matchLabels:
      app: nginx
  template:
    metadata:
      labels:
        app: nginx
    spec:
      containers:
      - name: nginx
        image: nginx:1.14.2
        args:
        - --test
        - --arg
        ports:
        - containerPort: 80
      imagePullSecrets:
      - name: myregistrykey
`
)

func Test_pod_Process(t *testing.T) {
	t.Run("deployment with args", func(t *testing.T) {
		var deploy appsv1.Deployment
		obj := internal.GenerateObj(strDeployment)
		err := runtime.DefaultUnstructuredConverter.FromUnstructured(obj.Object, &deploy)
		specMap, tmpl, err := ProcessSpec("nginx", &metadata.Service{}, deploy.Spec.Template.Spec, 0)
		assert.NoError(t, err)

		assert.Equal(t, map[string]interface{}{
			"containers": []interface{}{
				map[string]interface{}{
					"args": "{{- toYaml .Values.nginx.nginx.args | nindent 8 }}",
					"env": []interface{}{
						map[string]interface{}{
							"name":  "KUBERNETES_CLUSTER_DOMAIN",
							"value": "{{ quote .Values.kubernetesClusterDomain }}",
						},
					},
					"image": "{{ .Values.nginx.nginx.image.repository }}:{{ .Values.nginx.nginx.image.tag | default .Chart.AppVersion }}",
					"name":  "nginx", "ports": []interface{}{
						map[string]interface{}{
							"containerPort": int64(80),
						},
					},
					"resources": map[string]interface{}{},
				},
			},
			"tolerations":               "{{- toYaml .Values.nginx.tolerations | nindent 8 }}",
			"topologySpreadConstraints": "{{- toYaml .Values.nginx.topologySpreadConstraints | nindent 8 }}",
			"nodeSelector":              "{{- toYaml .Values.nginx.nodeSelector | nindent 8 }}",
			"serviceAccountName":        `{{ include ".serviceAccountName" . }}`,
		}, specMap)

		assert.Equal(t, helmify.Values{
			"nginx": map[string]interface{}{
				"nginx": map[string]interface{}{
					"image": map[string]interface{}{
						"repository": "nginx",
						"tag":        "1.14.2",
					},
					"args": []interface{}{
						"--test",
						"--arg",
					},
				},
				"nodeSelector":              map[string]interface{}{},
				"tolerations":               []interface{}{},
				"topologySpreadConstraints": []interface{}{},
			},
		}, tmpl)
	})

	t.Run("deployment with no args", func(t *testing.T) {
		var deploy appsv1.Deployment
		obj := internal.GenerateObj(strDeploymentWithNoArgs)
		err := runtime.DefaultUnstructuredConverter.FromUnstructured(obj.Object, &deploy)
		specMap, tmpl, err := ProcessSpec("nginx", &metadata.Service{}, deploy.Spec.Template.Spec, 0)
		assert.NoError(t, err)

		assert.Equal(t, map[string]interface{}{
			"containers": []interface{}{
				map[string]interface{}{
					"env": []interface{}{
						map[string]interface{}{
							"name":  "KUBERNETES_CLUSTER_DOMAIN",
							"value": "{{ quote .Values.kubernetesClusterDomain }}",
						},
					},
					"image": "{{ .Values.nginx.nginx.image.repository }}:{{ .Values.nginx.nginx.image.tag | default .Chart.AppVersion }}",
					"name":  "nginx", "ports": []interface{}{
						map[string]interface{}{
							"containerPort": int64(80),
						},
					},
					"resources": map[string]interface{}{},
				},
			},
			"nodeSelector":              "{{- toYaml .Values.nginx.nodeSelector | nindent 8 }}",
			"serviceAccountName":        `{{ include ".serviceAccountName" . }}`,
			"tolerations":               "{{- toYaml .Values.nginx.tolerations | nindent 8 }}",
			"topologySpreadConstraints": "{{- toYaml .Values.nginx.topologySpreadConstraints | nindent 8 }}",
		}, specMap)

		assert.Equal(t, helmify.Values{
			"nginx": map[string]interface{}{
				"nginx": map[string]interface{}{
					"image": map[string]interface{}{
						"repository": "nginx",
						"tag":        "1.14.2",
					},
				},
				"nodeSelector":              map[string]interface{}{},
				"tolerations":               []interface{}{},
				"topologySpreadConstraints": []interface{}{},
			},
		}, tmpl)
	})

	t.Run("deployment with image tag and digest", func(t *testing.T) {
		var deploy appsv1.Deployment
		obj := internal.GenerateObj(strDeploymentWithTagAndDigest)
		err := runtime.DefaultUnstructuredConverter.FromUnstructured(obj.Object, &deploy)
		specMap, tmpl, err := ProcessSpec("nginx", &metadata.Service{}, deploy.Spec.Template.Spec, 0)
		assert.NoError(t, err)

		assert.Equal(t, map[string]interface{}{
			"containers": []interface{}{
				map[string]interface{}{
					"env": []interface{}{
						map[string]interface{}{
							"name":  "KUBERNETES_CLUSTER_DOMAIN",
							"value": "{{ quote .Values.kubernetesClusterDomain }}",
						},
					},
					"image": "{{ .Values.nginx.nginx.image.repository }}:{{ .Values.nginx.nginx.image.tag | default .Chart.AppVersion }}",
					"name":  "nginx", "ports": []interface{}{
						map[string]interface{}{
							"containerPort": int64(80),
						},
					},
					"resources": map[string]interface{}{},
				},
			},
			"nodeSelector":              "{{- toYaml .Values.nginx.nodeSelector | nindent 8 }}",
			"serviceAccountName":        `{{ include ".serviceAccountName" . }}`,
			"tolerations":               "{{- toYaml .Values.nginx.tolerations | nindent 8 }}",
			"topologySpreadConstraints": "{{- toYaml .Values.nginx.topologySpreadConstraints | nindent 8 }}",
		}, specMap)

		assert.Equal(t, helmify.Values{
			"nginx": map[string]interface{}{
				"nginx": map[string]interface{}{
					"image": map[string]interface{}{
						"repository": "nginx",
						"tag":        "1.14.2@sha256:cb5c1bddd1b5665e1867a7fa1b5fa843a47ee433bbb75d4293888b71def53229",
					},
				},
				"nodeSelector":              map[string]interface{}{},
				"tolerations":               []interface{}{},
				"topologySpreadConstraints": []interface{}{},
			},
		}, tmpl)
	})

	t.Run("deployment with image tag and port", func(t *testing.T) {
		var deploy appsv1.Deployment
		obj := internal.GenerateObj(strDeploymentWithPort)
		err := runtime.DefaultUnstructuredConverter.FromUnstructured(obj.Object, &deploy)
		specMap, tmpl, err := ProcessSpec("nginx", &metadata.Service{}, deploy.Spec.Template.Spec, 0)
		assert.NoError(t, err)

		assert.Equal(t, map[string]interface{}{
			"containers": []interface{}{
				map[string]interface{}{
					"env": []interface{}{
						map[string]interface{}{
							"name":  "KUBERNETES_CLUSTER_DOMAIN",
							"value": "{{ quote .Values.kubernetesClusterDomain }}",
						},
					},
					"image": "{{ .Values.nginx.nginx.image.repository }}:{{ .Values.nginx.nginx.image.tag | default .Chart.AppVersion }}",
					"name":  "nginx", "ports": []interface{}{
						map[string]interface{}{
							"containerPort": int64(80),
						},
					},
					"resources": map[string]interface{}{},
				},
			},
			"nodeSelector":              "{{- toYaml .Values.nginx.nodeSelector | nindent 8 }}",
			"serviceAccountName":        `{{ include ".serviceAccountName" . }}`,
			"tolerations":               "{{- toYaml .Values.nginx.tolerations | nindent 8 }}",
			"topologySpreadConstraints": "{{- toYaml .Values.nginx.topologySpreadConstraints | nindent 8 }}",
		}, specMap)

		assert.Equal(t, helmify.Values{
			"nginx": map[string]interface{}{
				"nginx": map[string]interface{}{
					"image": map[string]interface{}{
						"repository": "localhost:6001/my_project",
						"tag":        "latest",
					},
				},
				"nodeSelector":              map[string]interface{}{},
				"tolerations":               []interface{}{},
				"topologySpreadConstraints": []interface{}{},
			},
		}, tmpl)
	})
	t.Run("deployment with securityContext", func(t *testing.T) {
		var deploy appsv1.Deployment
		obj := internal.GenerateObj(strDeploymentWithPodSecurityContext)
		err := runtime.DefaultUnstructuredConverter.FromUnstructured(obj.Object, &deploy)
		specMap, tmpl, err := ProcessSpec("nginx", &metadata.Service{}, deploy.Spec.Template.Spec, 0)
		assert.NoError(t, err)
		assert.Equal(t, map[string]interface{}{
			"containers": []interface{}{
				map[string]interface{}{
					"env": []interface{}{
						map[string]interface{}{
							"name":  "KUBERNETES_CLUSTER_DOMAIN",
							"value": "{{ quote .Values.kubernetesClusterDomain }}",
						},
					},
					"image":     "{{ .Values.nginx.nginx.image.repository }}:{{ .Values.nginx.nginx.image.tag | default .Chart.AppVersion }}",
					"name":      "nginx",
					"resources": map[string]interface{}{},
				},
			},
			"securityContext":           "{{- toYaml .Values.nginx.podSecurityContext | nindent 8 }}",
			"nodeSelector":              "{{- toYaml .Values.nginx.nodeSelector | nindent 8 }}",
			"serviceAccountName":        `{{ include ".serviceAccountName" . }}`,
			"tolerations":               "{{- toYaml .Values.nginx.tolerations | nindent 8 }}",
			"topologySpreadConstraints": "{{- toYaml .Values.nginx.topologySpreadConstraints | nindent 8 }}",
		}, specMap)

		assert.Equal(t, helmify.Values{
			"nginx": map[string]interface{}{
				"podSecurityContext": map[string]interface{}{
					"fsGroup":      int64(20000),
					"runAsGroup":   int64(30000),
					"runAsNonRoot": true,
					"runAsUser":    int64(65532),
				},
				"nginx": map[string]interface{}{
					"image": map[string]interface{}{
						"repository": "localhost:6001/my_project",
						"tag":        "latest",
					},
				},
				"nodeSelector":              map[string]interface{}{},
				"tolerations":               []interface{}{},
				"topologySpreadConstraints": []interface{}{},
			},
		}, tmpl)
	})
	t.Run("deployment without imagePullSecrets", func(t *testing.T) {
		var deploy appsv1.Deployment
		obj := internal.GenerateObj(strDeployment)
		err := runtime.DefaultUnstructuredConverter.FromUnstructured(obj.Object, &deploy)
		assert.NoError(t, err)
		specMap, _, err := ProcessSpec("nginx", &metadata.Service{}, deploy.Spec.Template.Spec)
		assert.NoError(t, err)

		// when ImagePullSecrets is disabled in config, spec should not contain imagePullSecrets key
		_, ok := specMap["imagePullSecrets"]
		assert.False(t, ok)
	})

	t.Run("deployment with imagePullSecrets enabled but not provided in source", func(t *testing.T) {
		var deploy appsv1.Deployment
		obj := internal.GenerateObj(strDeployment)
		err := runtime.DefaultUnstructuredConverter.FromUnstructured(obj.Object, &deploy)
		assert.NoError(t, err)
		// enable ImagePullSecrets in config via metadata.New but source doesn't include imagePullSecrets
		svc := metadata.New(config.Config{ImagePullSecrets: true})
		specMap, tmpl, err := ProcessSpec("nginx", svc, deploy.Spec.Template.Spec)
		assert.NoError(t, err)

		// spec should contain templated imagePullSecrets
		assert.Equal(t, "{{ .Values.imagePullSecrets | default list | toJson }}", specMap["imagePullSecrets"])

		assert.Equal(t, []interface{}{}, tmpl["imagePullSecrets"])
	})

	t.Run("deployment with imagePullSecrets", func(t *testing.T) {

		var deploy appsv1.Deployment
		obj := internal.GenerateObj(strDeploymentWithImagePullSecrets)
		err := runtime.DefaultUnstructuredConverter.FromUnstructured(obj.Object, &deploy)
		assert.NoError(t, err)
		// enable ImagePullSecrets in config via metadata.New
		svc := metadata.New(config.Config{ImagePullSecrets: true})
		specMap, tmpl, err := ProcessSpec("nginx", svc, deploy.Spec.Template.Spec)
		assert.NoError(t, err)

		// spec should contain templated imagePullSecrets
		assert.Equal(t, "{{ .Values.imagePullSecrets | default list | toJson }}", specMap["imagePullSecrets"])

		// values should contain the original imagePullSecrets slice
		assert.Equal(t, []interface{}{
			map[string]interface{}{"name": "myregistrykey"},
		}, tmpl["imagePullSecrets"])
	})

}
