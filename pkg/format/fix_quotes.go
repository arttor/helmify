package format

import (
	"strings"
)

// FixUnterminatedQuotes check for Unterminated Quotes in helm templated strings
// See https://github.com/arttor/helmify/issues/12
func FixUnterminatedQuotes(in string) string {
	sb := strings.Builder{}
	hasUntermQuotes := false
	lines := strings.Split(in, "\n")
	for i, line := range lines {
		if hasUntermQuotes {
			line = " " + strings.TrimSpace(line)
			hasUntermQuotes = false
		} else {
			hasUntermQuotes = strings.Count(line, "\"")%2 != 0
		}
		sb.WriteString(line)
		if !hasUntermQuotes && i != len(lines)-1 {
			sb.WriteString("\n")
		}
	}
	return sb.String()
}
