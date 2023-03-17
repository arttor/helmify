package format

import "regexp"

var removeWhitespace = regexp.MustCompile(`(\s+)(\n|$)`)

func RemoveTrailingWhitespaces(in string) string {
	return removeWhitespace.ReplaceAllString(in, "$2")
}
