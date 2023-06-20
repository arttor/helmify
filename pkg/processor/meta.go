package processor

import (
	"fmt"
	"strings"

	"github.com/iancoleman/strcase"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	"github.com/arttor/helmify/pkg/helmify"
	yamlformat "github.com/arttor/helmify/pkg/yaml"
)

const defaultMetaTemplate = `apiVersion: %[1]s
kind: %[2]s
metadata:
  name: %[3]s
  labels:
%[5]s
  {{- include "%[4]s.labels" . | nindent 4 }}
%[6]s`

const annotationsMetaTemplate = `apiVersion: %[1]s
kind: %[2]s
metadata:
  name: %[3]s
  labels:
%[5]s
  {{- include "%[4]s.labels" . | nindent 4 }}
  annotations:
%[6]s
  {{- toYaml .Values.%[7]s.%[8]s.annotations | nindent 4 }}`

type MetaOpt interface {
	apply(*options)
}

type options struct {
	values      helmify.Values
	annotations bool
}

type annotationsOption struct {
	values helmify.Values
}

func (a annotationsOption) apply(opts *options) {
	opts.annotations = true
	opts.values = a.values
}

func WithAnnotations(values helmify.Values) MetaOpt {
	return annotationsOption{
		values: values,
	}
}

// ProcessObjMeta - returns object apiVersion, kind and metadata as helm template.
func ProcessObjMeta(appMeta helmify.AppMetadata, obj *unstructured.Unstructured, opts ...MetaOpt) (string, error) {
	options := &options{
		values:      nil,
		annotations: false,
	}
	for _, opt := range opts {
		opt.apply(options)
	}

	var err error
	var labels, annotations string
	if len(obj.GetLabels()) != 0 {
		l := obj.GetLabels()
		// provided by Helm
		delete(l, "app.kubernetes.io/name")
		delete(l, "app.kubernetes.io/instance")
		delete(l, "app.kubernetes.io/version")
		delete(l, "app.kubernetes.io/managed-by")
		delete(l, "helm.sh/chart")

		// Since we delete labels above, it is possible that at this point there are no more labels.
		if len(l) > 0 {
			labels, err = yamlformat.Marshal(l, 4)
			if err != nil {
				return "", err
			}
		}
	}
	if len(obj.GetAnnotations()) != 0 {
		annotations, err = yamlformat.Marshal(map[string]interface{}{"annotations": obj.GetAnnotations()}, 2)
		if err != nil {
			return "", err
		}
	}
	templatedName := appMeta.TemplatedName(obj.GetName())
	apiVersion, kind := obj.GetObjectKind().GroupVersionKind().ToAPIVersionAndKind()

	var metaStr string
	if options.values != nil && options.annotations {
		if len(obj.GetAnnotations()) != 0 {
			annotations, err = yamlformat.Marshal(obj.GetAnnotations(), 4)
			if err != nil {
				return "", err
			}
		}
		name := strcase.ToLowerCamel(appMeta.TrimName(obj.GetName()))
		err = unstructured.SetNestedField(options.values, map[string]interface{}{}, name, strings.ToLower(kind), "annotations")
		metaStr = fmt.Sprintf(annotationsMetaTemplate, apiVersion, kind, templatedName, appMeta.ChartName(), labels, annotations, name, strings.ToLower(kind))
	} else {
		metaStr = fmt.Sprintf(defaultMetaTemplate, apiVersion, kind, templatedName, appMeta.ChartName(), labels, annotations)
	}
	metaStr = strings.Trim(metaStr, " \n")
	metaStr = strings.ReplaceAll(metaStr, "\n\n", "\n")
	return metaStr, nil
}
