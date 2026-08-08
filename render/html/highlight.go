package htmlrender

import (
	"io"
	"regexp"
	"sort"
	"strings"

	"github.com/alecthomas/chroma/v2"
	chromahtml "github.com/alecthomas/chroma/v2/formatters/html"
	"github.com/alecthomas/chroma/v2/lexers"
	"github.com/alecthomas/chroma/v2/styles"
)

var chromaFormatter = chromahtml.New(chromahtml.WithClasses(true))

// highlight writes chroma-highlighted HTML for code; returns false when the
// language is unknown so the caller can fall back to plain rendering.
func highlight(w io.Writer, code, lang string) bool {
	if lang == "" {
		return false
	}
	lexer := lexers.Get(lang)
	if lexer == nil {
		return false
	}
	lexer = chroma.Coalesce(lexer)
	it, err := lexer.Tokenise(nil, code)
	if err != nil {
		return false
	}
	return chromaFormatter.Format(w, styles.Get("github"), it) == nil
}

// chromaCSS returns class-based CSS for a chroma style ("github",
// "github-dark"). Used by full-page assembly.
func chromaCSS(styleName string) (string, error) {
	style := styles.Get(styleName)
	var b strings.Builder
	if err := chromaFormatter.WriteCSS(&b, style); err != nil {
		return "", err
	}
	return b.String(), nil
}

// chromaTokenClassRe matches a chroma per-token CSS rule's selector, e.g.
// the ".nx" in ".chroma .nx { color: ... }".
var chromaTokenClassRe = regexp.MustCompile(`\.chroma \.([A-Za-z0-9]+)\b`)

// chromaTokenClasses returns the set of chroma token class names (e.g. "kd",
// "nx", "s") that css defines rules for.
func chromaTokenClasses(css string) map[string]bool {
	classes := map[string]bool{}
	for _, m := range chromaTokenClassRe.FindAllStringSubmatch(css, -1) {
		classes[m[1]] = true
	}
	return classes
}

// neutralizeMissingClasses returns CSS that resets color/background-color to
// "inherit" for every chroma token class that other defines but css does
// not.
//
// Two chroma styles rarely style the exact same set of token types — e.g.
// "github" (light) sets a color for NameOther (".nx", a plain identifier
// like a Go package name) but "github-dark" leaves it unstyled, relying on
// its ".chroma" wrapper's own inherited text color when rendered alone. In
// full-page assembly, the light chroma CSS is emitted unconditionally and
// the dark chroma CSS is layered underneath in a
// "prefers-color-scheme: dark" media query (or, for the explicit dark
// theme, would be layered under a light block if one existed). Because
// ".chroma .nx" has equal specificity in both blocks, a class the "other"
// (already-emitted) block styles but css leaves unstyled doesn't get a
// competing rule from css at all — the earlier block's rule survives
// unchallenged, e.g. light's near-black ".nx" text color surviving into a
// dark-background page and becoming unreadable.
//
// The fix is to make every class the other block defines get an explicit
// rule in css too, even when that rule just says "inherit" (falling back to
// css's own ".chroma" wrapper color) — same specificity, later in source,
// so it wins the tie instead of leaving the earlier block's rule standing.
func neutralizeMissingClasses(css, other string) string {
	have := chromaTokenClasses(css)
	want := chromaTokenClasses(other)
	var missing []string
	for cls := range want {
		if !have[cls] {
			missing = append(missing, cls)
		}
	}
	if len(missing) == 0 {
		return ""
	}
	sort.Strings(missing)
	var b strings.Builder
	for _, cls := range missing {
		b.WriteString(".chroma ." + cls + "{color:inherit;background-color:inherit}")
	}
	return b.String()
}
