package processor

import (
	"github.com/arttor/helmify/pkg/config"
	"testing"

	"github.com/arttor/helmify/internal"
	"github.com/arttor/helmify/pkg/metadata"
	"github.com/stretchr/testify/assert"
)

func TestProcessObjMeta(t *testing.T) {
	testMeta := metadata.New(config.Config{ChartName: "chart-name"})
	testMeta.Load(internal.TestNs)
	res, err := ProcessObjMeta(testMeta, internal.TestNs)
	assert.NoError(t, err)
	assert.Contains(t, res, "chart-name.labels")
	assert.Contains(t, res, "chart-name.fullname")
}

func TestStripSelectorLabels(t *testing.T) {
	tests := []struct {
		name   string
		labels map[string]string
		want   map[string]string
	}{
		{
			name:   "nil labels returned as-is",
			labels: nil,
			want:   nil,
		},
		{
			name:   "empty labels returned as-is",
			labels: map[string]string{},
			want:   map[string]string{},
		},
		{
			name: "drops only keys provided by selectorLabels",
			labels: map[string]string{
				"app.kubernetes.io/name":     "myapp",
				"app.kubernetes.io/instance": "myapp",
				"app.kubernetes.io/version":  "1.0", // kept: not part of selectorLabels
				"control-plane":              "controller-manager",
			},
			want: map[string]string{
				"app.kubernetes.io/version": "1.0",
				"control-plane":             "controller-manager",
			},
		},
		{
			name: "only selectorLabels keys yields empty map",
			labels: map[string]string{
				"app.kubernetes.io/name":     "myapp",
				"app.kubernetes.io/instance": "myapp",
			},
			want: map[string]string{},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			input := tt.labels
			got := StripSelectorLabels(input)
			assert.Equal(t, tt.want, got)
			// helper must not mutate the input map.
			if len(input) > 0 {
				assert.Contains(t, input, "app.kubernetes.io/name")
			}
		})
	}
}
