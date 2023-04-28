package rbac

import (
	"fmt"
	"github.com/arttor/helmify/pkg/helmify"
	yamlformat "github.com/arttor/helmify/pkg/yaml"
	"github.com/iancoleman/strcase"
	"io"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"strings"
)

var serviceAccountGVC = schema.GroupVersionKind{
	Group:   "",
	Version: "v1",
	Kind:    "ServiceAccount",
}

// ServiceAccount creates processor for k8s ServiceAccount resource.
func ServiceAccount() helmify.Processor {
	return &serviceAccount{}
}

type serviceAccount struct{}

// Process k8s ServiceAccount object into helm template. Returns false if not capable of processing given resource type.
func (sa serviceAccount) Process(appMeta helmify.AppMetadata, obj *unstructured.Unstructured) (bool, helmify.Template, error) {
	if obj.GroupVersionKind() != serviceAccountGVC {
		return false, nil, nil
	}
	meta, values, err := sa.processServiceAccountMeta(appMeta, obj)
	if err != nil {
		return true, nil, err
	}

	return true, &saResult{
		data:   []byte(meta),
		values: values,
	}, nil
}

const metaTeml = `apiVersion: %[1]s
kind: %[2]s
metadata:
  name: %[3]s
  labels:
%[5]s
  {{- include "%[4]s.labels" . | nindent 4 }}
  annotations:
%[6]s
  {{ toYaml .Values.%[7]s.annotations | nindent 4 }}`

func (sa serviceAccount) processServiceAccountMeta(appMeta helmify.AppMetadata, obj *unstructured.Unstructured) (string, helmify.Values, error) {
	var err error
	var labels, annotations string
	values := helmify.Values{}
	name := strcase.ToLowerCamel(appMeta.TrimName(obj.GetName()))
	err = unstructured.SetNestedField(values, map[string]interface{}{}, name, "serviceaccount", "annotations")
	if err != nil {
		return "", nil, err
	}
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
				return "", nil, err
			}
		}
	}
	if len(obj.GetAnnotations()) != 0 {
		annotations, err = yamlformat.Marshal(obj.GetAnnotations(), 4)
		if err != nil {
			return "", nil, err
		}
	}
	templatedName := appMeta.TemplatedName(obj.GetName())
	apiVersion, kind := obj.GetObjectKind().GroupVersionKind().ToAPIVersionAndKind()
	metaStr := fmt.Sprintf(metaTeml, apiVersion, kind, templatedName, appMeta.ChartName(), labels, annotations, name)
	metaStr = strings.Trim(metaStr, " \n")
	metaStr = strings.ReplaceAll(metaStr, "\n\n", "\n")
	return metaStr, values, nil
}

type saResult struct {
	data   []byte
	values helmify.Values
}

func (r *saResult) Filename() string {
	return "serviceaccount.yaml"
}

func (r *saResult) Values() helmify.Values {
	return r.values
}

func (r *saResult) Write(writer io.Writer) error {
	_, err := writer.Write(r.data)
	return err
}
