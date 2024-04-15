package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/arttor/helmify/pkg/config"
)

const helpText = `Helmify parses kubernetes resources from std.in and converts it to a Helm chart.

Example 1: 'kustomize build <kustomize_dir> | helmify mychart' 
  - will create 'mychart' directory with Helm chart from kustomize output.

Example 2: 'cat my-app.yaml | helmify mychart' 
  - will create 'mychart' directory with Helm chart from yaml file.

Example 3: 'helmify -f ./test_data/dir  mychart' 
  - will scan directory ./test_data/dir for files with k8s manifests and create 'mychart' directory with Helm chart.

Example 4: 'helmify -f ./test_data/dir -r  mychart' 
  - will scan directory ./test_data/dir recursively and  create 'mychart' directory with Helm chart.

Example 5: 'helmify -f ./test_data/dir -f ./test_data/sample-app.yaml -f ./test_data/dir/another_dir  mychart' 
  - will scan provided multiple files and directories and  create 'mychart' directory with Helm chart.

Example 6: 'awk 'FNR==1 && NR!=1  {print "---"}{print}' /my_directory/*.yaml | helmify mychart' 
  - will create 'mychart' directory with Helm chart from all yaml files in my_directory directory.

Usage:
  helmify [flags] CHART_NAME  -  CHART_NAME is optional. Default is 'chart'. Can be a directory, e.g. 'deploy/charts/mychart'.

Flags:
`

type arrayFlags []string

func (i *arrayFlags) String() string {
	if i == nil || len(*i) == 0 {
		return ""
	}
	return strings.Join(*i, ", ")
}

func (i *arrayFlags) Set(value string) error {
	*i = append(*i, value)
	return nil
}

// ReadFlags command-line flags into app config.
func ReadFlags() config.Config {
	files := arrayFlags{}
	result := config.Config{}
	var h, help, version, crd bool
	flag.BoolVar(&h, "h", false, "Print help. Example: helmify -h")
	flag.BoolVar(&help, "help", false, "Print help. Example: helmify -help")
	flag.BoolVar(&version, "version", false, "Print helmify version. Example: helmify -version")
	flag.BoolVar(&result.Verbose, "v", false, "Enable verbose output (print WARN & INFO). Example: helmify -v")
	flag.BoolVar(&result.VeryVerbose, "vv", false, "Enable very verbose output. Same as verbose but with DEBUG. Example: helmify -vv")
	flag.BoolVar(&crd, "crd-dir", false, "Enable crd install into 'crds' directory.\nWarning: CRDs placed in 'crds' directory will not be templated by Helm.\nSee https://helm.sh/docs/chart_best_practices/custom_resource_definitions/#some-caveats-and-explanations\nExample: helmify -crd-dir")
	flag.BoolVar(&result.ImagePullSecrets, "image-pull-secrets", false, "Allows the user to use existing secrets as imagePullSecrets in values.yaml")
	flag.BoolVar(&result.GenerateDefaults, "generate-defaults", false, "Allows the user to add empty placeholders for tipical customization options in values.yaml. Currently covers: topology constraints, node selectors, tolerances")
	flag.BoolVar(&result.CertManagerAsSubchart, "cert-manager-as-subchart", false, "Allows the user to add cert-manager as a subchart")
	flag.StringVar(&result.CertManagerVersion, "cert-manager-version", "v1.12.2", "Allows the user to specify cert-manager subchart version. Only useful with cert-manager-as-subchart.")
	flag.BoolVar(&result.FilesRecursively, "r", false, "Scan dirs from -f option recursively")
	flag.BoolVar(&result.OriginalName, "original-name", false, "Retains Kubernetes resource's original name.")
	flag.Var(&files, "f", "File or directory containing k8s manifests")

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
	if crd {
		result.Crd = crd
	}
	result.Files = files
	return result
}
