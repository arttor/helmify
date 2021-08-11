package config

const DefaultChartName = "chart"

var SupportedResources = map[string]struct{}{"crd": {}, "deployment": {}, "rbac": {}, "secret": {}}

type Config struct {
	ChartName   string
	ProcessOnly []string
	SkipValues  bool
	Verbose     bool
}
