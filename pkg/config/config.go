package config

import (
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/util/validation"
)

// DefaultChartName - default name for a helm chart directory.
const DefaultChartName = "chart"

// Config for Helmify application.
type Config struct {
	// ChartName overrides DefaultChartName.
	ChartName string
	// Verbose set true to see WARN and INFO logs.
	Verbose bool
	// VeryVerbose set true to see WARN, INFO, and DEBUG logs.
	VeryVerbose bool
}

func (c Config) Validate() error {
	err := validation.IsDNS1123Subdomain(c.ChartName)
	if err != nil {
		for _, e := range err {
			logrus.Errorf("Invalid chart name %s", e)
		}
		return errors.Errorf("Invalid chart name %s", c.ChartName)
	}
	return nil
}
