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

type Resolver = htmlrender.Resolver

const (
	ResolveLink     = htmlrender.ResolveLink
	ResolveImage    = htmlrender.ResolveImage
	ResolveWikiLink = htmlrender.ResolveWikiLink
)

type config struct {
	render htmlrender.Options
}

type Option func(*config)

func WithTheme(name string) Option     { return func(c *config) { c.render.ThemeName = name } }
func Fragment() Option                 { return func(c *config) { c.render.Fragment = true } }
func AllowRawHTML() Option             { return func(c *config) { c.render.Unsafe = true } }
func DisableMermaid() Option           { return func(c *config) { c.render.Mermaid = false } }
func DisableMath() Option              { return func(c *config) { c.render.Math = false } }
func DisableHighlighting() Option      { return func(c *config) { c.render.Highlight = false } }
func WithResolver(r Resolver) Option   { return func(c *config) { c.render.Resolver = r } }

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
