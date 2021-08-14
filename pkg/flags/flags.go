package flags

import (
	"flag"
	"fmt"
	"github.com/arttor/helmify/pkg/config"
	"os"
)

const helpText = `Helmify parses kubernetes resources from std.in and converts it to a Helm chart.
Use this command as a pipe for 'kustomize build' command output.

Example: 'kustomize build <kustomize_dir> | helmify mychart' 
  - will create 'mychart' directory with Helm chart.

Usage:
  helmify [flags] CHART_NAME  -  CHART_NAME is optional. Default is 'chart'.

Flags:
`

func Read() config.Config {
	result := config.Config{}
	var h, help bool
	flag.BoolVar(&h, "h", false, "Print help. Example: helmify -h")
	flag.BoolVar(&help, "help", false, "Print help. Example: helmify -help")
	flag.BoolVar(&result.Verbose, "v", false, "Enable verbose output (print WARN & INFO). Example: helmify -v")
	flag.BoolVar(&result.VeryVerbose, "vv", false, "Enable very verbose output. Same as verbose but with DEBUG. Example: helmify -vv")
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
	return result
}
