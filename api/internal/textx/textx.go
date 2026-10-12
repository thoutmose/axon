// Package textx holds small text helpers shared across packages.
package textx

import (
	"strings"
	"unicode/utf8"
)

// Cut returns s as valid UTF-8 in at most n bytes, never splitting a
// character; invalid bytes become U+FFFD. PostgreSQL refuses invalid UTF-8,
// so text from outside (a request header, a model's answer) goes through
// this before it is stored.
func Cut(s string, n int) string {
	s = strings.ToValidUTF8(s, "\uFFFD")
	if len(s) <= n {
		return s
	}
	for n > 0 && !utf8.RuneStart(s[n]) {
		n--
	}
	return s[:n]
}
