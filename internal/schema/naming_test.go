package schema

import "testing"

func TestPascal(t *testing.T) {
	cases := map[string]string{
		"user":          "User",
		"blog_post":     "BlogPost",
		"kebab-case":    "KebabCase",
		"space name":    "SpaceName",
		"":              "",
		"alreadyUpper":  "AlreadyUpper",
		"__double__":    "Double",
		"mixed_snake-Kebab": "MixedSnakeKebab",
	}
	for in, want := range cases {
		if got := Pascal(in); got != want {
			t.Errorf("Pascal(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestCamel(t *testing.T) {
	cases := map[string]string{
		"user":        "user",
		"blog_post":   "blogPost",
		"BlogPost":    "blogPost",
		"":            "",
		"a":           "a",
	}
	for in, want := range cases {
		if got := Camel(in); got != want {
			t.Errorf("Camel(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestLower(t *testing.T) {
	cases := map[string]string{
		"User":   "user",
		"  Blog  ": "blog",
		"ABC":    "abc",
	}
	for in, want := range cases {
		if got := Lower(in); got != want {
			t.Errorf("Lower(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestPlural(t *testing.T) {
	cases := map[string]string{
		"post":   "posts",
		"box":    "boxes",
		"class":  "classes",
		"bush":   "bushes",
		"baby":   "babies",
		"day":    "days",
		"user":   "users",
		"status": "statuses",
	}
	for in, want := range cases {
		if got := Plural(in); got != want {
			t.Errorf("Plural(%q) = %q, want %q", in, got, want)
		}
	}
}
