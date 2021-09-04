package app

import (
	"bufio"
	"os"
	"testing"

	"github.com/arttor/helmify/pkg/config"
	"github.com/stretchr/testify/assert"
	"helm.sh/helm/v3/pkg/action"
)

const (
	operatorChartName = "my_test_operator_chart"
	appChartName      = "my_test_app_chart"
)

func TestOperator(t *testing.T) {
	file, err := os.Open("../../test_data/k8s-operator-kustomize.output")
	assert.NoError(t, err)

	objects := bufio.NewReader(file)
	err = Start(objects, config.Config{ChartName: operatorChartName})
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

	objects := bufio.NewReader(file)
	err = Start(objects, config.Config{ChartName: appChartName})
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
