package main

import (
	"flag"
	"github.com/arttor/helmify/pkg/app"
	"github.com/arttor/helmify/pkg/config"
	"github.com/sirupsen/logrus"
	"os"
	"strings"
)

const defaultChartName = "chart"

var supportedResources = map[string]struct{}{"crd": {}, "deployment": {}, "rbac": {}}

func main() {
	conf := readFlags()
	stat, err := os.Stdin.Stat()
	if err != nil {
		logrus.WithError(err).Error("stdin error")
		os.Exit(1)
	}
	if (stat.Mode() & os.ModeCharDevice) != 0 {
		logrus.Warn("no data piped in stdin")
		os.Exit(1)
	}
	if err := app.Start(os.Stdin, conf); err != nil {
		logrus.WithError(err).Error("helmify finished with error")
		os.Exit(1)
	}
}

func readFlags() config.Config {
	result := config.Config{}
	var only string
	flag.StringVar(&only, "only", "chart", "A comma-separated list of processed kubernetes resources."+
		" Useful if you want to update certain objects of existing chart. "+
		"Supported values: crd,deployment,rbac. Example: helmify -only=crd,rbac")
	flag.BoolVar(&result.UpdateValues, "values", true, "Set false if you don't want"+
		" to update Helm values.yaml. The default value is true. Example: helmify -values=false")
	flag.Parse()
	result.ChartName = flag.Arg(0)
	if result.ChartName == "" {
		result.ChartName = defaultChartName
	}
	onlyResources := strings.Split(only, ",")
	for _, res := range onlyResources {
		_, contains := supportedResources[res]
		if contains {
			result.ProcessOnly = append(result.ProcessOnly, res)
		}
	}
	return result
}
