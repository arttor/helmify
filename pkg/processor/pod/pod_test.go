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

	strDeploymentWithPodSpecs = `
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
      nodeSelector:
        region: east
        type: user-node
      tolerations:
      - key: "dedicated"
        operator: "Equal"
        value: "special-user"
        effect: "NoSchedule"
      topologySpreadConstraints:
      - maxSkew: 1
        topologyKey: kubernetes.io/hostname
        whenUnsatisfiable: DoNotSchedule
        labelSelector:
          matchLabels:
            app: nginx
      affinity:
        nodeAffinity:
          requiredDuringSchedulingIgnoredDuringExecution:
            nodeSelectorTerms:
            - matchExpressions:
              - key: kubernetes.io/os
                operator: In
                values:
                - linux
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
			"affinity":                  "{{- toYaml .Values.nginx.affinity | nindent 8 }}",
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
				"affinity":                  map[string]interface{}{},
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
			"affinity":                  "{{- toYaml .Values.nginx.affinity | nindent 8 }}",
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
				"affinity":                  map[string]interface{}{},
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
			"affinity":                  "{{- toYaml .Values.nginx.affinity | nindent 8 }}",
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
				"affinity":                  map[string]interface{}{},
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
			"affinity":                  "{{- toYaml .Values.nginx.affinity | nindent 8 }}",
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
				"affinity":                  map[string]interface{}{},
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
			"affinity":                  "{{- toYaml .Values.nginx.affinity | nindent 8 }}",
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
				"affinity":                  map[string]interface{}{},
				"nodeSelector":              map[string]interface{}{},
				"tolerations":               []interface{}{},
				"topologySpreadConstraints": []interface{}{},
			},
		}, tmpl)
	})

	t.Run("deployment with pod-level specs", func(t *testing.T) {
		var deploy appsv1.Deployment
		obj := internal.GenerateObj(strDeploymentWithPodSpecs)
		err := runtime.DefaultUnstructuredConverter.FromUnstructured(obj.Object, &deploy)
		specMap, tmpl, err := ProcessSpec("nginx", &metadata.Service{}, deploy.Spec.Template.Spec, 0)
		assert.NoError(t, err)

		// specMap should have all pod-level specs as templated values
		assert.Equal(t, "{{- toYaml .Values.nginx.nodeSelector | nindent 8 }}", specMap["nodeSelector"])
		assert.Equal(t, "{{- toYaml .Values.nginx.tolerations | nindent 8 }}", specMap["tolerations"])
		assert.Equal(t, "{{- toYaml .Values.nginx.topologySpreadConstraints | nindent 8 }}", specMap["topologySpreadConstraints"])
		assert.Equal(t, "{{- toYaml .Values.nginx.affinity | nindent 8 }}", specMap["affinity"])

		// values should contain the actual values from the manifest
		nginxValues := tmpl["nginx"].(map[string]interface{})

		// nodeSelector values
		nodeSelector := nginxValues["nodeSelector"].(map[string]interface{})
		assert.Equal(t, "east", nodeSelector["region"])
		assert.Equal(t, "user-node", nodeSelector["type"])

		// tolerations values
		tolerations := nginxValues["tolerations"].([]interface{})
		assert.Len(t, tolerations, 1)
		toleration := tolerations[0].(map[string]interface{})
		assert.Equal(t, "dedicated", toleration["key"])
		assert.Equal(t, "Equal", toleration["operator"])
		assert.Equal(t, "special-user", toleration["value"])
		assert.Equal(t, "NoSchedule", toleration["effect"])

		// topologySpreadConstraints values
		tsc := nginxValues["topologySpreadConstraints"].([]interface{})
		assert.Len(t, tsc, 1)
		constraint := tsc[0].(map[string]interface{})
		assert.Equal(t, float64(1), constraint["maxSkew"])
		assert.Equal(t, "kubernetes.io/hostname", constraint["topologyKey"])
		assert.Equal(t, "DoNotSchedule", constraint["whenUnsatisfiable"])

		// affinity values
		affinity := nginxValues["affinity"].(map[string]interface{})
		assert.Contains(t, affinity, "nodeAffinity")
		nodeAffinity := affinity["nodeAffinity"].(map[string]interface{})
		assert.Contains(t, nodeAffinity, "requiredDuringSchedulingIgnoredDuringExecution")
	})

}
