package processor

import (
	"fmt"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"strings"

	"github.com/arttor/helmify/pkg/helmify"
	yamlformat "github.com/arttor/helmify/pkg/yaml"
)

const metaTeml = `apiVersion: %[1]s
kind: %[2]s
metadata:
  name: %[3]s
  labels:
  {{- include "%[2]s.labels" . | nindent 4 }}
%[4]s
%[5]s
`

// ProcessMetadata - returns object apiVersion, kind and metadata as helm template.
func ProcessMetadata(info helmify.ChartInfo, obj *unstructured.Unstructured) (metaStr string, name string, err error) {
	var labels, annotations string
	if len(obj.GetLabels()) != 0 {
		labels, err = yamlformat.Marshal(obj.GetLabels(), 4)
	}
	if err != nil {
		return
	}
	if len(obj.GetAnnotations()) != 0 {
		annotations, err = yamlformat.Marshal(map[string]interface{}{"annotations": obj.GetAnnotations()}, 2)
	}
	if err != nil {
		return
	}
	name = strings.TrimPrefix(obj.GetName(), info.ApplicationName)
	name = strings.Trim(name, "-_. /")
	templatedName := fmt.Sprintf(`{{ include "%s.fullname" . }}-%s`, info.ChartName, name)
	apiVersion, kind := obj.GetObjectKind().GroupVersionKind().ToAPIVersionAndKind()
	metaStr = fmt.Sprintf(metaTeml, apiVersion, kind, templatedName, labels, annotations)
	metaStr = strings.Trim(metaStr, " \n") + "\n"
	return metaStr, name, nil
}
