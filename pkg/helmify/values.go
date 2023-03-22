package helmify

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/iancoleman/strcase"
	"github.com/imdario/mergo"
	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

// Values - represents helm template values.yaml.
type Values map[string]interface{}

// Merge given values with current instance.
func (v *Values) Merge(values Values) error {
	if err := mergo.Merge(v, values, mergo.WithAppendSlice); err != nil {
		return errors.Wrap(err, "unable to merge helm values")
	}
	return nil
}

// Add - adds given value to values and returns its helm template representation {{ .Values.<valueName> }}
func (v *Values) Add(value interface{}, name ...string) (string, error) {
	name = toCamelCase(name)
	switch val := value.(type) {
	case int:
		value = int64(val)
	case int8:
		value = int64(val)
	case int16:
		value = int64(val)
	case int32:
		value = int64(val)
	}

	err := unstructured.SetNestedField(*v, value, name...)
	if err != nil {
		return "", errors.Wrapf(err, "unable to set value: %v", name)
	}
	_, isString := value.(string)
	if isString {
		return "{{ .Values." + strings.Join(name, ".") + " | quote }}", nil
	}
	_, isSlice := value.([]interface{})
	if isSlice {
		spaces := strconv.Itoa(len(name) * 2)
		return "{{ toYaml .Values." + strings.Join(name, ".") + " | nindent " + spaces + " }}", nil
	}
	return "{{ .Values." + strings.Join(name, ".") + " }}", nil
}

// AddYaml - adds given value to values and returns its helm template representation as Yaml {{ .Values.<valueName> | toYaml | indent i }}
// indent  <= 0 will be omitted.
func (v *Values) AddYaml(value interface{}, indent int, name ...string) (string, error) {
	name = toCamelCase(name)
	err := unstructured.SetNestedField(*v, value, name...)
	if err != nil {
		return "", errors.Wrapf(err, "unable to set value: %v", name)
	}
	if indent > 0 {
		return "{{ .Values." + strings.Join(name, ".") + fmt.Sprintf(" | toYaml | indent %d }}", indent), nil
	}
	return "{{ .Values." + strings.Join(name, ".") + " | toYaml }}", nil
}

// AddSecret - adds empty value to values and returns its helm template representation {{ required "<valueName>" .Values.<valueName> }}.
// Set toBase64=true for Secret data to be base64 encoded and set false for Secret stringData.
func (v *Values) AddSecret(toBase64 bool, name ...string) (string, error) {
	name = toCamelCase(name)
	nameStr := strings.Join(name, ".")
	err := unstructured.SetNestedField(*v, "", name...)
	if err != nil {
		return "", errors.Wrapf(err, "unable to set value: %v", nameStr)
	}
	res := fmt.Sprintf(`{{ required "%[1]s is required" .Values.%[1]s`, nameStr)
	if toBase64 {
		res += " | b64enc"
	}
	return res + " | quote }}", err
}

func toCamelCase(name []string) []string {
	for i, n := range name {
		camelCase := strcase.ToLowerCamel(n)
		if n == strings.ToUpper(n) {
			camelCase = strcase.ToLowerCamel(strings.ToLower(n))
		}
		name[i] = camelCase
	}
	return name
}
