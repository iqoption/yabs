package utils

import (
	"strings"
	"unicode"
)

//Remove control symbols
func Trim(str string) string {
	return strings.TrimFunc(str, func(c rune) bool {
		return unicode.IsControl(c)
	})
}
