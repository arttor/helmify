package app

import (
	"bufio"
	"context"
	"os"
	"testing"

	"github.com/arttor/helmify/pkg/config"
	"github.com/arttor/helmify/pkg/translator/k8smanifest"
	"github.com/stretchr/testify/assert"
	"helm.sh/helm/v3/pkg/action"
)

const (
	operatorChartName = "test-operator"
	appChartName      = "test-app"
)

func TestOperator(t *testing.T) {
	file, err := os.Open("../../test_data/k8s-operator-kustomize.output")
	assert.NoError(t, err)

	conf := config.Config{ChartName: operatorChartName}
	objects := bufio.NewReader(file)
	trans := k8smanifest.New(conf, objects)
	engine := NewEngine(conf)
	
	err = engine.Run(context.Background(), trans)
	assert.NoError(t, err)

	t.Cleanup(func() {
		err = os.RemoveAll(operatorChartName)
		assert.NoError(t, err)
	})

	helmLint := action.NewLint()
	helmLint.Strict = true
	helmLint.Namespace = "test-ns"
	result := helmLint.Run([]string{operatorChartName}, nil)
	for _, err = range result.Errors {
		assert.NoError(t, err)
	}
}

func TestApp(t *testing.T) {
	file, err := os.Open("../../test_data/sample-app.yaml")
	assert.NoError(t, err)

	conf := config.Config{ChartName: appChartName}
	objects := bufio.NewReader(file)
	trans := k8smanifest.New(conf, objects)
	engine := NewEngine(conf)

	err = engine.Run(context.Background(), trans)
	assert.NoError(t, err)

	t.Cleanup(func() {
		err = os.RemoveAll(appChartName)
		assert.NoError(t, err)
	})

	helmLint := action.NewLint()
	helmLint.Strict = true
	helmLint.Namespace = "test-ns"
	result := helmLint.Run([]string{appChartName}, nil)
	for _, err = range result.Errors {
		assert.NoError(t, err)
	}
}
