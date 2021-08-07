package helm

import (
	"github.com/arttor/helmify/pkg/helmify"
	"io/ioutil"
	"os"
	"path/filepath"
	"sigs.k8s.io/yaml"
)

func NewOutput() helmify.Output {
	return &output{}
}

type output struct {
}

func (o *output) Create(chartInfo helmify.ChartInfo, templates []helmify.Template) error {
	// init Helm structure if not exists (Chart.yaml, .helmignore, templates/_helpers.tpl)
	err := o.init(chartInfo.ChartName, chartInfo.OperatorName)
	if err != nil {
		return err
	}
	// align templates into files
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
		return err
	}
	f, err := os.OpenFile(file, os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0644)
	if err != nil {
		return err
	}
	defer f.Close()
	for i, t := range templates {
		err = t.Write(f)
		if err != nil {
			return err
		}
		if i != len(templates)-1 {
			_, err = f.Write([]byte("\n---\n"))
			if err != nil {
				return err
			}
		}
	}
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
