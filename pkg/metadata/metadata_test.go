package metadata

import (
	"fmt"
	"testing"

	"github.com/arttor/helmify/internal"
	"github.com/stretchr/testify/assert"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

const res = `apiVersion: v1
kind: Secret
metadata:
  name: %s
  namespace: %s`

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

func Test_Service(t *testing.T) {
	t.Run("load ns from object", func(t *testing.T) {
		obj := createRes("name", "ns")
		testSvc := New("")
		testSvc.Load(obj)
		assert.Equal(t, "ns", testSvc.Namespace())
		testSvc.Load(internal.TestNs)
		assert.Equal(t, internal.TestNsName, testSvc.Namespace())
	})
	t.Run("get chart name", func(t *testing.T) {
		testSvc := New("name")
		assert.Equal(t, "name", testSvc.ChartName())
	})
	t.Run("trim common prefix abc", func(t *testing.T) {
		testSvc := New("")
		testSvc.Load(createRes("abc-name1", "ns"))
		testSvc.Load(createRes("abc-name2", "ns"))
		testSvc.Load(createRes("abc-service", "ns"))

		assert.Equal(t, "name1", testSvc.TrimName("abc-name1"))
		assert.Equal(t, "name2", testSvc.TrimName("abc-name2"))
		assert.Equal(t, "service", testSvc.TrimName("abc-service"))
	})
	t.Run("trim common prefix: no common", func(t *testing.T) {
		testSvc := New("")
		testSvc.Load(createRes("name1", "ns"))
		testSvc.Load(createRes("abc", "ns"))
		testSvc.Load(createRes("service", "ns"))

		assert.Equal(t, "name1", testSvc.TrimName("name1"))
		assert.Equal(t, "abc", testSvc.TrimName("abc"))
		assert.Equal(t, "service", testSvc.TrimName("service"))
	})
	t.Run("template name", func(t *testing.T) {
		testSvc := New("chart-name")
		testSvc.Load(createRes("abc", "ns"))
		templated := testSvc.TemplatedName("abc")
		assert.Equal(t, `{{ include "chart-name.fullname" . }}-abc`, templated)
	})
	t.Run("template name: not process unknown name", func(t *testing.T) {
		testSvc := New("chart-name")
		testSvc.Load(createRes("abc", "ns"))
		assert.Equal(t, "qwe", testSvc.TemplatedName("qwe"))
		assert.NotEqual(t, "abc", testSvc.TemplatedName("abc"))
	})
}

func createRes(name, ns string) *unstructured.Unstructured {
	objYaml := fmt.Sprintf(res, name, ns)
	return internal.GenerateObj(objYaml)
}
