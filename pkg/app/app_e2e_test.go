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
	chartName = "my_test_chart"
)

func TestStart(t *testing.T) {
	file, err := os.Open("../../test_data/k8s-operator-kustomize.output")
	assert.NoError(t, err)

	objects := bufio.NewReader(file)
	err = Start(objects, config.Config{ChartName: chartName})
	assert.NoError(t, err)

	t.Cleanup(func() {
		err = os.RemoveAll(chartName)
		assert.NoError(t, err)
	})

	helmLint := action.NewLint()
	helmLint.Strict = true
	helmLint.Namespace = "test-ns"
	result := helmLint.Run([]string{chartName}, nil)
	for _, err = range result.Errors {
		assert.NoError(t, err)
	}
}
