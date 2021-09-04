package metadata

import "testing"

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
