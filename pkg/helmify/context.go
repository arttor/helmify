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
	// we need to add all objects before start processing only to define operator name and namespace.
	if c.info.OperatorNamespace == "" {
		c.info.OperatorNamespace = processor.ExtractOperatorNamespace(obj)
	}
	c.info.OperatorName = processor.ExtractOperatorName(obj, c.info.OperatorName)
	c.objects = append(c.objects, obj)
}

// CreateHelm creates helm chart from context k8s objects
func (c *Context) CreateHelm(stop <-chan struct{}) error {
	logrus.WithFields(logrus.Fields{
		"ChartName":         c.info.ChartName,
		"OperatorName":      c.info.OperatorName,
		"OperatorNamespace": c.info.OperatorNamespace,
	}).Info("creating a chart")
	var templates []Template
	for _, obj := range c.objects {
		template, err := c.process(obj)
		if err != nil {
			return err
		}
		if template != nil {
			templates = append(templates, template)
		}
		select {
		case <-stop:
			return nil
		default:
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
			logrus.WithFields(logrus.Fields{
				"ApiVersion": obj.GetAPIVersion(),
				"Kind":       obj.GetKind(),
				"Name":       obj.GetName(),
			}).Debug("processed")
			return result, nil
		}
	}
	logrus.WithFields(logrus.Fields{
		"Resource": obj.GetObjectKind().GroupVersionKind().String(),
		"Name":     obj.GetName(),
	}).Warn("skip object: no processor defined")
	return nil, nil
}
