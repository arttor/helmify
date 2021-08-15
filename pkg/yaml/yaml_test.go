package yaml

import (
	"reflect"
	"testing"
)

func TestIndent(t *testing.T) {
	type args struct {
		content []byte
		n       int
	}
	tests := []struct {
		name string
		args args
		want []byte
	}{
		{
			name: "negative",
			args: args{[]byte("a"), -1},
			want: []byte("a"),
		},
		{
			name: "none",
			args: args{[]byte("a"), 0},
			want: []byte("a"),
		},
		{
			name: "one",
			args: args{[]byte("a"), 1},
			want: []byte(" a"),
		},
		{
			name: "two",
			args: args{[]byte("a"), 2},
			want: []byte("  a"),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := Indent(tt.args.content, tt.args.n); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Indent() = %v, want %v", got, tt.want)
			}
		})
	}
}
