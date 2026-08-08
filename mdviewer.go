// Package markdownviewer is an embeddable Markdown previewer: it parses
// Markdown (CommonMark + GFM + modern extensions) into a document model and
// renders themed, self-contained, sanitized HTML.
//
// Quick start:
//
//	html, err := markdownviewer.Render(src)
//
// See the document package for the AST and render/html for renderer options.
//
// # Concurrency
//
// All top-level functions in this package (Parse, Render, RenderTo) are
// safe for concurrent use: they share no mutable state across calls beyond
// two package-level values that are constructed once and never mutated
// afterward — bluemonday's sanitizer Policy (its README documents Sanitize
// as safe to call concurrently on a constructed Policy; only
// construction/editing is not) and chroma's HTML formatter (its style
// cache is internally mutex-protected for concurrent Format calls). A
// *document.Document returned by Parse must not be mutated concurrently
// with a Render/RenderTo call that reads it, or with another mutation —
// document.Document itself has no internal synchronization.
//
// # Resource exhaustion
//
// Parse/Render/RenderTo have no built-in wall-clock or work budget. A
// deeply nested list (thousands of levels) can take goldmark's parser tens
// of seconds even though the input itself is small, because the cost is
// super-quadratic in nesting depth rather than in input size. Hosts that
// process untrusted input should wrap these calls in their own wall-clock
// timeout. See SECURITY.md for measurements and the recommended pattern.
package markdownviewer

import (
	"bytes"
	"io"

	"github.com/sriannamalai/markdownviewer/document"
	"github.com/sriannamalai/markdownviewer/parser"
	htmlrender "github.com/sriannamalai/markdownviewer/render/html"
)

// ResolveKind identifies which kind of target a Resolver is being asked to
// resolve: a standard link, an image, or a wiki-link.
type ResolveKind = htmlrender.ResolveKind

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

// Fragment emits body-only HTML instead of a full page: no <html>/<head>,
// no theme <style>, no embedded mermaid/KaTeX <script> assets. The host
// owns the page and is responsible for supplying styling — see the
// Theming section of README.md for the CSS variables the markup expects.
//
// Diagram and math nodes still render their markup (a <pre class="mermaid">
// block, a <span>/<div class="math ...">), but as inert placeholders: with
// no mermaid.js/KaTeX loaded, the raw diagram/math source is what a viewer
// sees until the host provides those libraries itself. The embedded copies
// this package bundles for full-page output are not currently exported for
// fragment hosts to reuse (see docs/Design.md's Roadmap).
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
// Keys must match the pattern --[a-zA-Z0-9_-]+; non-conforming keys are silently
// dropped. Overrides are emitted after the base theme in sorted key order, ensuring
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
