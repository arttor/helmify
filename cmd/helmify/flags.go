package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/arttor/helmify/pkg/config"
)

const helpText = `Helmify parses kubernetes resources from std.in and converts it to a Helm chart.

Example 1: 'kustomize build <kustomize_dir> | helmify mychart' 
  - will create 'mychart' directory with Helm chart from kustomize output.

Example 2: 'cat my-app.yaml | helmify mychart' 
  - will create 'mychart' directory with Helm chart from yaml file.

Example 3: 'awk 'FNR==1 && NR!=1  {print "---"}{print}' /my_directory/*.yaml | helmify mychart' 
  - will create 'mychart' directory with Helm chart from all yaml files in my_directory directory.

Usage:
  helmify [flags] CHART_NAME  -  CHART_NAME is optional. Default is 'chart'. Can be a directory, e.g. 'deploy/charts/mychart'.

Flags:
`

// ReadFlags command-line flags into app config.
func ReadFlags() config.Config {
	result := config.Config{}
	var h, help, version bool
	flag.BoolVar(&h, "h", false, "Print help. Example: helmify -h")
	flag.BoolVar(&help, "help", false, "Print help. Example: helmify -help")
	flag.BoolVar(&version, "version", false, "Print helmify version. Example: helmify -version")
	flag.BoolVar(&result.Verbose, "v", false, "Enable verbose output (print WARN & INFO). Example: helmify -v")
	flag.BoolVar(&result.VeryVerbose, "vv", false, "Enable very verbose output. Same as verbose but with DEBUG. Example: helmify -vv")
	flag.Parse()
	if h || help {
		fmt.Print(helpText)
		flag.PrintDefaults()
		os.Exit(0)
	}
	if version {
		printVersion()
		os.Exit(0)
	}
	name := flag.Arg(0)
	if name != "" {
		result.ChartName = filepath.Base(name)
		result.ChartDir = filepath.Dir(name)
	}

	return result
}
