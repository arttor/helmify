package security_context

import (
	"testing"

	"github.com/arttor/helmify/pkg/helmify"
	"github.com/stretchr/testify/assert"
)

func TestProcessContainerSecurityContext(t *testing.T) {
	type args struct {
		nameCamel string
		specMap   map[string]interface{}
		values    *helmify.Values
	}
	tests := []struct {
		name string
		args args
		want *helmify.Values
	}{
		{
			name: "test with empty specMap",
			args: args{
				nameCamel: "someResourceName",
				specMap:   map[string]interface{}{},
				values:    &helmify.Values{},
			},
			want: &helmify.Values{},
		},
		{
			name: "test with single container",
			args: args{
				nameCamel: "someResourceName",
				specMap: map[string]interface{}{
					"containers": []interface{}{
						map[string]interface{}{
							"name": "SomeContainerName",
							"securityContext": map[string]interface{}{
								"privileged": true,
							},
						},
					},
				},
				values: &helmify.Values{},
			},
			want: &helmify.Values{
				"someResourceName": map[string]interface{}{
					"someContainerName": map[string]interface{}{
						"containerSecurityContext": map[string]interface{}{
							"privileged": true,
						},
					},
				},
			},
		},
		{
			name: "test with multiple containers",
			args: args{
				nameCamel: "someResourceName",
				specMap: map[string]interface{}{
					"containers": []interface{}{
						map[string]interface{}{
							"name": "FirstContainer",
							"securityContext": map[string]interface{}{
								"privileged": true,
							},
						},
						map[string]interface{}{
							"name": "SecondContainer",
							"securityContext": map[string]interface{}{
								"allowPrivilegeEscalation": true,
							},
						},
					},
				},
				values: &helmify.Values{},
			},
			want: &helmify.Values{
				"someResourceName": map[string]interface{}{
					"firstContainer": map[string]interface{}{
						"containerSecurityContext": map[string]interface{}{
							"privileged": true,
						},
					},
					"secondContainer": map[string]interface{}{
						"containerSecurityContext": map[string]interface{}{
							"allowPrivilegeEscalation": true,
						},
					},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ProcessContainerSecurityContext(tt.args.nameCamel, tt.args.specMap, tt.args.values, 8)
			assert.Equal(t, tt.want, tt.args.values)
		})
	}
}

func Test_setSecContextValue(t *testing.T) {
	type args struct {
		resourceName            string
		containerName           string
		castedContainer         map[string]interface{}
		values                  *helmify.Values
		fieldName               string
		useRenderedHelmTemplate bool
	}
	tests := []struct {
		name string
		args args
		want *helmify.Values
	}{
		{
			name: "simple test with single container and single value",
			args: args{
				resourceName:  "someResource",
				containerName: "someContainer",
				castedContainer: map[string]interface{}{
					"securityContext": map[string]interface{}{
						"someField": "someValue",
					},
				},
				values:                  &helmify.Values{},
				fieldName:               "someField",
				useRenderedHelmTemplate: false,
			},
			want: &helmify.Values{
				"someResource": map[string]interface{}{
					"someContainer": map[string]interface{}{
						"containerSecurityContext": map[string]interface{}{
							"someField": "someValue",
						},
					},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			setSecContextValue(tt.args.resourceName, tt.args.containerName, tt.args.castedContainer, tt.args.values, 8)
			assert.Equal(t, tt.want, tt.args.values)
		})
	}
}
