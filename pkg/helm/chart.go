package helm

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/arttor/helmify/pkg/cluster"
	"github.com/arttor/helmify/pkg/helmify"

	"github.com/sirupsen/logrus"

	"gopkg.in/yaml.v3"
	k8syaml "sigs.k8s.io/yaml"
)

// NewOutput creates interface to dump processed input to filesystem in Helm chart format.
func NewOutput() helmify.Output {
	return &output{}
}

type output struct{}

// Create a helm chart in the current directory:
// chartName/
//
//	├── .helmignore   	# Contains patterns to ignore when packaging Helm charts.
//	├── Chart.yaml    	# Information about your chart
//	├── values.yaml   	# The default values for your templates
//	└── templates/    	# The template files
//	    └── _helpers.tp   # Helm default template partials
//
// Overwrites existing values.yaml and templates in templates dir on every run.
func (o output) Create(chartDir, chartName string, crd bool, certManagerAsSubchart bool, certManagerVersion string, certManagerInstallCRD bool, templates []helmify.Template, filenames []string) error {
	err := initChartDir(chartDir, chartName, crd, certManagerAsSubchart, certManagerVersion)
	if err != nil {
		return err
	}
	// group templates into files
	files := map[string][]helmify.Template{}
	values := helmify.Values{}
	values[cluster.DomainKey] = cluster.DefaultDomain
	for i, template := range templates {
		file := files[filenames[i]]
		file = append(file, template)
		files[filenames[i]] = file
		err = values.Merge(template.Values())
		if err != nil {
			return err
		}
	}
	cDir := filepath.Join(chartDir, chartName)
	for filename, tpls := range files {
		err = overwriteTemplateFile(filename, cDir, crd, tpls)
		if err != nil {
			return err
		}
	}
	err = overwriteValuesFile(cDir, values, certManagerAsSubchart, certManagerInstallCRD)
	if err != nil {
		return err
	}
	return nil
}

func overwriteTemplateFile(filename, chartDir string, crd bool, templates []helmify.Template) error {
	// pull in crd-dir setting and siphon crds into folder
	var subdir string
	if strings.Contains(filename, "crd") && crd {
		subdir = "crds"
		// create "crds" if not exists
		if _, err := os.Stat(filepath.Join(chartDir, "crds")); os.IsNotExist(err) {
			err = os.MkdirAll(filepath.Join(chartDir, "crds"), 0750)
			if err != nil {
				return fmt.Errorf("%w: unable create crds dir", err)
			}
		}
	} else {
		subdir = "templates"
	}
	file := filepath.Join(chartDir, subdir, filename)
	f, err := os.OpenFile(file, os.O_APPEND|os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		return fmt.Errorf("%w: unable to open %s", err, file)
	}
	defer f.Close()
	for i, t := range templates {
		logrus.WithField("file", file).Debug("writing a template into")
		err = t.Write(f)
		if err != nil {
			return fmt.Errorf("%w: unable to write into %s", err, file)
		}
		if i != len(templates)-1 {
			_, err = f.Write([]byte("\n---\n"))
			if err != nil {
				return fmt.Errorf("%w: unable to write into %s", err, file)
			}
		}
	}
	if len(templates) != 0 {
		_, err = f.Write([]byte("\n"))
		if err != nil {
			return fmt.Errorf("%w: unable to write newline into %s", err, file)
		}
	}
	logrus.WithField("file", file).Info("overwritten")
	return nil
}

func overwriteValuesFile(chartDir string, values helmify.Values, certManagerAsSubchart bool, certManagerInstallCRD bool) error {
	if certManagerAsSubchart {
		_, err := values.Add(certManagerInstallCRD, "certmanager", "installCRDs")
		if err != nil {
			return fmt.Errorf("%w: unable to add cert-manager.installCRDs", err)
		}

		_, err = values.Add(true, "certmanager", "enabled")
		if err != nil {
			return fmt.Errorf("%w: unable to add cert-manager.enabled", err)
		}
	}
	// Use custom marshaler to preserve desired logical ordering
	res, err := marshalOrdered(values)
	if err != nil {
		return fmt.Errorf("%w: unable to write marshal values.yaml", err)
	}

	file := filepath.Join(chartDir, "values.yaml")
	err = os.WriteFile(file, res, 0600)
	if err != nil {
		return fmt.Errorf("%w: unable to write values.yaml", err)
	}
	logrus.WithField("file", file).Info("overwritten")
	return nil
}

func marshalOrdered(v interface{}) ([]byte, error) {
	var b strings.Builder
	enc := yaml.NewEncoder(&b)
	enc.SetIndent(2)
	node := toNode(v, 0)
	err := enc.Encode(node)
	res := b.String()
	res = strings.ReplaceAll(res, "\n  # helmify-newline\n", "\n\n")
	return []byte(res), err
}

func toNode(v interface{}, depth int) *yaml.Node {
	switch val := v.(type) {
	case map[string]interface{}:
		content := make([]*yaml.Node, 0, len(val)*2)
		keys := make([]string, 0, len(val))
		for k := range val {
			keys = append(keys, k)
		}

		sort.Slice(keys, func(i, j int) bool {
			pi := getPriority(keys[i], val[keys[i]])
			pj := getPriority(keys[j], val[keys[j]])
			if pi != pj {
				return pi < pj
			}
			return keys[i] < keys[j]
		})

		var prevPriority int
		for _, k := range keys {
			p := getPriority(k, val[k])
			keyNode := &yaml.Node{Kind: yaml.ScalarNode, Value: k}
			if depth == 1 && prevPriority != 0 && p != prevPriority {
				keyNode.HeadComment = "helmify-newline"
			}
			content = append(content, keyNode)
			content = append(content, toNode(val[k], depth+1))
			prevPriority = p
		}
		return &yaml.Node{Kind: yaml.MappingNode, Content: content}
	case []interface{}:
		content := make([]*yaml.Node, len(val))
		for i, item := range val {
			content[i] = toNode(item, depth+1)
		}
		return &yaml.Node{Kind: yaml.SequenceNode, Content: content}
	case helmify.Values:
		return toNode(map[string]interface{}(val), depth)
	default:
		var node yaml.Node
		b, _ := k8syaml.Marshal(val)
		_ = yaml.Unmarshal(b, &node)
		if len(node.Content) > 0 {
			return node.Content[0]
		}
		return &node
	}
}

func getPriority(key string, value interface{}) int {
	// 1. Workload (Priority 1)
	workloadKeys := map[string]bool{
		"image": true, "repository": true, "tag": true, "imagePullPolicy": true,
		"replicas": true, "strategy": true, "resources": true, "nodeSelector": true,
		"tolerations": true, "topologySpreadConstraints": true, "revisionHistoryLimit": true,
		"podLabels": true, "podAnnotations": true, "podSecurityContext": true,
	}
	if workloadKeys[key] {
		return 1
	}

	// 2. Identity (Priority 2)
	if key == "serviceAccount" {
		return 2
	}

	// 5. Networking (Priority 5-7)
	if key == "route" {
		return 5
	}
	if key == "ingress" {
		return 6
	}
	if key == "service" || key == "type" || key == "ports" || key == "clusterIP" || key == "loadBalancerIP" {
		return 7
	}

	// 10+ Security & Extensions
	if strings.Contains(key, "role") || strings.Contains(key, "Role") {
		return 10
	}
	if key == "webhook" {
		return 11
	}
	if key == "crds" {
		return 12
	}

	// 3-4 ConfigMap vs Secret Heuristic
	if str, ok := value.(string); ok {
		if str == "" {
			return 4 // Secret (Priority 4)
		}
		return 3 // ConfigMap (Priority 3)
	}

	return 50 // Others
}
