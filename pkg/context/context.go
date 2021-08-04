package context

import (
	"github.com/arttor/helmify/pkg/config"
	"github.com/arttor/helmify/pkg/processor"
	"github.com/imdario/mergo"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"io"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

type Context struct {
	processors   []Processor
	templates    []Template
	values       Values
	operatorName string
	output       Output
	config       config.Config
}

func (c *Context) WithProcessor(processor Processor) *Context {
	c.processors = append(c.processors, processor)
	return c
}
func (c *Context) WithOutput(output Output) *Context {
	c.output = output
	return c
}
func (c *Context) WithConfig(config config.Config) *Context {
	c.config = config
	return c
}

func (c *Context) WithProcessors(processors ...Processor) *Context {
	c.processors = append(c.processors, processors...)
	return c
}

func (c *Context) Process(obj *unstructured.Unstructured) error {
	if c.operatorName == "" {
		c.operatorName = processor.GetOperatorName(obj)
	}
	if c.values == nil {
		c.values = Values{}
	}
	for _, p := range c.processors {
		if processed, result, err := p.Process(obj); processed {
			if err != nil {
				return err
			}
			err = c.values.Merge(result.Values())
			if err != nil {
				return err
			}
			c.templates = append(c.templates, result)
			return nil
		}
	}
	logrus.WithFields(logrus.Fields{
		"Resource": obj.GetObjectKind().GroupVersionKind().String(),
		"Name":     obj.GetName(),
	}).Warn("skipped: no suitable processor found")
	return nil
}

func (c *Context) postProcess() {
	for _, t := range c.templates {
		t.PostProcess(c)
	}
}

func (c *Context) Values() Values {
	return c.values
}

func (c *Context) Name() string {
	return c.operatorName
}

func (c *Context) CreateHelm(name string) error {
	c.postProcess()
	err := c.output.Init(name, c.operatorName)
	if err != nil {
		return err
	}
	for _, t := range c.templates {
		t.SetChartName(name)
		c.output.Add(t)
	}
	return c.output.Flush(name, c.values)
}

type Processor interface {
	Process(unstructured *unstructured.Unstructured) (bool, Template, error)
}
type Data interface {
	Values() Values
	Name() string
}

type Template interface {
	Filename() string
	GVK() schema.GroupVersionKind
	Values() Values
	Write(writer io.Writer) error
	PostProcess(data Data)
	SetChartName(name string)
}

type Output interface {
	Init(chartName, appName string) error
	Add(template Template)
	Flush(chartName string, values Values) error
}

type Values map[string]interface{}

func (v *Values) Merge(values Values) error {
	if err := mergo.Merge(v, values, mergo.WithAppendSlice); err != nil {
		return errors.Wrap(err, "unable to merge helm values")
	}
	return nil
}
