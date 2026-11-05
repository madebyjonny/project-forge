// Package gen implements the make:* generators.
package gen

import (
	"regexp"
	"strings"
	"unicode"
)

var wordRe = regexp.MustCompile(`[A-Z]+[a-z0-9]*|[a-z0-9]+`)

func words(s string) []string {
	var out []string
	for _, part := range regexp.MustCompile(`[^A-Za-z0-9]+`).Split(s, -1) {
		// split "BlogPost" / "HTTPServer" style words
		for _, w := range wordRe.FindAllString(splitAcronyms(part), -1) {
			out = append(out, strings.ToLower(w))
		}
	}
	return out
}

func splitAcronyms(s string) string {
	var b strings.Builder
	r := []rune(s)
	for i, c := range r {
		if i > 0 && unicode.IsUpper(c) && i+1 < len(r) && unicode.IsLower(r[i+1]) && unicode.IsUpper(r[i-1]) {
			b.WriteRune(' ')
		}
		b.WriteRune(c)
	}
	return b.String()
}

func Pascal(s string) string {
	var b strings.Builder
	for _, w := range words(s) {
		b.WriteString(strings.ToUpper(w[:1]) + w[1:])
	}
	return b.String()
}

func Camel(s string) string {
	p := Pascal(s)
	if p == "" {
		return p
	}
	return strings.ToLower(p[:1]) + p[1:]
}

func Snake(s string) string { return strings.Join(words(s), "_") }
func Kebab(s string) string { return strings.Join(words(s), "-") }

// Plural is intentionally simple; pass --table to override.
func Plural(s string) string {
	lower := strings.ToLower(s)
	irregular := map[string]string{"person": "people", "child": "children", "man": "men", "woman": "women", "mouse": "mice"}
	for k, v := range irregular {
		if strings.HasSuffix(lower, k) {
			return s[:len(s)-len(k)] + matchCase(s[len(s)-len(k):], v)
		}
	}
	switch {
	case strings.HasSuffix(lower, "y") && len(s) > 1 && !strings.ContainsRune("aeiou", rune(lower[len(lower)-2])):
		return s[:len(s)-1] + "ies"
	case strings.HasSuffix(lower, "s"), strings.HasSuffix(lower, "x"), strings.HasSuffix(lower, "z"),
		strings.HasSuffix(lower, "ch"), strings.HasSuffix(lower, "sh"):
		return s + "es"
	}
	return s + "s"
}

func matchCase(orig, repl string) string {
	if orig != "" && unicode.IsUpper(rune(orig[0])) {
		return strings.ToUpper(repl[:1]) + repl[1:]
	}
	return repl
}

// Singular strips a trailing role suffix like "Controller" from a name.
func Strip(name, suffix string) string {
	if strings.HasSuffix(name, suffix) && len(name) > len(suffix) {
		return name[:len(name)-len(suffix)]
	}
	return name
}

// Names bundles every casing generators need for one resource.
type Names struct {
	Name   string // Post, BlogPost
	Camel  string // post, blogPost
	File   string // post, blog-post
	Kebab  string
	Var    string // posts, blogPosts (table const)
	Table  string // posts, blog_posts
	Plural string // posts, blog-posts
	Path   string // /api/blog-posts
}

func NamesFor(name string) Names {
	p := Pascal(name)
	return Names{
		Name:   p,
		Camel:  Camel(p),
		File:   Kebab(p),
		Kebab:  Kebab(p),
		Var:    Camel(Plural(p)),
		Table:  Snake(Plural(p)),
		Plural: Kebab(Plural(p)),
		Path:   "/api/" + Kebab(Plural(p)),
	}
}
