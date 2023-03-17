package format

import "testing"

func TestRemoveTrailingWhitespaces(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{
			name: "",
			in:   `abc   `,
			want: `abc`,
		},
		{
			name: "",
			in: `abc   
edf`,
			want: `abc
edf`,
		},
		{
			name: "",
			in: `abc   
edf   `,
			want: `abc
edf`,
		},
		{
			name: "",
			in: `abc   .
edf   .`,
			want: `abc   .
edf   .`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := RemoveTrailingWhitespaces(tt.in); got != tt.want {
				t.Errorf("RemoveTrailingWhitespaces() = %v, want %v", got, tt.want)
			}
		})
	}
}
