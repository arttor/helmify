package config

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestConfig_Validate(t *testing.T) {
	type fields struct {
		ChartName   string
		Verbose     bool
		VeryVerbose bool
	}
	tests := []struct {
		name    string
		fields  fields
		wantErr bool
	}{
		{name: "valid", fields: fields{ChartName: ""}, wantErr: false},
		{name: "valid", fields: fields{ChartName: "my.chart123"}, wantErr: false},
		{name: "valid", fields: fields{ChartName: "my-chart123"}, wantErr: false},
		{name: "invalid", fields: fields{ChartName: "my_chart123"}, wantErr: true},
		{name: "invalid", fields: fields{ChartName: "my char123t"}, wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &Config{
				ChartName:   tt.fields.ChartName,
				Verbose:     tt.fields.Verbose,
				VeryVerbose: tt.fields.VeryVerbose,
			}
			if err := c.Validate(); (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
	t.Run("chart name not set", func(t *testing.T) {
		c := &Config{}
		err := c.Validate()
		assert.NoError(t, err)
		assert.Equal(t, defaultChartName, c.ChartName)
	})
	t.Run("chart name set", func(t *testing.T) {
		c := &Config{ChartName: "test"}
		err := c.Validate()
		assert.NoError(t, err)
		assert.Equal(t, "test", c.ChartName)
	})
}
