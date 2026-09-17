package stringutil

import (
	"strings"
	"unicode"
)

// IsEmpty reports whether s is empty or contains only whitespace.
func IsEmpty(s string) bool {
	return strings.TrimSpace(s) == ""
}

// Truncate shortens s to at most maxLen runes.
func Truncate(s string, maxLen int) string {
	runes := []rune(s)
	if len(runes) <= maxLen {
		return s
	}
	return string(runes[:maxLen])
}

// ToSnakeCase converts a CamelCase string to snake_case.
func ToSnakeCase(s string) string {
	var b strings.Builder
	for i, r := range s {
		if unicode.IsUpper(r) && i > 0 {
			b.WriteByte('_')
		}
		b.WriteRune(unicode.ToLower(r))
	}
	return b.String()
}

// Coalesce returns the first non-empty string from the provided values.
func Coalesce(values ...string) string {
	for _, v := range values {
		if !IsEmpty(v) {
			return v
		}
	}
	return ""
}
