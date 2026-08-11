// Package boundary implements the shared JSON boundary consumed by both
// exported entry points — the cgo FFI (ffi/) and the wasm build (wasm/):
// strict version-1 options decoding plus markdown/document/HTML
// conversions and the embedded static asset registry. It has no cgo or
// wasm dependency itself so it can be linked into either main package
// unmodified.
package boundary

import (
	"fmt"

	markdownviewer "github.com/sriannamalai/markdownviewer"
	"github.com/sriannamalai/markdownviewer/assets"
	"github.com/sriannamalai/markdownviewer/document"
	htmlrender "github.com/sriannamalai/markdownviewer/render/html"
	"github.com/sriannamalai/markdownviewer/theme"
)

// Render renders markdown to HTML per the strict version-1 options JSON;
// resolver may be nil for default resolution.
func Render(md, optsJSON []byte, resolver markdownviewer.Resolver) ([]byte, error) {
	o, err := decodeOptions(optsJSON)
	if err != nil {
		return nil, err
	}
	return markdownviewer.Render(md, o.toFacadeOptions(resolver)...)
}

// Parse parses markdown into version-1 document JSON. Options are
// validated for consistency with the other calls, but no current field
// affects parsing.
func Parse(md, optsJSON []byte) ([]byte, error) {
	if _, err := decodeOptions(optsJSON); err != nil {
		return nil, err
	}
	doc, err := markdownviewer.Parse(md)
	if err != nil {
		return nil, err
	}
	return document.MarshalJSON(doc)
}

// RenderDoc renders a version-1 document JSON to HTML per the strict
// version-1 options JSON; resolver may be nil for default resolution.
func RenderDoc(docJSON, optsJSON []byte, resolver markdownviewer.Resolver) ([]byte, error) {
	o, err := decodeOptions(optsJSON)
	if err != nil {
		return nil, err
	}
	doc, err := document.UnmarshalJSON(docJSON)
	if err != nil {
		return nil, err
	}
	return markdownviewer.RenderDoc(doc, o.toFacadeOptions(resolver)...)
}

// Asset returns an embedded static asset by registry name. The registry
// is append-only (like document Kind values); names are case-sensitive.
// theme-*.css are composed: theme tokens plus that
// theme's chroma highlighting CSS — what a full page embeds for the
// mode — so fragment hosts get working syntax highlighting by applying
// the one file. Standalone theme-dark.css needs no light-rule
// neutralization; that pairing concern exists only inside full pages.
func Asset(name string) ([]byte, error) {
	switch name {
	case "mermaid.js":
		return []byte(assets.MermaidJS()), nil
	case "katex.js":
		return []byte(assets.KatexJS()), nil
	case "katex.css":
		return []byte(assets.KatexCSS()), nil
	case "base.css":
		return []byte(theme.BaseCSS()), nil
	case "theme-light.css":
		return composedThemeCSS(theme.Light())
	case "theme-dark.css":
		return composedThemeCSS(theme.Dark())
	}
	return nil, fmt.Errorf("unknown asset %q (valid: base.css, katex.css, katex.js, mermaid.js, theme-dark.css, theme-light.css)", name)
}

func composedThemeCSS(t theme.Theme) ([]byte, error) {
	hl, err := htmlrender.HighlightCSS(t)
	if err != nil {
		return nil, err
	}
	return []byte(t.CSS(":root") + "\n" + hl), nil
}
