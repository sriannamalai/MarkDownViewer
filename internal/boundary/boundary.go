// Package boundary implements the shared JSON boundary consumed by both
// exported entry points — the cgo FFI (ffi/) and the wasm build (wasm/):
// strict version-1 options decoding plus markdown/document/HTML
// conversions and the embedded static asset registry. It has no cgo or
// wasm dependency itself so it can be linked into either main package
// unmodified.
package boundary

import (
	"encoding/json"
	"fmt"

	"github.com/alecthomas/chroma/v2"
	"github.com/alecthomas/chroma/v2/styles"
	markdownviewer "github.com/sriannamalai/markdownviewer"
	"github.com/sriannamalai/markdownviewer/assets"
	"github.com/sriannamalai/markdownviewer/document"
	htmlrender "github.com/sriannamalai/markdownviewer/render/html"
	"github.com/sriannamalai/markdownviewer/render/tree"
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

// Parse parses markdown into version-1 document JSON. The nested
// "parser" options object selects the syntax-extension set; every other
// field is validated for consistency with the other calls but does not
// affect parsing.
func Parse(md, optsJSON []byte) ([]byte, error) {
	o, err := decodeOptions(optsJSON)
	if err != nil {
		return nil, err
	}
	var doc *document.Document
	if o.Parser != nil {
		doc, err = markdownviewer.ParseWith(md, o.Parser.toConfig())
	} else {
		doc, err = markdownviewer.Parse(md)
	}
	if err != nil {
		return nil, err
	}
	return document.MarshalJSON(doc)
}

// RenderDoc renders a version-1 document JSON to HTML per the strict
// version-1 options JSON; resolver may be nil for default resolution.
// The nested "parser" options object is decoded (and validated) but has
// no effect here: the document is already parsed, and parser config is a
// parse-time concern (same precedent as theme being ignored by Parse).
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

// RenderTree parses markdown and builds the version-1 native render
// tree as strict JSON (the render/tree wire schema) per the strict
// version-1 options JSON; resolver may be nil for default resolution.
// The nested "parser" options object selects the syntax-extension set,
// exactly as in Parse; the option→tree mapping (and which HTML-only
// fields are decoded and ignored) is documented on
// options.toTreeOptions. Block ids are content hashes over the markdown
// source (see "Block identity" in the render/tree package docs).
func RenderTree(md, optsJSON []byte, resolver markdownviewer.Resolver) ([]byte, error) {
	o, err := decodeOptions(optsJSON)
	if err != nil {
		return nil, err
	}
	var doc *document.Document
	if o.Parser != nil {
		doc, err = markdownviewer.ParseWith(md, o.Parser.toConfig())
	} else {
		doc, err = markdownviewer.Parse(md)
	}
	if err != nil {
		return nil, err
	}
	t, err := tree.Build(doc, o.toTreeOptions(resolver, md))
	if err != nil {
		return nil, err
	}
	return t.MarshalJSON()
}

// RenderTreeDoc builds the version-1 render tree from version-1
// document JSON (as produced by Parse). Same options semantics as
// RenderTree, with two doc-path differences: the nested "parser" object
// is decoded (and validated) but has no effect — the document is
// already parsed (same precedent as RenderDoc) — and, with no markdown
// source at hand, block ids take the deterministic kind+ordinal
// fallback form instead of content hashes (see "Block identity" in the
// render/tree package docs).
func RenderTreeDoc(docJSON, optsJSON []byte, resolver markdownviewer.Resolver) ([]byte, error) {
	o, err := decodeOptions(optsJSON)
	if err != nil {
		return nil, err
	}
	doc, err := document.UnmarshalJSON(docJSON)
	if err != nil {
		return nil, err
	}
	t, err := tree.Build(doc, o.toTreeOptions(resolver, nil))
	if err != nil {
		return nil, err
	}
	return t.MarshalJSON()
}

// Asset returns an embedded static asset by registry name. The registry
// is append-only (like document Kind values); names are case-sensitive.
// theme-*.css are composed: theme tokens plus that
// theme's chroma highlighting CSS — what a full page embeds for the
// mode — so fragment hosts get working syntax highlighting by applying
// the one file. Standalone theme-dark.css needs no light-rule
// neutralization; that pairing concern exists only inside full pages.
// theme-*.json carry the same palette as data (see themeJSON) so native
// hosts don't have to parse CSS to recover colors, and
// highlight-*.json carry the matching chroma styles' token colors as
// data (see highlightJSON) for hosts styling render-tree token runs.
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
	case "theme-light.json":
		return themeJSONAsset(theme.Light())
	case "theme-dark.json":
		return themeJSONAsset(theme.Dark())
	case "highlight-light.json":
		return highlightJSONAsset(theme.Light())
	case "highlight-dark.json":
		return highlightJSONAsset(theme.Dark())
	}
	return nil, fmt.Errorf("unknown asset %q (valid: base.css, highlight-dark.json, highlight-light.json, katex.css, katex.js, mermaid.js, theme-dark.css, theme-dark.json, theme-light.css, theme-light.json)", name)
}

func composedThemeCSS(t theme.Theme) ([]byte, error) {
	hl, err := htmlrender.HighlightCSS(t)
	if err != nil {
		return nil, err
	}
	return []byte(t.CSS(":root") + "\n" + hl), nil
}

// themeJSON is the version-1 shape of the theme-*.json assets: the
// theme's --md-* palette as data, for native hosts that render without
// CSS. Generated from the same theme.Theme structs that produce
// theme-*.css, so the two representations cannot drift. Output is
// deterministic: struct fields marshal in declaration order and
// encoding/json emits map keys sorted.
type themeJSON struct {
	Version     int               `json:"version"`
	Mode        string            `json:"mode"` // "light" or "dark"
	ChromaStyle string            `json:"chromaStyle"`
	Vars        map[string]string `json:"vars"` // theme.Theme.Vars verbatim
}

func themeJSONAsset(t theme.Theme) ([]byte, error) {
	return json.Marshal(themeJSON{
		Version:     1,
		Mode:        t.Name,
		ChromaStyle: t.ChromaStyle,
		Vars:        t.Vars,
	})
}

// highlightJSON is the version-1 shape of the highlight-*.json assets:
// chroma token-type name → foreground color, for hosts styling the
// render tree's code token runs (CodeBlock runs carry
// TokenType.String() names — the same names used as keys here). Only
// token types the style resolves a foreground color for are present;
// hosts fall back to their default text color for any missing type.
type highlightJSON struct {
	Version int               `json:"version"`
	Style   string            `json:"style"`  // chroma style name, e.g. "github"
	Colors  map[string]string `json:"colors"` // TokenType.String() → "#rrggbb"
}

// highlightJSONAsset resolves the theme's paired chroma style into
// token colors — the SAME styles.Get(t.ChromaStyle) object the CSS path
// (htmlrender.HighlightCSS via the chroma HTML formatter) formats
// against, walked the same way that formatter's WriteCSS is: every
// standard token type (chroma.StandardTypes, not just the style's
// direct entries — the CSS expands category inheritance the same way,
// e.g. github-dark colors LiteralStringSingle ".s1" via its Literal
// entry), resolved through Style.Get's inheritance and stripped of the
// Background entry's fields for non-background types. Data and CSS
// therefore cannot drift, and a run's concrete tokenType is always a
// direct key — hosts never need chroma's category-fallback walk.
// Colors are chroma.Colour.String() hex-normalized "#rrggbb". Output is
// deterministic: struct fields marshal in declaration order and
// encoding/json emits map keys sorted.
func highlightJSONAsset(t theme.Theme) ([]byte, error) {
	style := styles.Get(t.ChromaStyle)
	bg := style.Get(chroma.Background)
	colors := map[string]string{}
	for tt := range chroma.StandardTypes {
		entry := style.Get(tt)
		if tt != chroma.Background {
			entry = entry.Sub(bg)
		}
		if entry.Colour.IsSet() {
			colors[tt.String()] = entry.Colour.String()
		}
	}
	return json.Marshal(highlightJSON{Version: 1, Style: t.ChromaStyle, Colors: colors})
}
