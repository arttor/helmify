package app

import (
	"bytes"
	"fmt"
	"io"
	"strings"

	"github.com/arttor/helmify/pkg/config"
	"github.com/arttor/helmify/pkg/helmify"
	"github.com/arttor/helmify/pkg/metadata"
	"github.com/arttor/helmify/pkg/processor/pod"
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
	fileNames        []string
}

// New returns context with config set.
func New(config config.Config, output helmify.Output) *appContext {
	return &appContext{
		config:  config,
		appMeta: metadata.New(config),
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
func (c *appContext) Add(obj *unstructured.Unstructured, filename string) {
	// we need to add all objects before start processing only to define app metadata.
	c.appMeta.Load(obj)
	c.objects = append(c.objects, obj)
	c.fileNames = append(c.fileNames, filename)
}

// processedTemplate pairs a processed template with the original object index.
type processedTemplate struct {
	template helmify.Template
	filename string
	objIndex int
}

// CreateHelm creates helm chart from context k8s objects.
func (c *appContext) CreateHelm(stop <-chan struct{}) error {
	logrus.WithFields(logrus.Fields{
		"ChartName": c.appMeta.ChartName(),
		"Namespace": c.appMeta.Namespace(),
	}).Info("creating a chart")

	var processed []processedTemplate
	for i, obj := range c.objects {
		template, err := c.process(obj)
		if err != nil {
			return err
		}
		if template != nil {
			filename := template.Filename()
			if c.fileNames[i] != "" {
				filename = c.fileNames[i]
			}
			processed = append(processed, processedTemplate{
				template: template,
				filename: filename,
				objIndex: i,
			})
		}
		select {
		case <-stop:
			return nil
		default:
		}
	}

	if c.config.AddChecksumAnnotations {
		c.addChecksumAnnotations(processed)
	}

	templates := make([]helmify.Template, len(processed))
	filenames := make([]string, len(processed))
	for i, p := range processed {
		templates[i] = p.template
		filenames[i] = p.filename
	}

	return c.output.Create(c.config.ChartDir, c.config.ChartName, c.config.Crd, c.config.CertManagerAsSubchart, c.config.CertManagerVersion, c.config.CertManagerInstallCRD, templates, filenames)
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

// addChecksumAnnotations wraps workload templates to inject checksum annotations
// for referenced ConfigMaps and Secrets. It uses the actual resolved filenames
// so that the template paths are correct regardless of how input files are organized.
func (c *appContext) addChecksumAnnotations(processed []processedTemplate) {
	// Build maps: object name -> actual template filename for ConfigMaps and Secrets.
	configMapFiles := map[string]string{}
	secretFiles := map[string]string{}
	for _, p := range processed {
		obj := c.objects[p.objIndex]
		switch obj.GroupVersionKind() {
		case metadata.ConfigMapGVK:
			configMapFiles[obj.GetName()] = p.filename
		case metadata.SecretGVK:
			secretFiles[obj.GetName()] = p.filename
		}
	}

	if len(configMapFiles) == 0 && len(secretFiles) == 0 {
		return
	}

	// Wrap workload templates with checksum annotations.
	for i, p := range processed {
		obj := c.objects[p.objIndex]
		checksumAnns := pod.ChecksumAnnotations(c.appMeta, obj, configMapFiles, secretFiles)
		if checksumAnns != "" {
			processed[i].template = &checksumTemplate{
				wrapped:     p.template,
				annotations: checksumAnns,
			}
		}
	}
}

// checksumTemplate wraps a Template to inject checksum annotations into its output.
type checksumTemplate struct {
	wrapped     helmify.Template
	annotations string
}

func (t *checksumTemplate) Filename() string {
	return t.wrapped.Filename()
}

func (t *checksumTemplate) Values() helmify.Values {
	return t.wrapped.Values()
}

func (t *checksumTemplate) Write(writer io.Writer) error {
	var buf bytes.Buffer
	if err := t.wrapped.Write(&buf); err != nil {
		return err
	}
	output := injectAnnotations(buf.String(), t.annotations)
	_, err := fmt.Fprint(writer, output)
	return err
}

// injectAnnotations injects checksum annotations into the pod template metadata
// section of a workload YAML. It looks for the "spec:" → "template:" → "metadata:"
// pattern (2-space indent, which is what helmify always produces) and inserts or
// appends to an annotations block.
func injectAnnotations(yaml string, annotations string) string {
	lines := strings.Split(yaml, "\n")
	var result []string
	injected := false

	for i := 0; i < len(lines); i++ {
		result = append(result, lines[i])

		if injected {
			continue
		}

		if strings.TrimSpace(lines[i]) != "template:" {
			continue
		}
		indent := len(lines[i]) - len(strings.TrimLeft(lines[i], " "))
		if indent < 2 {
			continue
		}

		// Verify parent "spec:" at indent-2 by scanning back until we find
		// a line at a lower or equal indent level (the parent block).
		hasSpec := false
		for k := i - 1; k >= 0; k-- {
			kIndent := len(lines[k]) - len(strings.TrimLeft(lines[k], " "))
			trimmed := strings.TrimSpace(lines[k])
			if trimmed == "" {
				continue
			}
			if kIndent < indent {
				hasSpec = trimmed == "spec:" && kIndent == indent-2
				break
			}
		}
		if !hasSpec {
			continue
		}

		// Expect "metadata:" at indent+2.
		if i+1 >= len(lines) {
			continue
		}
		nextIndent := len(lines[i+1]) - len(strings.TrimLeft(lines[i+1], " "))
		if strings.TrimSpace(lines[i+1]) != "metadata:" || nextIndent != indent+2 {
			continue
		}

		// Found pod template metadata — inject annotations.
		result = append(result, lines[i+1]) // metadata: line
		i = i + 1

		annKeyIndent := strings.Repeat(" ", indent+4)
		annValueIndent := strings.Repeat(" ", indent+6)

		if i+1 < len(lines) && strings.TrimSpace(lines[i+1]) == "annotations:" {
			result = append(result, lines[i+1])
			i = i + 1
		} else {
			result = append(result, annKeyIndent+"annotations:")
		}
		for _, ann := range strings.Split(annotations, "\n") {
			result = append(result, annValueIndent+ann)
		}

		injected = true
	}

	return strings.Join(result, "\n")
}
