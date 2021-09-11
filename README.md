# Helmify
[![CI](https://github.com/arttor/helmify/actions/workflows/ci.yml/badge.svg)](https://github.com/arttor/helmify/actions/workflows/ci.yml)
![GitHub go.mod Go version](https://img.shields.io/github/go-mod/go-version/arttor/helmify)
![GitHub](https://img.shields.io/github/license/arttor/helmify)
![GitHub release (latest by date)](https://img.shields.io/github/v/release/arttor/helmify)
[![Go Report Card](https://goreportcard.com/badge/github.com/arttor/helmify)](https://goreportcard.com/report/github.com/arttor/helmify)
[![GoDoc](https://godoc.org/github.com/arttor/helmify?status.svg)](https://pkg.go.dev/github.com/arttor/helmify?tab=doc)
[![Maintainability](https://api.codeclimate.com/v1/badges/2ee755bb948d363207bb/maintainability)](https://codeclimate.com/github/arttor/helmify/maintainability)
[![Test Coverage](https://api.codeclimate.com/v1/badges/2ee755bb948d363207bb/test_coverage)](https://codeclimate.com/github/arttor/helmify/test_coverage)

CLI that creates [Helm](https://github.com/helm/helm) charts from kubernetes yamls.

Helmify reads a list of [supported k8s objects](#status) from stdin and converts it to a helm chart.

Submit issue if some features missing for your use-case.

## Usage

1) From yaml file: 
```shell
cat my-app.yaml | helmify mychart
```
- Will create 'mychart' directory with Helm chart from yaml file with k8s objects.
    <details>
    <summary>Show sample input taml:</summary>
  
    ```yaml
    apiVersion: apps/v1
    kind: Deployment
    metadata:
      name: someapp
      namespace: my-ns
    spec:
      replicas: 3
      selector:
        matchLabels:
          app: myapp
      template:
        metadata:
          labels:
            app: myapp
        spec:
          containers:
            - name: app
              command:
                - /myapp
              volumeMounts:
                - mountPath: /my_config.yaml
                  name: manager-config
                  subPath: my_config.yaml
                - name: secret-volume
                  mountPath: /my.ca
              env:
                - name: VAR
                  valueFrom:
                    secretKeyRef:
                      name: secret-vars
                      key: VAR
              image: myimage:latest
              resources:
                limits:
                  cpu: 100m
                  memory: 30Mi
                requests:
                  cpu: 100m
                  memory: 20Mi
            - name: proxy-sidecar
              image: proxy-image:v0.8.0
              ports:
                - containerPort: 8443
                  name: https
          volumes:
            - configMap:
                name: my-config
              name: manager-config
            - name: secret-volume
              secret:
                secretName: my-secret-ca
    ---
    apiVersion: v1
    kind: Service
    metadata:
      labels:
        app: myapp
      name: my-service
      namespace: my-ns
    spec:
      ports:
        - name: https
          port: 8443
          targetPort: https
      selector:
        app: myapp
    ---
    apiVersion: v1
    kind: Secret
    metadata:
      name: my-secret-ca
      namespace: my-ns
    type: opaque
    data:
      ca.crt: |
        c3VwZXJsb25ndGVzdGNydC1zdXBlcmxvbmd0ZXN0Y3J0LXN1cGVybG9uZ3Rlc3RjcnQtc3
        VwZXJsb25ndGVzdGNydC1zdXBlcmxvbmd0ZXN0Y3J0LXN1cGVybG9uZ3Rlc3RjcnQtc3Vw
        ZXJsb25ndGVzdGNydC0Kc3VwZXJsb25ndGVzdGNydC1zdXBlcmxvbmd0ZXN0Y3J0LXN1cG
        VybG9uZ3Rlc3RjcnQtc3VwZXJsb25ndGVzdGNydC1zdXBlcmxvbmd0ZXN0Y3J0LXN1cGVy
        bG9uZ3Rlc3RjcnQKc3VwZXJsb25ndGVzdGNydC1zdXBlcmxvbmd0ZXN0Y3J0LXN1cGVybG
        9uZ3Rlc3RjcnQtc3VwZXJsb25ndGVzdGNydC1zdXBlcmxvbmd0ZXN0Y3J0LXN1cGVybG9u
        Z3Rlc3RjcnQ=
    ---
    apiVersion: v1
    kind: Secret
    metadata:
      name: secret-vars
      namespace: my-ns
    type: opaque
    data:
      VAR: bXlfc2VjcmV0X3Zhcl8x
    ---
    apiVersion: v1
    kind: ConfigMap
    metadata:
      name: my-config
      namespace: my-ns
    data:
      dummyconfigmapkey: dummyconfigmapvalue
      my_config.yaml: |
        health:
          healthProbeBindAddress: :8081
        metrics:
          bindAddress: 127.0.0.1:8080
      my_config.properties: |
        health.healthProbeBindAddress=8081
        metrics.bindAddress=127.0.0.1:8080
    ```

    </details>
    <details>
    <summary>Show resulted helm chart:</summary>
    mychart Helm chart directory with following structure: 
  
    ```
    mychart
    ├── Chart.yaml
    ├── templates
    │   ├── _helpers.tpl
    │   ├── deployment.yaml
    │   ├── my-config.yaml
    │   ├── my-secret-ca.yaml
    │   ├── my-service.yaml
    │   └── secret-vars.yaml
    └── values.yaml
    ```
  
    and contents:
    ```yaml
    # Values.yaml
    image:
      app:
        repository: myimage
        tag: latest
      proxySidecar:
        repository: proxy-image
        tag: v0.8.0
    myConfig:
      dummyconfigmapkey: dummyconfigmapvalue
      myConfigProperties:
        health:
          healthProbeBindAddress: "8081"
        metrics:
          bindAddress: 127.0.0.1:8080
      myConfigYaml:
        health:
          healthProbeBindAddress: :8081
        metrics:
          bindAddress: 127.0.0.1:8080
    mySecretCa:
      caCrt: ""
    myService:
      ports:
        - name: https
          port: 8443
          targetPort: https
      type: ClusterIP
    secretVars:
      var: ""
    someapp:
      app:
        resources:
          limits:
            cpu: 100m
            memory: 30Mi
          requests:
            cpu: 100m
            memory: 20Mi
      replicas: 3
    ---
    # templates/deployment.yaml
    apiVersion: apps/v1
    kind: Deployment
    metadata:
      name: {{ include "mychart.fullname" . }}-someapp
      labels:
            {{- include "mychart.labels" . | nindent 4 }}
    spec:
      replicas: {{ .Values.someapp.replicas }}
      selector:
        matchLabels:
          app: myapp
              {{- include "mychart.selectorLabels" . | nindent 6 }}
      template:
        metadata:
          labels:
            app: myapp
                {{- include "mychart.selectorLabels" . | nindent 8 }}
        spec:
          containers:
            - command:
                - /myapp
              env:
                - name: VAR
                  valueFrom:
                    secretKeyRef:
                      key: VAR
                      name: {{ include "mychart.fullname" . }}-secret-vars
              image: {{ .Values.image.app.repository }}:{{ .Values.image.app.tag | default .Chart.AppVersion
                       }}
              name: app
              resources: {{- toYaml .Values.someapp.app.resources | nindent 10 }}
              volumeMounts:
                - mountPath: /my_config.yaml
                  name: manager-config
                  subPath: my_config.yaml
                - mountPath: /my.ca
                  name: secret-volume
            - image: {{ .Values.image.proxySidecar.repository }}:{{ .Values.image.proxySidecar.tag
                       | default .Chart.AppVersion }}
              name: proxy-sidecar
              ports:
                - containerPort: 8443
                  name: https
              resources: {}
          volumes:
            - configMap:
                name: {{ include "mychart.fullname" . }}-my-config
              name: manager-config
            - name: secret-volume
              secret:
                secretName: {{ include "mychart.fullname" . }}-my-secret-ca
    ---              
    # templates/my-config.yaml
    apiVersion: v1
    kind: ConfigMap
    metadata:
      name: {{ include "mychart.fullname" . }}-my-config
      labels:
            {{- include "mychart.labels" . | nindent 4 }}
    data:
      dummyconfigmapkey: {{ .Values.myConfig.dummyconfigmapkey | quote }}
      my_config.properties: |
        health.healthProbeBindAddress={{ .Values.myConfig.myConfigProperties.health.healthProbeBindAddress | quote }}
        metrics.bindAddress={{ .Values.myConfig.myConfigProperties.metrics.bindAddress | quote }}
      my_config.yaml: |
        health:
          healthProbeBindAddress: {{ .Values.myConfig.myConfigYaml.health.healthProbeBindAddress
            | quote }}
        metrics:
          bindAddress: {{ .Values.myConfig.myConfigYaml.metrics.bindAddress | quote }}
    ---
    # templates/my-secret-ca.yaml
    apiVersion: v1
    kind: Secret
    metadata:
      name: {{ include "mychart.fullname" . }}-my-secret-ca
      labels:
            {{- include "mychart.labels" . | nindent 4 }}
    data:
      ca.crt: '{{ required "secret mySecretCa.caCrt is required" .Values.mySecretCa.caCrt
        | b64enc }}'
    ---
    # templates/my-service.yaml
    apiVersion: v1
    kind: Service
    metadata:
      name: {{ include "mychart.fullname" . }}-my-service
      labels:
        app: myapp
            {{- include "mychart.labels" . | nindent 4 }}
    spec:
      type: {{ .Values.myService.type }}
      selector:
        app: myapp
            {{- include "mychart.selectorLabels" . | nindent 4 }}
      ports:
            {{- .Values.myService.ports | toYaml | nindent 2 -}}
    ---
    # templates/secret-vars.yaml
    apiVersion: v1
    kind: Secret
    metadata:
      name: {{ include "mychart.fullname" . }}-secret-vars
      labels:
            {{- include "mychart.labels" . | nindent 4 }}
    data:
      VAR: '{{ required "secret secretVars.var is required" .Values.secretVars.var | b64enc
        }}'
    ```
    </details>

2) From directory with yamls:
```shell
awk 'FNR==1 && NR!=1  {print "---"}{print}' /<my_directory>/*.yaml | helmify mychart
```
- Will create 'mychart' directory with Helm chart from all yaml files in `<my_directory> `directory.

3) From [kustomize](https://kustomize.io/) output:
```shell
   kustomize build <kustomize_dir> | helmify mychart
```
- Will create 'mychart' directory with Helm chart from kustomize output.

### Integrate to your Operator-SDK/Kubebuilder project
Tested with operator-sdk version: "v1.8.0".
1. Open `Makefile` in your operator project generated by 
   [Operator-SDK](https://github.com/operator-framework/operator-sdk) or [Kubebuilder](https://github.com/kubernetes-sigs/kubebuilder).
2. Add these lines to `Makefile`:
```makefile
HELMIFY = $(shell pwd)/bin/helmify
helmify:
	$(call go-get-tool,$(HELMIFY),github.com/arttor/helmify/cmd/helmify@v0.3.0)

helm: manifests kustomize helmify
	$(KUSTOMIZE) build config/default | $(HELMIFY)
```
3. Run `make helm` in project root. It will generate helm chart with name 'chart' in 'chart' directory.

## Install

Manually:
- Download suitable for your system binary from [the Releases page](https://github.com/arttor/helmify/releases/latest).
- Unpack the helmify binary and add it to your PATH and you are good to go!

With [Homebrew](https://brew.sh/):
 ```shell
brew install arttor/tap/helmify
```

## Available options
Helmify takes a chart name for an argument.
Usage:

```helmify [flags] CHART_NAME```  -  `CHART_NAME` is optional. Default is 'chart'.

| flag | description | sample |
| --- | --- | --- |
| -h -help | Prints help | `helmify -h`|
| -v | Enable verbose output. Prints WARN and INFO. | `helmify -v`|
| -vv | Enable very verbose output. Also prints DEBUG. | `helmify -vv`|
| -version | Print helmify version. | `helmify -version`|

## Status
Supported default operator resources:
- deployment
- service
- RBAC (serviceaccount, (cluster-)role, (cluster-)rolebinding)
- configs (configmap, secret)
- webhooks (cert, issuer, ValidatingWebhookConfiguration)

### Known issues
- Helmify will not overwrite `Chart.yaml` file if presented. Done on purpose.
- Helmify will not delete existing template files, only overwrite.
- Helmify overwrites templates and values files on every run. 
  This meas that all your manual changes in helm template files will be lost on the next run.
  
## Develop
To support a new type of k8s object template:
1. Implement `helmify.Processor` interface. Place implementation in `pkg/processor`. The package contains 
examples for most k8s objects.
2. Register your processor in the `pkg/app/app.go`
3. Add relevant input sample to `test_data/kustomize.output`.


### Run
Clone repo and execute command:

```shell
cat test_data/k8s-operator-kustomize.output | go run ./cmd/helmify mychart
```

Will generate `mychart` Helm chart form file `test_data/k8s-operator-kustomize.output` representing typical operator
[kustomize](https://github.com/kubernetes-sigs/kustomize) output.

### Test
For manual testing, run program with debug output:
```shell
cat test_data/k8s-operator-kustomize.output | go run ./cmd/helmify -vv mychart
```
Then inspect logs and generated chart in `./mychart` directory.

To execute tests, run:
```shell
go test ./...
```
Beside unit-tests, project contains e2e test `pkg/app/app_e2e_test.go`.
It's a go test, which uses `test_data/*` to generate a chart in temporary directory. 
Then runs `helm lint --strict` to check if generated chart is valid.
