package processor

import (
	"github.com/arttor/helmify/internal"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"testing"
)

func Test_commonPrefix(t *testing.T) {
	type args struct {
		left, right string
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{
			name: "left is a prefix of right",
			args: args{left: "test", right: "testimony"},
			want: "test",
		},
		{
			name: "common prefix",
			args: args{left: "testimony", right: "testicle"},
			want: "testi",
		},
		{
			name: "no common",
			args: args{left: "testimony", right: "abc"},
			want: "",
		},
		{
			name: "right is empty",
			args: args{left: "testimony", right: ""},
			want: "",
		},
		{
			name: "left is empty",
			args: args{left: "", right: "abc"},
			want: "",
		},
		{
			name: "both are empty",
			args: args{left: "", right: ""},
			want: "",
		},
		{
			name: "unicode",
			args: args{left: "багет", right: "багаж"},
			want: "баг",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := commonPrefix(tt.args.left, tt.args.right); got != tt.want {
				t.Errorf("commonPrefix() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestExtractOperatorName(t *testing.T) {
	createObj := func(name string) *unstructured.Unstructured {
		return &unstructured.Unstructured{
			Object: map[string]interface{}{
				"metadata": map[string]interface{}{
					"name": name,
				},
			},
		}
	}
	type args struct {
		obj      *unstructured.Unstructured
		prevName string
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{
			name: "have common prefix",
			args: args{createObj("blaAAA"), "blaBBB"},
			want: "bla",
		},
		{
			name: "trim '-'",
			args: args{createObj("bla-AAA"), "bla-BBB"},
			want: "bla",
		},
		{
			name: "if previous is empty return new",
			args: args{createObj("bla"), ""},
			want: "bla",
		},
		{
			name: "if no common prefix return previous",
			args: args{createObj("xyz"), "abc"},
			want: "abc",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ExtractOperatorName(tt.args.obj, tt.args.prevName); got != tt.want {
				t.Errorf("ExtractOperatorName() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestExtractOperatorNamespace(t *testing.T) {
	configmap := `apiVersion: v1
kind: ConfigMap
metadata:
  name: my-operator-manager-config
  namespace: my-operator-system`
	type args struct {
		obj *unstructured.Unstructured
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{
			name: "namespace",
			args: args{internal.TestNs},
			want: internal.TestNsName,
		},
		{
			name: "not a namespace",
			args: args{internal.GenerateObj(configmap)},
			want: "",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ExtractOperatorNamespace(tt.args.obj); got != tt.want {
				t.Errorf("ExtractOperatorNamespace() = %v, want %v", got, tt.want)
			}
		})
	}
}
