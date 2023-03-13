package constraints

import (
	"testing"

	"github.com/arttor/helmify/pkg/helmify"
	"github.com/stretchr/testify/require"
	v1 "k8s.io/api/core/v1"
)

const templatedResult = "{{- if .Values.topologySpreadConstraints }}\n" +
	"      topologySpreadConstraints: {{- include \"tplvalues.render\" (dict \"value\" .Values.topologySpreadConstraints \"context\" $) | nindent 8 }}\n" +
	"{{- end }}\n" +
	"{{- if .Values.nodeSelector }}\n" +
	"      nodeSelector: {{- include \"tplvalues.render\" ( dict \"value\" .Values.nodeSelector \"context\" $) | nindent 8 }}\n" +
	"{{- end }}\n" +
	"{{- if .Values.tolerations }}\n" +
	"      tolerations: {{- include \"tplvalues.render\" (dict \"value\" .Values.tolerations \"context\" .) | nindent 8 }}\n" +
	"{{- end }}\n"

func TestProcessSpecMap(t *testing.T) {

	tests := []struct {
		name       string
		specMap    map[string]interface{}
		values     *helmify.Values
		podspec    v1.PodSpec
		want       string
		wantValues *helmify.Values
	}{
		{name: "no predefined resource returns still a template and to fill in values",
			specMap: make(map[string]interface{}, 4),
			values: &helmify.Values{
				"mydep": map[string]interface{}{},
			},
			want: templatedResult,
			wantValues: &helmify.Values{
				"mydep": map[string]interface{}{
					"nodeSelector":              map[string]string{},
					"tolerations":               []interface{}{},
					"topologySpreadConstraints": []interface{}{},
				},
			},
		},
		{name: "predefined resource are added to values, template is the same",
			values: &helmify.Values{
				"mydep": map[string]interface{}{},
			},
			specMap: map[string]interface{}{

				"topologySpreadConstraints": []v1.TopologySpreadConstraint{
					{
						MaxSkew:           0,
						TopologyKey:       "trtr",
						WhenUnsatisfiable: "test",
						LabelSelector:     nil,
					},
				},
			},
			want: templatedResult,
			wantValues: &helmify.Values{
				"mydep": map[string]interface{}{
					"nodeSelector": map[string]string{},
					"tolerations":  []interface{}{},
					"topologySpreadConstraints": []v1.TopologySpreadConstraint{
						{
							MaxSkew:           0,
							TopologyKey:       "trtr",
							WhenUnsatisfiable: "test",
						},
					},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ProcessSpecMap("mydep", tt.specMap, tt.values, true)
			require.Contains(t, got, tt.want)
			require.Equal(t, *tt.wantValues, *tt.values)
		})
	}
}
