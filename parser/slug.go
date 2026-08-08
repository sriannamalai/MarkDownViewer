package parser

import (
	"fmt"
	"strings"
	"unicode"
)

// slugify produces a GitHub-style anchor slug: lowercase, spaces to hyphens,
// punctuation dropped, letters/digits/hyphen/underscore kept.
func slugify(s string) string {
	var b strings.Builder
	for _, r := range strings.TrimSpace(s) {
		switch {
		case unicode.IsLetter(r) || unicode.IsDigit(r) || r == '_' || r == '-':
			b.WriteRune(unicode.ToLower(r))
		case r == ' ':
			b.WriteByte('-')
		}
	}
	return b.String()
}

func (t *transformer) slug(text string) string {
	s := slugify(text)
	n, seen := t.slugs[s]
	t.slugs[s] = n + 1
	if seen {
		return fmt.Sprintf("%s-%d", s, n)
	}
	return s
}
