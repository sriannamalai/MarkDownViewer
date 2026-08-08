// Package parser converts Markdown source into the document model. goldmark
// is an internal implementation detail and must not leak into the API.
package parser

import (
	"github.com/yuin/goldmark"
	gparser "github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/text"

	"github.com/sriannamalai/markdownviewer/document"
)

func Parse(src []byte) (*document.Document, error) {
	return ParseWith(src, Default())
}

func ParseWith(src []byte, cfg Config) (*document.Document, error) {
	md := build(cfg)
	ctx := gparser.NewContext()
	root := md.Parser().Parse(text.NewReader(src), gparser.WithContext(ctx))
	t := &transformer{src: src, cfg: cfg, slugs: map[string]int{}}
	return t.document(root, ctx), nil
}

// build assembles the goldmark instance for cfg. Extensions are appended
// here in Tasks 3–8 as they are implemented.
func build(cfg Config) goldmark.Markdown {
	var opts []goldmark.Option
	if exts := extensions(cfg); len(exts) > 0 {
		opts = append(opts, goldmark.WithExtensions(exts...))
	}
	return goldmark.New(opts...)
}

func extensions(cfg Config) []goldmark.Extender {
	var exts []goldmark.Extender
	// Populated by Tasks 3–8.
	return exts
}
