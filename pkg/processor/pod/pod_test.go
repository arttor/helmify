package pod

import (
	"testing"

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
)

func Test_pod_Process(t *testing.T) {
	t.Run("deployment with args", func(t *testing.T) {
		var deploy appsv1.Deployment
		obj := internal.GenerateObj(strDeployment)
		err := runtime.DefaultUnstructuredConverter.FromUnstructured(obj.Object, &deploy)
		specMap, tmpl, err := ProcessSpec("nginx", &metadata.Service{}, deploy.Spec.Template.Spec, deploy.TypeMeta.Kind)
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
			},
		}, tmpl)
	})

	t.Run("deployment with no args", func(t *testing.T) {
		var deploy appsv1.Deployment
		obj := internal.GenerateObj(strDeploymentWithNoArgs)
		err := runtime.DefaultUnstructuredConverter.FromUnstructured(obj.Object, &deploy)
		specMap, tmpl, err := ProcessSpec("nginx", &metadata.Service{}, deploy.Spec.Template.Spec, deploy.TypeMeta.Kind)
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
		}, specMap)

		assert.Equal(t, helmify.Values{
			"nginx": map[string]interface{}{
				"nginx": map[string]interface{}{
					"image": map[string]interface{}{
						"repository": "nginx",
						"tag":        "1.14.2",
					},
				},
			},
		}, tmpl)
	})

	t.Run("deployment with image tag and digest", func(t *testing.T) {
		var deploy appsv1.Deployment
		obj := internal.GenerateObj(strDeploymentWithTagAndDigest)
		err := runtime.DefaultUnstructuredConverter.FromUnstructured(obj.Object, &deploy)
		specMap, tmpl, err := ProcessSpec("nginx", &metadata.Service{}, deploy.Spec.Template.Spec, deploy.TypeMeta.Kind)
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
		}, specMap)

		assert.Equal(t, helmify.Values{
			"nginx": map[string]interface{}{
				"nginx": map[string]interface{}{
					"image": map[string]interface{}{
						"repository": "nginx",
						"tag":        "1.14.2@sha256:cb5c1bddd1b5665e1867a7fa1b5fa843a47ee433bbb75d4293888b71def53229",
					},
				},
			},
		}, tmpl)
	})

	t.Run("deployment with image tag and port", func(t *testing.T) {
		var deploy appsv1.Deployment
		obj := internal.GenerateObj(strDeploymentWithPort)
		err := runtime.DefaultUnstructuredConverter.FromUnstructured(obj.Object, &deploy)
		specMap, tmpl, err := ProcessSpec("nginx", &metadata.Service{}, deploy.Spec.Template.Spec, deploy.TypeMeta.Kind)
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
		}, specMap)

		assert.Equal(t, helmify.Values{
			"nginx": map[string]interface{}{
				"nginx": map[string]interface{}{
					"image": map[string]interface{}{
						"repository": "localhost:6001/my_project",
						"tag":        "latest",
					},
				},
			},
		}, tmpl)
	})

}
