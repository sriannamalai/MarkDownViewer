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
// All top-level functions in this package (Parse, ParseWith, Render,
// RenderTo, RenderDoc, RenderDocTo, ParseContext, RenderContext,
// RenderDocContext) are safe for concurrent use: they share
// no mutable state across calls beyond two package-level values that are
// constructed once and never mutated afterward — bluemonday's sanitizer
// Policy (its README documents Sanitize as safe to call concurrently on a
// constructed Policy; only construction/editing is not) and chroma's HTML
// formatter (its style cache is internally mutex-protected for concurrent
// Format calls). A *document.Document returned by Parse/ParseWith must not
// be mutated concurrently with a Render/RenderTo/RenderDoc/RenderDocTo call
// that reads it, or with another mutation — document.Document itself has no
// internal synchronization.
//
// ParseContext/RenderContext/RenderDocContext extend this contract: when
// ctx ends before the underlying work finishes, the function returns
// ctx.Err() immediately, but the abandoned goroutine may still be reading
// src (ParseContext, RenderContext) or doc (RenderDocContext) for an
// unbounded window afterward — there is no signal for when it actually
// stops. Getting ctx.Err() back is not a guarantee that those inputs are
// safe to mutate or reuse; callers must treat src/doc passed to a Context
// variant as immutable for the lifetime of the call, and must not assume
// that lifetime has ended just because the call returned.
//
// # Resource exhaustion
//
// Parse/ParseWith/Render/RenderTo have no built-in wall-clock or work
// budget. A deeply nested list (thousands of levels) can take goldmark's
// parser tens of seconds even though the input itself is small, because the
// cost is super-quadratic in nesting depth rather than in input size. Hosts
// that process untrusted input should wrap these calls in their own
// wall-clock timeout. See SECURITY.md for measurements and the recommended
// pattern. RenderDoc/RenderDocTo receive an already-parsed tree and so are
// not subject to the parse-time cost, but rendering a pathologically deep
// tree built by other means can still be costly.
//
// ParseContext/RenderContext/RenderDocContext implement exactly this
// timeout pattern for the caller: they honor ctx's deadline/cancellation
// and return promptly, but the abandoned goroutine keeps running to
// completion — its CPU is not reclaimed. See SECURITY.md.
package markdownviewer

import (
	"bytes"
	"context"
	"io"
	"strings"

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
	render   htmlrender.Options
	parse    parser.Config
	parseSet bool
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
// sees until the host provides those libraries itself. The same mermaid.js
// and KaTeX JS/CSS this package embeds for full-page output are available
// to fragment hosts via the assets package (assets.MermaidJS, assets.KatexJS,
// assets.KatexCSS) — see README.md's Theming section for the injection
// snippet.
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

// WithMaxWidth constrains the rendered page's content width, via the
// --md-max-width CSS custom property (theme.BaseCSS's
// "max-width: var(--md-max-width, none)"). width accepts any CSS length,
// e.g. "860px" or "70ch". The default is fluid: an empty string (or never
// calling this option) leaves --md-max-width unset, so the page has no
// max-width constraint and fills its container.
//
// width flows into a CSS custom-property value inside the page's <style>
// element. As a defense-in-depth measure on top of the usual </style
// stripping applied to all theme override values, a width containing ';' or
// '}' — either of which could break out of the single declaration this
// option controls — is rejected outright: the option no-ops, leaving
// whatever width was already configured (or the fluid default) unchanged,
// rather than emitting a truncated or defanged value.
//
// Implemented via the same mechanism as WithThemeOverrides (it sets the
// "--md-max-width" key), so calling WithThemeOverrides after WithMaxWidth
// replaces the whole override map, including this key; call WithMaxWidth
// after WithThemeOverrides (or fold "--md-max-width" into that map
// directly) if you need both.
func WithMaxWidth(width string) Option {
	return func(c *config) {
		if width == "" || strings.ContainsAny(width, ";}") {
			return
		}
		if c.render.ThemeOverrides == nil {
			c.render.ThemeOverrides = map[string]string{}
		}
		c.render.ThemeOverrides["--md-max-width"] = width
	}
}

// WithSourceMap annotates top-level block elements with data-md-line
// attributes for editor↔preview scroll synchronization.
func WithSourceMap() Option {
	return func(c *config) { c.render.SourceMap = true }
}

// WithStylesheet replaces the base stylesheet entirely with the provided CSS.
// When non-empty, it overrides theme.BaseCSS() and takes precedence over
// theme-based styling. Content is emitted into the page's <style> element;
// </style sequences are stripped defensively.
func WithStylesheet(css string) Option {
	return func(c *config) { c.render.Stylesheet = css }
}

// WithExtraCSS appends css after the page's base styling — the base+theme
// stylesheets by default, or the WithStylesheet replacement when one is
// set. Full-page rendering only, like WithStylesheet; it has no effect in
// fragment mode. Content is emitted into the page's <style> element;
// </style sequences are stripped defensively.
func WithExtraCSS(css string) Option {
	return func(c *config) { c.render.ExtraCSS = css }
}

// WithCodeHeader wraps each rendered code block in a header row carrying a
// language label (the fence language, or "code" when unlabeled) and a Copy
// button, using the md-code / md-code-header / md-code-lang / md-code-copy
// classes styled by the base stylesheet. Live mermaid diagrams and math are
// not wrapped; their engine-disabled plain-code fallbacks are. Full pages
// also get a small inline clipboard script wiring the buttons; fragment
// hosts receive the markup and wire their own handler.
func WithCodeHeader() Option {
	return func(c *config) { c.render.CodeHeader = true }
}

// WithParserConfig selects which Markdown extensions the parser enables
// when Render/RenderTo (and their Context variants) parse the source.
// It has no effect on RenderDoc, which receives an already-parsed tree.
func WithParserConfig(cfg parser.Config) Option {
	return func(c *config) { c.parse, c.parseSet = cfg, true }
}

// newConfig folds opts into a config seeded with default render options,
// shared by RenderTo and RenderDocTo so option-folding happens exactly once.
func newConfig(opts []Option) *config {
	cfg := &config{render: htmlrender.DefaultOptions()}
	for _, o := range opts {
		o(cfg)
	}
	return cfg
}

// Parse returns the document model for src.
func Parse(src []byte) (*document.Document, error) { return parser.Parse(src) }

// ParseWith parses src with an explicit extension configuration.
func ParseWith(src []byte, cfg parser.Config) (*document.Document, error) {
	return parser.ParseWith(src, cfg)
}

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
	cfg := newConfig(opts)
	parseCfg := parser.Default()
	if cfg.parseSet {
		parseCfg = cfg.parse
	}
	doc, err := parser.ParseWith(src, parseCfg)
	if err != nil {
		return err
	}
	return htmlrender.Render(w, doc, cfg.render)
}

// RenderDoc renders an already-parsed document, enabling parse-once /
// render-many workflows such as switching themes without re-parsing.
func RenderDoc(doc *document.Document, opts ...Option) ([]byte, error) {
	var buf bytes.Buffer
	if err := RenderDocTo(&buf, doc, opts...); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// RenderDocTo renders an already-parsed document into w.
func RenderDocTo(w io.Writer, doc *document.Document, opts ...Option) error {
	cfg := newConfig(opts)
	return htmlrender.Render(w, doc, cfg.render)
}

// ParseContext is Parse with caller-side deadline support. If ctx ends
// first, ParseContext returns ctx.Err() immediately — but the underlying
// parse continues on its goroutine until it finishes and cannot be
// stopped (the Markdown engine has no cancellation hooks). The guarantee
// is bounded caller latency, not reclaimed CPU; see SECURITY.md. The
// abandoned goroutine may still be reading src after this function
// returns, for an unbounded window — do not mutate or reuse src until you
// can otherwise guarantee that goroutine has finished (see the package's
// Concurrency section).
func ParseContext(ctx context.Context, src []byte) (*document.Document, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	type result struct {
		doc *document.Document
		err error
	}
	ch := make(chan result, 1)
	go func() {
		d, err := Parse(src)
		ch <- result{d, err}
	}()
	select {
	case r := <-ch:
		return r.doc, r.err
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

// RenderContext is Render with caller-side deadline support (same
// abandonment semantics as ParseContext, including that src must not be
// mutated or reused after a cancelled return until the abandoned
// goroutine is otherwise known to have finished).
func RenderContext(ctx context.Context, src []byte, opts ...Option) ([]byte, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	type result struct {
		out []byte
		err error
	}
	ch := make(chan result, 1)
	go func() {
		out, err := Render(src, opts...)
		ch <- result{out, err}
	}()
	select {
	case r := <-ch:
		return r.out, r.err
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

// RenderDocContext is RenderDocTo with caller-side deadline support. The
// render happens into an internal buffer; w is written only on success,
// so an abandoned render never touches w after this function returns. As
// with ParseContext/RenderContext, the abandoned goroutine may still be
// reading doc for an unbounded window after a cancelled return — its CPU
// is not reclaimed, and doc must not be mutated or reused until you can
// otherwise guarantee that goroutine has finished; see SECURITY.md.
func RenderDocContext(ctx context.Context, w io.Writer, doc *document.Document, opts ...Option) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	type result struct {
		out []byte
		err error
	}
	ch := make(chan result, 1)
	go func() {
		out, err := RenderDoc(doc, opts...)
		ch <- result{out, err}
	}()
	select {
	case r := <-ch:
		if r.err != nil {
			return r.err
		}
		_, err := w.Write(r.out)
		return err
	case <-ctx.Done():
		return ctx.Err()
	}
}
