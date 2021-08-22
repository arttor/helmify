package helmify

import (
	"github.com/iancoleman/strcase"
	"github.com/imdario/mergo"
	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"strconv"
	"strings"
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
func (v *Values) Add(value interface{}, name []string) (string, error) {
	name = toCamelCase(name)
	val, isString := parseValue(value)
	err := unstructured.SetNestedField(*v, val, name...)
	if err != nil {
		return "", errors.Wrapf(err, "unable to set value: %v", name)
	}
	if isString {
		return "{{ .Values." + strings.Join(name, ".") + " | quote }}", nil
	}
	return "{{ .Values." + strings.Join(name, ".") + " }}", nil
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

func parseValue(value interface{}) (val interface{}, isString bool) {
	str, ok := value.(string)
	if !ok {
		return value, false
	}
	str = strings.Trim(str, "\"")
	i, err := strconv.ParseInt(str, 10, 64)
	if err == nil {
		return i, false
	}
	f, err := strconv.ParseFloat(str, 64)
	if err == nil {
		return f, false
	}
	b, err := strconv.ParseBool(str)
	if err == nil {
		return b, false
	}
	return str, true
}
