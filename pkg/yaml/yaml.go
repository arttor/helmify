package yaml

import (
	"bytes"

	"sigs.k8s.io/yaml"
)

// Indent - adds indentation to given content.
func Indent(content []byte, n int) []byte {
	if n < 0 {
		return content
	}
	prefix := append([]byte("\n"), bytes.Repeat([]byte(" "), n)...)
	content = append(prefix[1:], content...)
	return bytes.ReplaceAll(content, []byte("\n"), prefix)
}

// Marshal object to yaml string with indentation.
func Marshal(object interface{}, indent int) (string, error) {
	objectBytes, err := yaml.Marshal(object)
	if err != nil {
		return "", err
	}
	objectBytes = Indent(objectBytes, indent)
	objectBytes = bytes.TrimRight(objectBytes, "\n ")
	return string(objectBytes), nil
}
