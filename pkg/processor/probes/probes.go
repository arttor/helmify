package probes

import (
	"fmt"
	"strings"

	"github.com/arttor/helmify/pkg/helmify"
	yamlformat "github.com/arttor/helmify/pkg/yaml"
	"github.com/iancoleman/strcase"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

const livenessProbe = "livenessProbe"
const readinessProbe = "readinessProbe"

const livenessProbeTemplate = "\n{{- if .Values.%[1]s.%[2]s.livenessProbe }}\n" +
	"livenessProbe: {{- include \"tplvalues.render\" (dict \"value\" .Values.%[1]s.%[2]s.livenessProbe \"context\" $) | nindent 10 }}\n" +
	" {{- else }}\n" +
	"livenessProbe:\n%[3]s" +
	"\n{{- end }}"

const readinessProbeTemplate = "\n{{- if .Values.%[1]s.%[2]s.readinessProbe }}\n" +
	"readinessProbe: {{- include \"tplvalues.render\" (dict \"value\" .Values.%[1]s.%[2]s.readinessProbe \"context\" $) | nindent 10 }}\n" +
	" {{- else }}\n" +
	"readinessProbe:\n%[3]s" +
	"\n{{- end }}"

// ProcessSpecMap adds 'probes' to the Containers in specMap, if they are defined
func ProcessSpecMap(name string, specMap map[string]interface{}, values *helmify.Values) (string, error) {

	cs, _, err := unstructured.NestedSlice(specMap, "containers")

	if err != nil {
		return "", err
	}

	strContainers := make([]interface{}, len(cs))
	for i, c := range cs {
		castedContainer := c.(map[string]interface{})
		strContainers[i], err = setProbesTemplates(name, castedContainer, values)
		if err != nil {
			return "", err
		}
	}
	specMap["containers"] = strContainers
	specs, err := yamlformat.Marshal(specMap, 6)
	if err != nil {
		return "", err
	}
	res := strings.ReplaceAll(string(specs), "|\n        ", "")
	res = strings.ReplaceAll(res, "|-\n        ", "")

	return res, nil
}

func setProbesTemplates(name string, castedContainer map[string]interface{}, values *helmify.Values) (string, error) {

	var ready, live string
	var err error
	if _, defined := castedContainer[livenessProbe]; defined {
		live, err = setProbe(name, castedContainer, values, livenessProbe)
		if err != nil {
			return "", err
		}
		delete(castedContainer, livenessProbe)
	}
	if _, defined := castedContainer[readinessProbe]; defined {
		ready, err = setProbe(name, castedContainer, values, readinessProbe)
		if err != nil {
			return "", err
		}
		delete(castedContainer, readinessProbe)
	}
	return setMap(name, castedContainer, live, ready)

}

func setMap(name string, castedContainer map[string]interface{}, live string, ready string) (string, error) {
	containerName := strcase.ToLowerCamel(castedContainer["name"].(string))
	content, err := yamlformat.Marshal(castedContainer, 0)
	if err != nil {
		return "", err
	}
	strContainer := string(content)
	if live != "" {
		strContainer = strContainer + fmt.Sprintf(livenessProbeTemplate, name, containerName, live)
	}
	if ready != "" {
		strContainer = strContainer + fmt.Sprintf(readinessProbeTemplate, name, containerName, ready)
	}

	return strContainer, nil
}

func setProbe(name string, castedContainer map[string]interface{}, values *helmify.Values, probe string) (string, error) {
	containerName := strcase.ToLowerCamel(castedContainer["name"].(string))
	templatedProbe, err := yamlformat.Marshal(castedContainer[probe], 1)
	if err != nil {
		return "", err
	}

	return templatedProbe, unstructured.SetNestedField(*values, castedContainer[probe], name, containerName, probe)

}
