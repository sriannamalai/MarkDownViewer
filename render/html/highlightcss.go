package htmlrender

import "github.com/sriannamalai/markdownviewer/theme"

// HighlightCSS returns the chroma syntax-highlighting stylesheet for t's
// ChromaStyle — the same class-based CSS a full page embeds for that
// theme's mode. Fragment hosts combine it with theme.BaseCSS and the
// theme's own CSS to get working highlighting; full-page output already
// includes it and does not need this function.
func HighlightCSS(t theme.Theme) (string, error) {
	return chromaCSS(t.ChromaStyle)
}
