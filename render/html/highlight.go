package htmlrender

import (
	"io"
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
