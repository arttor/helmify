package helm

import (
	"github.com/arttor/helmify/pkg/context"
	"io/ioutil"
	"os"
	"path/filepath"
	"sigs.k8s.io/yaml"
)

func NewOutput() context.Output {
	return &output{}
}

type output struct {
	files map[string][]context.Template
}

func (o *output) Add(template context.Template) {
	if o.files == nil {
		o.files = map[string][]context.Template{}
	}
	file := o.files[template.Filename()]
	file = append(file, template)
	o.files[template.Filename()] = file
}

func (o *output) Flush(chartName string, values context.Values) error {
	err := o.writeValues(chartName, values)
	if err != nil {
		return err
	}
	for filename, tpls := range o.files {
		err = o.write(filename, chartName, tpls)
		if err != nil {
			return err
		}
	}
	return nil
}
func (o *output) write(filename, chartName string, templates []context.Template) error {
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
func (o *output) writeValues(chartName string, values context.Values) error {
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
