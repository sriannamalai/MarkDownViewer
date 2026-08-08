// Package parser converts Markdown source into the document model. goldmark
// is an internal implementation detail and must not leak into the API.
package parser

import (
	"github.com/yuin/goldmark"
	emoji "github.com/yuin/goldmark-emoji"
	"github.com/yuin/goldmark/extension"
	gparser "github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/text"
	"go.abhg.dev/goldmark/frontmatter"
	"go.abhg.dev/goldmark/wikilink"

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
	if cfg.Tables {
		exts = append(exts, extension.Table)
	}
	if cfg.Strikethrough {
		exts = append(exts, extension.Strikethrough)
	}
	if cfg.TaskLists {
		exts = append(exts, extension.TaskList)
	}
	if cfg.Linkify {
		exts = append(exts, extension.Linkify)
	}
	if cfg.Footnotes {
		exts = append(exts, extension.Footnote)
	}
	if cfg.DefinitionLists {
		exts = append(exts, extension.DefinitionList)
	}
	if cfg.FrontMatter {
		exts = append(exts, &frontmatter.Extender{})
	}
	if cfg.Emoji {
		exts = append(exts, emoji.Emoji)
	}
	if cfg.WikiLinks {
		exts = append(exts, &wikilink.Extender{})
	}
	if cfg.Math {
		exts = append(exts, mathExt{})
	}
	return exts
}
