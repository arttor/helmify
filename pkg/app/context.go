package app

import (
	"github.com/arttor/helmify/pkg/config"
	"github.com/arttor/helmify/pkg/helmify"
	"github.com/arttor/helmify/pkg/metadata"
	"github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

// appContext helm processing context. Stores processed objects.
type appContext struct {
	processors       []helmify.Processor
	defaultProcessor helmify.Processor
	output           helmify.Output
	config           config.Config
	appMeta          *metadata.Service
	objects          []*unstructured.Unstructured
}

// New returns context with config set.
func New(config config.Config, output helmify.Output) *appContext {
	return &appContext{
		config:  config,
		appMeta: metadata.New(config.ChartName),
		output:  output,
	}
}

// WithProcessors  add processors to the context and returns it.
func (c *appContext) WithProcessors(processors ...helmify.Processor) *appContext {
	c.processors = append(c.processors, processors...)
	return c
}

// WithDefaultProcessor  add defaultProcessor for unknown resources to the context and returns it.
func (c *appContext) WithDefaultProcessor(processor helmify.Processor) *appContext {
	c.defaultProcessor = processor
	return c
}

// Add k8s object to app context.
func (c *appContext) Add(obj *unstructured.Unstructured) {
	// we need to add all objects before start processing only to define app metadata.
	c.appMeta.Load(obj)
	c.objects = append(c.objects, obj)
}

// CreateHelm creates helm chart from context k8s objects.
func (c *appContext) CreateHelm(stop <-chan struct{}) error {
	logrus.WithFields(logrus.Fields{
		"ChartName": c.appMeta.ChartName(),
		"Namespace": c.appMeta.Namespace(),
	}).Info("creating a chart")
	var templates []helmify.Template
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
	return c.output.Create(c.config.ChartDir, c.config.ChartName, templates)
}

func (c *appContext) process(obj *unstructured.Unstructured) (helmify.Template, error) {
	for _, p := range c.processors {
		if processed, result, err := p.Process(c.appMeta, obj); processed {
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
	if c.defaultProcessor == nil {
		logrus.WithFields(logrus.Fields{
			"ApiVersion": obj.GetAPIVersion(),
			"Kind":       obj.GetKind(),
			"Name":       obj.GetName(),
		}).Warn("Skipping: no suitable processor for resource.")
		return nil, nil
	}
	_, t, err := c.defaultProcessor.Process(c.appMeta, obj)
	return t, err
}
