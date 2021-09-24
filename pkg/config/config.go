package config

import (
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/util/validation"
)

// defaultChartName - default name for a helm chart directory.
const defaultChartName = "chart"

// Config for Helmify application.
type Config struct {
	// ChartName name of the Helm chart and its base directory where Chart.yaml is located.
	ChartName string
	// ChartDir - optional path to chart dir. Full chart path will be: ChartDir/ChartName/Chart.yaml.
	ChartDir string
	// Verbose set true to see WARN and INFO logs.
	Verbose bool
	// VeryVerbose set true to see WARN, INFO, and DEBUG logs.
	VeryVerbose bool
}

func (c *Config) Validate() error {
	if c.ChartName == "" {
		logrus.Infof("Chart name is not set. Using default name '%s", defaultChartName)
		c.ChartName = defaultChartName
	}
	err := validation.IsDNS1123Subdomain(c.ChartName)
	if err != nil {
		for _, e := range err {
			logrus.Errorf("Invalid chart name %s", e)
		}
		return errors.Errorf("Invalid chart name %s", c.ChartName)
	}
	return nil
}
