package htmlrender

import (
	"crypto/sha256"
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

// highlightStyleName is the single chroma style code-block HTML is formatted
// against. It is fixed per build: the formatter runs in classes mode, so the
// emitted markup carries class names only, and dark-mode rendering is pure
// CSS layering on top of the same markup (see page.go and
// neutralizeMissingClasses). This constant is why the highlight cache key
// omits style identity.
const highlightStyleName = "github"

// tokenise produces chroma's token stream for (code, lang) — the expensive
// step (lexer selection + regexp2 tokenisation dominates render CPU and
// allocations). Returns nil when lang is empty/unknown or the lexer errors,
// mirroring highlight's fallback-to-plain semantics.
//
// This tokenise/format split is the native-render seam: a future native
// renderer consumes this token stream directly as (text, style) runs,
// without the HTML formatter below.
func tokenise(code, lang string) chroma.Iterator {
	if lang == "" {
		return nil
	}
	lexer := lexers.Get(lang)
	if lexer == nil {
		return nil
	}
	lexer = chroma.Coalesce(lexer)
	it, err := lexer.Tokenise(nil, code)
	if err != nil {
		return nil
	}
	return it
}

// formatTokens renders a chroma token stream as classes-mode HTML against
// the package-fixed style. This is the cheap half of the split.
func formatTokens(it chroma.Iterator) (string, bool) {
	var b strings.Builder
	if chromaFormatter.Format(&b, styles.Get(highlightStyleName), it) != nil {
		return "", false
	}
	return b.String(), true
}

// highlightHTML returns the highlighted HTML for (code, lang), consulting
// the bounded package-level cache first. Returns ok=false when the language
// is unknown or chroma fails, so the caller can fall back to plain
// rendering; failures are never cached.
func highlightHTML(code, lang string) (string, bool) {
	key := highlightKey{lang: lang, sum: sha256.Sum256([]byte(code))}
	if html, ok := hlCache.get(key); ok {
		return html, true
	}
	it := tokenise(code, lang)
	if it == nil {
		return "", false
	}
	html, ok := formatTokens(it)
	if !ok {
		return "", false
	}
	hlCache.put(key, html)
	return html, true
}

// highlight writes chroma-highlighted HTML for code; returns false when the
// language is unknown so the caller can fall back to plain rendering.
func highlight(w io.Writer, code, lang string) bool {
	html, ok := highlightHTML(code, lang)
	if !ok {
		return false
	}
	_, err := io.WriteString(w, html)
	return err == nil
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
