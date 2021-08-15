package yaml

import "bytes"

// Indent - adds indentation to given content
func Indent(content []byte, n int) []byte {
	if n < 0 {
		return content
	}
	prefix := append([]byte("\n"), bytes.Repeat([]byte(" "), n)...)
	content = append(prefix[1:], content...)
	return bytes.ReplaceAll(content, []byte("\n"), prefix)
}
