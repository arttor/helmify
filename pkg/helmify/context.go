package helmify

import (
	"github.com/arttor/helmify/pkg/config"
	"github.com/arttor/helmify/pkg/processor"
	"github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

// Context helm processing context. Stores processed objects
type Context struct {
	processors []Processor
	templates  []Template
	output     Output
	config     config.Config
	info       ChartInfo
	objects    []*unstructured.Unstructured
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
	c.info.ChartName = config.ChartName
	return c
}

func (c *Context) WithProcessors(processors ...Processor) *Context {
	c.processors = append(c.processors, processors...)
	return c
}

// Add k8s object to helmify context
func (c *Context) Add(obj *unstructured.Unstructured) {
	if c.info.OperatorNamespace == "" {
		c.info.OperatorNamespace = processor.ExtractOperatorNamespace(obj)
	}
	c.info.OperatorName = processor.ExtractOperatorName(obj, c.info.OperatorName)
	c.objects = append(c.objects, obj)
}

// CreateHelm creates helm chart from context k8s objects
func (c *Context) CreateHelm(stop <-chan struct{}) error {
	values := Values{}
	var templates []Template
	for _, obj := range c.objects {
		if interrupted(stop) {
			return nil
		}
		template, err := c.process(obj)
		if err != nil {
			return err
		}
		if template != nil {
			templates = append(templates, template)
			err = values.Merge(template.Values())
			if err != nil {
				return err
			}
		}
	}
	for _, t := range templates {
		t.PostProcess(values)
		if interrupted(stop) {
			return nil
		}
	}
	return c.output.Create(c.info, templates)
}

func (c *Context) process(obj *unstructured.Unstructured) (Template, error) {
	for _, p := range c.processors {
		if processed, result, err := p.Process(c.info, obj); processed {
			if err != nil {
				return nil, err
			}
			return result, nil
		}
	}
	logrus.WithFields(logrus.Fields{
		"Resource": obj.GetObjectKind().GroupVersionKind().String(),
		"Name":     obj.GetName(),
	}).Warn("skipped: no suitable processor found")
	return nil, nil
}

func interrupted(stop <-chan struct{}) bool {
	select {
	case <-stop:
		return true
	default:
		return false
	}
}
