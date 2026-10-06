package schema

import (
	"strings"
)

// Pascal converts a snake/kebab/space separated name to PascalCase.
func Pascal(s string) string {
	parts := strings.FieldsFunc(s, func(r rune) bool {
		return r == '_' || r == '-' || r == ' '
	})
	var b strings.Builder
	for _, p := range parts {
		if p == "" {
			continue
		}
		b.WriteString(strings.ToUpper(p[:1]))
		b.WriteString(p[1:])
	}
	return b.String()
}

// Camel converts a name to camelCase.
func Camel(s string) string {
	p := Pascal(s)
	if p == "" {
		return ""
	}
	return strings.ToLower(p[:1]) + p[1:]
}

// Lower returns the lowercase, separator-normalized form of a name.
func Lower(s string) string {
	return strings.ToLower(strings.TrimSpace(s))
}

// Snake converts a name to snake_case.
func Snake(s string) string {
	var b strings.Builder
	for i, r := range s {
		if r >= 'A' && r <= 'Z' {
			if i > 0 && s[i-1] != '_' && s[i-1] != '-' && (s[i-1] < 'A' || s[i-1] > 'Z') {
				b.WriteByte('_')
			}
			b.WriteRune(r + ('a' - 'A'))
		} else if r == '-' || r == ' ' {
			b.WriteByte('_')
		} else {
			b.WriteRune(r)
		}
	}
	return b.String()
}

// Plural returns a naive lowercase pluralized name used for table/collection
// names and route groups. It handles common English suffixes.
func Plural(s string) string {
	lower := Lower(s)
	switch {
	case strings.HasSuffix(lower, "s"), strings.HasSuffix(lower, "x"),
		strings.HasSuffix(lower, "z"), strings.HasSuffix(lower, "ch"),
		strings.HasSuffix(lower, "sh"):
		return lower + "es"
	case strings.HasSuffix(lower, "y") && len(lower) > 1 &&
		!isVowel(lower[len(lower)-2]):
		return lower[:len(lower)-1] + "ies"
	default:
		return lower + "s"
	}
}

func isVowel(c byte) bool {
	switch c {
	case 'a', 'e', 'i', 'o', 'u':
		return true
	}
	return false
}
