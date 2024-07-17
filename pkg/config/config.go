package config

import (
	"fmt"
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
	// crd-dir set true to enable crd folder.
	Crd bool
	// ImagePullSecrets flag
	ImagePullSecrets bool
	// GenerateDefaults enables the generation of empty values placeholders for common customization options of helm chart
	// current generated values: tolerances, node selectors, topology constraints
	GenerateDefaults bool
	// CertManagerAsSubchart enables the generation of a subchart for cert-manager
	CertManagerAsSubchart bool
	// CertManagerVersion sets cert-manager version in dependency
	CertManagerVersion string
	// CertManagerVersion enables installation of cert-manager CRD
	CertManagerInstallCRD bool
	// Files - directories or files with k8s manifests
	Files []string
	// FilesRecursively read Files recursively
	FilesRecursively bool
	// OriginalName retains Kubernetes resource's original name
	OriginalName bool
	// PreserveNs retains the namespaces on the Kubernetes manifests
	PreserveNs bool
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
		return fmt.Errorf("invalid chart name %s", c.ChartName)
	}
	return nil
}
