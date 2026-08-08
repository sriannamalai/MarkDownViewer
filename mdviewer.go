// Package markdownviewer is an embeddable Markdown previewer: it parses
// Markdown (CommonMark + GFM + modern extensions) into a stable document
// model and renders themed, self-contained, sanitized HTML.
//
// Quick start:
//
//	html, err := markdownviewer.Render(src)
//
// See the document package for the AST and render/html for renderer options.
package markdownviewer

import (
	"bytes"
	"io"

	"github.com/sriannamalai/markdownviewer/document"
	"github.com/sriannamalai/markdownviewer/parser"
	htmlrender "github.com/sriannamalai/markdownviewer/render/html"
)

// Resolver is a function that rewrites link and image targets. It accepts the
// resolution kind (link, image, or wiki-link) and target URL, returning the
// rewritten URL and true if resolution succeeded, or false to fall back to
// default handling.
//
// Trust contract: URLs returned with ok=true are emitted as-is without scheme
// filtering. Hosts fully control resolution and must not echo untrusted
// targets back unexamined.
type Resolver = htmlrender.Resolver

const (
	// ResolveLink signals resolution of a standard Markdown link target.
	ResolveLink = htmlrender.ResolveLink
	// ResolveImage signals resolution of a Markdown image target.
	ResolveImage = htmlrender.ResolveImage
	// ResolveWikiLink signals resolution of a wiki-link target.
	ResolveWikiLink = htmlrender.ResolveWikiLink
)

type config struct {
	render htmlrender.Options
}

// Option is a functional option that modifies render configuration.
type Option func(*config)

// WithTheme sets the theme ("light", "dark", or "auto").
func WithTheme(name string) Option {
	return func(c *config) { c.render.ThemeName = name }
}

// Fragment emits body-only HTML instead of a full page.
func Fragment() Option {
	return func(c *config) { c.render.Fragment = true }
}

// AllowRawHTML permits raw HTML and unsafe URL schemes in the output.
func AllowRawHTML() Option {
	return func(c *config) { c.render.Unsafe = true }
}

// DisableMermaid disables mermaid diagram rendering.
func DisableMermaid() Option {
	return func(c *config) { c.render.Mermaid = false }
}

// DisableMath disables KaTeX mathematical notation rendering.
func DisableMath() Option {
	return func(c *config) { c.render.Math = false }
}

// DisableHighlighting disables syntax highlighting in code blocks.
func DisableHighlighting() Option {
	return func(c *config) { c.render.Highlight = false }
}

// WithResolver provides a custom link/image resolution callback.
func WithResolver(r Resolver) Option {
	return func(c *config) { c.render.Resolver = r }
}

// WithThemeOverrides applies CSS custom-property overrides to the rendered theme.
// Keys should be property names (e.g., "--md-bg"), values the CSS values.
// Overrides are emitted after the base theme in sorted key order, ensuring
// they win in both light and dark variants. Content is emitted into the page's
// <style> element; </style sequences are stripped defensively.
func WithThemeOverrides(vars map[string]string) Option {
	return func(c *config) { c.render.ThemeOverrides = vars }
}

// WithStylesheet replaces the base stylesheet entirely with the provided CSS.
// When non-empty, it overrides theme.BaseCSS() and takes precedence over
// theme-based styling. Content is emitted into the page's <style> element;
// </style sequences are stripped defensively.
func WithStylesheet(css string) Option {
	return func(c *config) { c.render.Stylesheet = css }
}

// Parse returns the document model for src.
func Parse(src []byte) (*document.Document, error) { return parser.Parse(src) }

// Render parses src and renders HTML with the given options.
func Render(src []byte, opts ...Option) ([]byte, error) {
	var buf bytes.Buffer
	if err := RenderTo(&buf, src, opts...); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// RenderTo renders into w.
func RenderTo(w io.Writer, src []byte, opts ...Option) error {
	cfg := &config{render: htmlrender.DefaultOptions()}
	for _, o := range opts {
		o(cfg)
	}
	doc, err := parser.Parse(src)
	if err != nil {
		return err
	}
	return htmlrender.Render(w, doc, cfg.render)
}
