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
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
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

// CreateHelm creates helm chart from context k8s objects.
func (c *appContext) CreateHelm(stop <-chan struct{}) error {
	logrus.WithFields(logrus.Fields{
		"ChartName": c.appMeta.ChartName(),
		"Namespace": c.appMeta.Namespace(),
	}).Info("creating a chart")
	var templates []helmify.Template
	var filenames []string
	// objIndices tracks which c.objects index produced each template.
	var objIndices []int
	for i, obj := range c.objects {
		template, err := c.process(obj)
		if err != nil {
			return err
		}
		if template != nil {
			templates = append(templates, template)
			filename := template.Filename()
			if c.fileNames[i] != "" {
				filename = c.fileNames[i]
			}
			filenames = append(filenames, filename)
			objIndices = append(objIndices, i)
		}
		select {
		case <-stop:
			return nil
		default:
		}
	}

	if c.config.AddChecksumAnnotations {
		templates = c.addChecksumAnnotations(templates, filenames, objIndices)
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

var (
	configMapGVK = schema.GroupVersionKind{Group: "", Version: "v1", Kind: "ConfigMap"}
	secretGVK    = schema.GroupVersionKind{Group: "", Version: "v1", Kind: "Secret"}
)

// workloadGVKs lists the resource kinds whose pod templates should get checksum annotations.
var workloadGVKs = map[schema.GroupVersionKind]bool{
	{Group: "apps", Version: "v1", Kind: "Deployment"}: true,
	{Group: "apps", Version: "v1", Kind: "DaemonSet"}:  true,
	{Group: "apps", Version: "v1", Kind: "StatefulSet"}: true,
}

// addChecksumAnnotations wraps workload templates to inject checksum annotations
// for referenced ConfigMaps and Secrets. It uses the actual resolved filenames
// so that the template paths are correct regardless of how input files are organized.
func (c *appContext) addChecksumAnnotations(templates []helmify.Template, filenames []string, objIndices []int) []helmify.Template {
	// Build maps: object name -> actual template filename for ConfigMaps and Secrets.
	configMapFiles := map[string]string{}
	secretFiles := map[string]string{}
	for i, tmplIdx := range objIndices {
		obj := c.objects[tmplIdx]
		switch obj.GroupVersionKind() {
		case configMapGVK:
			configMapFiles[obj.GetName()] = filenames[i]
		case secretGVK:
			secretFiles[obj.GetName()] = filenames[i]
		}
	}

	if len(configMapFiles) == 0 && len(secretFiles) == 0 {
		return templates
	}

	// Wrap workload templates with checksum annotations.
	result := make([]helmify.Template, len(templates))
	copy(result, templates)
	for i, tmplIdx := range objIndices {
		obj := c.objects[tmplIdx]
		if !workloadGVKs[obj.GroupVersionKind()] {
			continue
		}
		podSpec := extractPodSpec(obj)
		if podSpec == nil {
			continue
		}
		checksumAnns := pod.ChecksumAnnotations(c.appMeta, *podSpec, configMapFiles, secretFiles)
		if checksumAnns != "" {
			result[i] = &checksumTemplate{
				wrapped:     templates[i],
				annotations: checksumAnns,
			}
		}
	}

	return result
}

// extractPodSpec extracts the PodSpec from a workload object.
func extractPodSpec(obj *unstructured.Unstructured) *corev1.PodSpec {
	switch obj.GroupVersionKind() {
	case schema.GroupVersionKind{Group: "apps", Version: "v1", Kind: "Deployment"}:
		var d appsv1.Deployment
		if err := runtime.DefaultUnstructuredConverter.FromUnstructured(obj.Object, &d); err != nil {
			return nil
		}
		return &d.Spec.Template.Spec
	case schema.GroupVersionKind{Group: "apps", Version: "v1", Kind: "DaemonSet"}:
		var d appsv1.DaemonSet
		if err := runtime.DefaultUnstructuredConverter.FromUnstructured(obj.Object, &d); err != nil {
			return nil
		}
		return &d.Spec.Template.Spec
	case schema.GroupVersionKind{Group: "apps", Version: "v1", Kind: "StatefulSet"}:
		var s appsv1.StatefulSet
		if err := runtime.DefaultUnstructuredConverter.FromUnstructured(obj.Object, &s); err != nil {
			return nil
		}
		return &s.Spec.Template.Spec
	}
	return nil
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
	output := buf.String()
	output = injectAnnotations(output, t.annotations)
	_, err := fmt.Fprint(writer, output)
	return err
}

// injectAnnotations injects checksum annotations into the pod template metadata
// section of a workload YAML. It looks for the `template:\n    metadata:` pattern
// and adds an `annotations:` block (or appends to an existing one).
func injectAnnotations(yaml string, annotations string) string {
	lines := strings.Split(yaml, "\n")
	var result []string
	injected := false

	for i := 0; i < len(lines); i++ {
		result = append(result, lines[i])

		if injected {
			continue
		}

		// Look for "  template:" (the pod template, not other uses of "template")
		trimmed := strings.TrimRight(lines[i], " ")
		if trimmed != "  template:" && trimmed != "    template:" {
			continue
		}
		templateIndent := strings.Repeat(" ", len(lines[i])-len(strings.TrimLeft(lines[i], " ")))

		// Find the metadata: line within the next few lines
		for j := i + 1; j < len(lines) && j <= i+2; j++ {
			if strings.TrimSpace(lines[j]) != "metadata:" {
				continue
			}
			metadataIndent := templateIndent + "  "
			annIndent := metadataIndent + "  "

			result = append(result, lines[j])
			i = j

			// Check if there's already an annotations: block right after metadata:
			if j+1 < len(lines) && strings.TrimSpace(lines[j+1]) == "annotations:" {
				result = append(result, lines[j+1])
				i = j + 1
				// Insert our annotations after the existing annotations: key
				for _, ann := range strings.Split(annotations, "\n") {
					result = append(result, annIndent+"  "+ann)
				}
			} else {
				// Add new annotations block
				result = append(result, annIndent+"annotations:")
				for _, ann := range strings.Split(annotations, "\n") {
					result = append(result, annIndent+"  "+ann)
				}
			}
			injected = true
			break
		}
	}

	return strings.Join(result, "\n")
}
