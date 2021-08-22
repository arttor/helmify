package helmify

import "testing"

func TestValues_Add(t *testing.T) {
	t.Run("", func(t *testing.T) {
		v := Values{}
		template, err := v.Add("", []string{""})

	})

	type args struct {
		value interface{}
		name  []string
	}
	tests := []struct {
		name    string
		v       Values
		args    args
		want    string
		wantErr bool
	}{
		{
			name:    "",
			v:       nil,
			args:    args{},
			want:    "",
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.v.Add(tt.args.value, tt.args.name)
			if (err != nil) != tt.wantErr {
				t.Errorf("Add() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("Add() got = %v, want %v", got, tt.want)
			}
		})
	}
}
