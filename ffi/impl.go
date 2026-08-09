package main

import (
	"fmt"

	"github.com/sriannamalai/markdownviewer/assets"
	markdownviewer "github.com/sriannamalai/markdownviewer"
	"github.com/sriannamalai/markdownviewer/document"
	"github.com/sriannamalai/markdownviewer/theme"
	htmlrender "github.com/sriannamalai/markdownviewer/render/html"
)

// renderImpl is mdv_render behind the cgo boundary: markdown -> HTML.
func renderImpl(md, optsJSON []byte) ([]byte, error) {
	o, err := decodeOptions(optsJSON)
	if err != nil {
		return nil, err
	}
	return markdownviewer.Render(md, o.toFacadeOptions()...)
}

// parseImpl is mdv_parse: markdown -> version-1 document JSON. Options are
// validated for consistency with the other calls, but no current field
// affects parsing.
func parseImpl(md, optsJSON []byte) ([]byte, error) {
	if _, err := decodeOptions(optsJSON); err != nil {
		return nil, err
	}
	doc, err := markdownviewer.Parse(md)
	if err != nil {
		return nil, err
	}
	return document.MarshalJSON(doc)
}

// renderDocImpl is mdv_render_doc: version-1 document JSON -> HTML.
func renderDocImpl(docJSON, optsJSON []byte) ([]byte, error) {
	o, err := decodeOptions(optsJSON)
	if err != nil {
		return nil, err
	}
	doc, err := document.UnmarshalJSON(docJSON)
	if err != nil {
		return nil, err
	}
	return markdownviewer.RenderDoc(doc, o.toFacadeOptions()...)
}

// assetImpl is mdv_asset: returns an embedded static asset by registry
// name. The registry is append-only (like document Kind values); names
// are case-sensitive. theme-*.css are composed: theme tokens plus that
// theme's chroma highlighting CSS — what a full page embeds for the
// mode — so fragment hosts get working syntax highlighting by applying
// the one file. Standalone theme-dark.css needs no light-rule
// neutralization; that pairing concern exists only inside full pages.
func assetImpl(name string) ([]byte, error) {
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
