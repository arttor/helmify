package flags

import (
	"flag"
	"fmt"
	"github.com/arttor/helmify/pkg/config"
	"os"
	"strings"
)

const helpText = `Helmify parses kubernetes resources from std.in and converts it to a Helm chart.
Use this command as a pipe for 'kustomize build' command output.

Example: 'kustomize build <kustomize_dir> | helmify mychart' 
  - will create 'mychart' directory with Helm chart.

Usage:
  helmify CHART_NAME [flags]  -  CHART_NAME is optional. Default is 'chart'.

Flags:
`

func Read() config.Config {
	result := config.Config{}
	var only string
	var h, help bool
	flag.BoolVar(&h, "h", false, "Print help. Example: helmify -h")
	flag.BoolVar(&help, "help", false, "Print help. Example: helmify -help")
	flag.BoolVar(&result.Verbose, "v", false, "Enable verbose output. Example: helmify -v")
	flag.StringVar(&only, "only", "", "A comma-separated list of processed kubernetes resources."+
		"\nUseful if you want to update certain objects of existing chart.\n"+
		"Supported values: crd,deployment,rbac. Example: helmify -only=crd,rbac")
	flag.BoolVar(&result.SkipValues, "skip-val", true, "Set the flag if you don't want"+
		" to update Helm values.yaml.\n Example: helmify -skip-val")
	flag.Parse()
	if h || help {
		fmt.Print(helpText)
		flag.PrintDefaults()
		os.Exit(0)
	}
	result.ChartName = flag.Arg(0)
	if result.ChartName == "" {
		result.ChartName = config.DefaultChartName
	}
	onlyResources := strings.Split(only, ",")
	for _, res := range onlyResources {
		_, contains := config.SupportedResources[res]
		if contains {
			result.ProcessOnly = append(result.ProcessOnly, res)
		}
	}
	return result
}
