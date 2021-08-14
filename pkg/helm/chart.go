package helm

import (
	"github.com/arttor/helmify/pkg/helmify"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"io/ioutil"
	"os"
	"path/filepath"
	"sigs.k8s.io/yaml"
)

// NewOutput creates interface to dump processed input to filesystem in Helm chart format
func NewOutput() helmify.Output {
	return &output{}
}

type output struct{}

// Create a helm chart in filesystem current directory:
// chartName/
//    ├── .helmignore   	# Contains patterns to ignore when packaging Helm charts.
//    ├── Chart.yaml    	# Information about your chart
//    ├── values.yaml   	# The default values for your templates
//    └── templates/    	# The template files
//        └── _helpers.tp   # Helm default template partials
// Overwrites existing values.yaml and templates/
func (o *output) Create(chartInfo helmify.ChartInfo, templates []helmify.Template) error {
	err := o.init(chartInfo.ChartName, chartInfo.OperatorName)
	if err != nil {
		return err
	}
	// combine templates into files
	files := map[string][]helmify.Template{}
	values := helmify.Values{}
	for _, template := range templates {
		file := files[template.Filename()]
		file = append(file, template)
		files[template.Filename()] = file
		err = values.Merge(template.Values())
		if err != nil {
			return err
		}
	}
	// overwrite values.yaml
	err = o.writeValues(chartInfo.ChartName, values)
	if err != nil {
		return err
	}
	logrus.Infof("'./%s/values.yaml' overwritten", chartInfo.ChartName)
	// overwrite templates files
	for filename, tpls := range files {
		err = o.write(filename, chartInfo.ChartName, tpls)
		if err != nil {
			return err
		}
	}
	return nil
}

func (o *output) write(filename, chartName string, templates []helmify.Template) error {
	file := filepath.Join(chartName, "templates", filename)
	err := os.Remove(file)
	if err != nil && !os.IsNotExist(err) {
		return errors.Wrap(err, "unable to remove previous template file")
	}
	f, err := os.OpenFile(file, os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0644)
	if err != nil {
		return errors.Wrap(err, "unable to open template file")
	}
	defer f.Close()
	for i, t := range templates {
		logrus.Debugf("writing a template into './%s/templates/%s'", chartName, filename)
		err = t.Write(f)
		if err != nil {
			return errors.Wrap(err, "unable to write into template file")
		}
		if i != len(templates)-1 {
			_, err = f.Write([]byte("\n---\n"))
			if err != nil {
				return errors.Wrap(err, "unable to write into template file")
			}
		}
	}
	logrus.Infof("'./%s/templates/%s' overwritten", chartName, filename)
	return nil
}

func (o *output) writeValues(chartName string, values helmify.Values) error {
	file := filepath.Join(chartName, "values.yaml")
	if fi, err := os.Stat(file); err == nil && fi.Size() != 0 {
		err = os.Remove(file)
		if err != nil {
			return err
		}
	}
	res, err := yaml.Marshal(values)
	if err != nil {
		return err
	}
	return ioutil.WriteFile(file, res, 0644)
}
