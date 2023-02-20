package processor

import (
	"fmt"
	"strings"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	"github.com/arttor/helmify/pkg/helmify"
	yamlformat "github.com/arttor/helmify/pkg/yaml"
)

const metaTeml = `apiVersion: %[1]s
kind: %[2]s
metadata:
  name: %[3]s
  namespace: {{ .Release.Namespace | quote }}
  labels:
%[5]s
  {{- include "%[4]s.labels" . | nindent 4 }}
%[6]s`

// ProcessObjMeta - returns object apiVersion, kind and metadata as helm template.
func ProcessObjMeta(appMeta helmify.AppMetadata, obj *unstructured.Unstructured) (string, error) {
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
	metaStr := fmt.Sprintf(metaTeml, apiVersion, kind, templatedName, appMeta.ChartName(), labels, annotations)
	metaStr = strings.Trim(metaStr, " \n")
	metaStr = strings.Replace(metaStr, "\n\n", "\n", -1)
	return metaStr, nil
}
