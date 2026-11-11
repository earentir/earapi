package health

import (
	"fmt"
	"regexp"
	"strings"
	"unicode"
)

var slugNonAlnum = regexp.MustCompile(`[^a-z0-9]+`)

// SlugID builds a URL-safe id from a display name.
func SlugID(name string) string {
	s := strings.ToLower(strings.TrimSpace(name))
	var b strings.Builder
	for _, r := range s {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			b.WriteRune(r)
		} else if r == ' ' || r == '-' || r == '/' {
			b.WriteByte('-')
		}
	}
	out := slugNonAlnum.ReplaceAllString(b.String(), "-")
	out = strings.Trim(out, "-")
	if out == "" {
		out = "item"
	}
	if len(out) > 80 {
		out = out[:80]
		out = strings.Trim(out, "-")
	}
	return out
}

// UniqueSlug picks slug or slug-2, slug-3, … if taken.
func UniqueSlug(name string, taken map[string]bool) string {
	base := SlugID(name)
	if !taken[base] {
		return base
	}
	for i := 2; i < 10000; i++ {
		cand := fmt.Sprintf("%s-%d", base, i)
		if !taken[cand] {
			return cand
		}
	}
	return base + "-x"
}
