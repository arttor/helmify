package config

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
