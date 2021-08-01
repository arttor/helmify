package yaml

import "bytes"

func Indent(b []byte, n int) []byte {
	prefix := append([]byte("\n"), bytes.Repeat([]byte(" "), n)...)
	b = append(prefix[1:], b...)
	return bytes.ReplaceAll(b, []byte("\n"), prefix)
}
