package helm

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/arttor/helmify/pkg/cluster"
	"github.com/arttor/helmify/pkg/helmify"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"

	"sigs.k8s.io/yaml"
)

// NewOutput creates interface to dump processed input to filesystem in Helm chart format.
func NewOutput() helmify.Output {
	return &output{}
}

type output struct{}

// Create a helm chart in the current directory:
// chartName/
//    ├── .helmignore   	# Contains patterns to ignore when packaging Helm charts.
//    ├── Chart.yaml    	# Information about your chart
//    ├── values.yaml   	# The default values for your templates
//    └── templates/    	# The template files
//        └── _helpers.tp   # Helm default template partials
// Overwrites existing values.yaml and templates in templates dir on every run.
func (o output) Create(chartDir, chartName string, crd bool, templates []helmify.Template) error {
	err := initChartDir(chartDir, chartName, crd)
	if err != nil {
		return err
	}
	// group templates into files
	files := map[string][]helmify.Template{}
	values := helmify.Values{}
	values[cluster.DomainKey] = cluster.DefaultDomain
	for _, template := range templates {
		file := files[template.Filename()]
		file = append(file, template)
		files[template.Filename()] = file
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
	err = overwriteValuesFile(cDir, values)
	if err != nil {
		return err
	}
	return nil
}

func overwriteTemplateFile(filename, chartDir string, crd bool, templates []helmify.Template) error {
	// pull in crd-install setting and siphon crds into folder
	var subdir string
	if strings.Contains(filename, "crd") && crd {
		subdir = "crds"
	} else {
		subdir = "templates"
	}
	file := filepath.Join(chartDir, subdir, filename)
	f, err := os.OpenFile(file, os.O_APPEND|os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		return errors.Wrap(err, "unable to open "+file)
	}
	defer f.Close()
	for i, t := range templates {
		logrus.WithField("file", file).Debug("writing a template into")
		err = t.Write(f)
		if err != nil {
			return errors.Wrap(err, "unable to write into "+file)
		}
		if i != len(templates)-1 {
			_, err = f.Write([]byte("\n---\n"))
			if err != nil {
				return errors.Wrap(err, "unable to write into "+file)
			}
		}
	}
	logrus.WithField("file", file).Info("overwritten")
	return nil
}

func overwriteValuesFile(chartDir string, values helmify.Values) error {
	res, err := yaml.Marshal(values)
	if err != nil {
		return errors.Wrap(err, "unable to write marshal values.yaml")
	}
	file := filepath.Join(chartDir, "values.yaml")
	err = ioutil.WriteFile(file, res, 0600)
	if err != nil {
		return errors.Wrap(err, "unable to write values.yaml")
	}
	logrus.WithField("file", file).Info("overwritten")
	return nil
}
