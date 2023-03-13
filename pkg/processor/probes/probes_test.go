package probes

import (
	"testing"

	"github.com/arttor/helmify/pkg/helmify"
	"github.com/stretchr/testify/require"
	"sigs.k8s.io/yaml"
)

func Test_setProbesTemplates(t *testing.T) {

	tests := []struct {
		name           string
		deploymentName string
		container      map[string]interface{}
		wantMap        string
		wantValue      string
		wantErr        bool
	}{
		{
			name:           "no probe no data generated",
			deploymentName: "test",
			container: map[string]interface{}{
				"name": "mycontainer",
			},
			wantMap: "",
			wantErr: false,
		},
		{
			name:           "readinessProbe probe",
			deploymentName: "test",
			container: map[string]interface{}{
				"name": "mycontainer",
				readinessProbe: map[string]interface{}{
					"timeoutSeconds": "1",
					"periodSeconds":  "20",
				},
			},
			wantMap: "\n{{- if .Values.test.mycontainer.readinessProbe }}\n" +
				"readinessProbe: {{- include \"tplvalues.render\" (dict \"value\" .Values.test.mycontainer.readinessProbe \"context\" $) | nindent 10 }}\n {{- else }}\n" +
				"readinessProbe:\n" +
				" periodSeconds: \"20\"\n" +
				" timeoutSeconds: \"1\"\n" +
				"{{- end }}",
			wantValue: "readinessProbe:\n  periodSeconds: \"20\"\n  timeoutSeconds: \"1\"\n",
			wantErr:   false,
		},
		{
			name:           "add livenessProbe probe",
			deploymentName: "test",
			container: map[string]interface{}{
				"name": "mycontainer",
				livenessProbe: map[string]interface{}{
					"timeoutSeconds": "14",
					"periodSeconds":  "2",
				},
			},
			wantMap: "{{- if .Values.test.mycontainer.livenessProbe }}\n" +
				"livenessProbe: {{- include \"tplvalues.render\" (dict \"value\" .Values.test.mycontainer.livenessProbe \"context\" $) | nindent 10 }}\n" +
				" {{- else }}\nlivenessProbe:\n" +
				" periodSeconds: \"2\"\n" +
				" timeoutSeconds: \"14\"\n" +
				"{{- end }}",
			wantValue: "livenessProbe:\n  periodSeconds: \"2\"\n  timeoutSeconds: \"14\"\n",
			wantErr:   false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v := make(helmify.Values)
			res, err := setProbesTemplates(tt.deploymentName, tt.container, &v)
			require.True(t, (err != nil) == tt.wantErr)

			require.Contains(t, res, tt.wantMap)
			if tt.wantValue != "" {
				val := (v)["test"].(map[string]interface{})["mycontainer"]
				t.Log("VAL", val)
				b, err := yaml.Marshal(val)
				require.Nil(t, err)
				require.Contains(t, string(b), tt.wantValue)
			} else {
				require.Empty(t, v)
			}
		})
	}
}
