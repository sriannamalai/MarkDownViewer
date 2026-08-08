// Package parser converts Markdown source into the document model. goldmark
// is an internal implementation detail and must not leak into the API.
//
// # Resource exhaustion
//
// Parse and ParseWith run goldmark's parser, whose list-nesting algorithm is
// super-quadratic in nesting depth: a few thousand levels of nested list
// items (an input on the order of tens of kilobytes) can take tens of
// seconds to parse. This is parse-time cost inside goldmark, so limiting
// input *size* does not bound it. Hosts that parse or render untrusted
// input should enforce a wall-clock timeout around the call (e.g. run it in
// a goroutine and select against a timer); context-aware Parse/Render
// variants that support cancellation are on the roadmap. See SECURITY.md
// for details.
package parser

import (
	"bytes"
	"unicode/utf8"

	"github.com/yuin/goldmark"
	emoji "github.com/yuin/goldmark-emoji"
	"github.com/yuin/goldmark/extension"
	gparser "github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/text"
	"go.abhg.dev/goldmark/frontmatter"
	"go.abhg.dev/goldmark/wikilink"

	"github.com/sriannamalai/markdownviewer/document"
)

// utf8BOM is the UTF-8 encoding of U+FEFF (byte order mark).
var utf8BOM = []byte{0xEF, 0xBB, 0xBF}

// invalidUTF8Replacement is substituted for each maximal run of invalid
// UTF-8 bytes in the source, mirroring what encoding/json's string encoder
// already does silently when a Text/CodeSpan/... value reaches MarshalJSON
// (see document/json.go). Sanitizing up front, before goldmark ever sees
// the bytes, keeps every string field the parser derives from the source
// already valid UTF-8, so JSON round-trips are lossless instead of
// silently diverging from the pre-JSON tree.
var invalidUTF8Replacement = []byte("�")

// stripBOM removes a leading UTF-8 byte order mark, if present. goldmark
// treats a BOM as ordinary text content rather than as whitespace, which
// otherwise breaks recognition of block-level constructs like "# Heading"
// on the first line.
func stripBOM(src []byte) []byte {
	return bytes.TrimPrefix(src, utf8BOM)
}

// sanitizeUTF8 replaces invalid UTF-8 byte sequences with the Unicode
// replacement character. Markdown source is not guaranteed to be valid
// UTF-8 (e.g. fuzzed or mis-decoded input), but every consumer downstream
// of this package — goldmark's own text handling, Span byte offsets, and
// the JSON codec in package document — assumes it is. The overwhelmingly
// common case is already-valid UTF-8, so check first and skip the copy
// bytes.ToValidUTF8 would otherwise do on every Parse call.
func sanitizeUTF8(src []byte) []byte {
	if utf8.Valid(src) {
		return src
	}
	return bytes.ToValidUTF8(src, invalidUTF8Replacement)
}

// Parse converts Markdown source into a document.Document using Default's
// extension set.
//
// See the package doc comment for the resource-exhaustion caveat on deeply
// nested list input.
func Parse(src []byte) (*document.Document, error) {
	return ParseWith(src, Default())
}

// ParseWith converts Markdown source into a document.Document using the
// syntax extensions selected by cfg.
//
// See the package doc comment for the resource-exhaustion caveat on deeply
// nested list input.
func ParseWith(src []byte, cfg Config) (*document.Document, error) {
	src = sanitizeUTF8(stripBOM(src))
	md := build(cfg)
	ctx := gparser.NewContext()
	root := md.Parser().Parse(text.NewReader(src), gparser.WithContext(ctx))
	t := &transformer{src: src, cfg: cfg, lines: newLineIndex(src), slugs: map[string]int{}}
	return t.document(root, ctx), nil
}

// build assembles the goldmark instance for cfg.
func build(cfg Config) goldmark.Markdown {
	var opts []goldmark.Option
	if exts := extensions(cfg); len(exts) > 0 {
		opts = append(opts, goldmark.WithExtensions(exts...))
	}
	return goldmark.New(opts...)
}

// extensions assembles the goldmark extension set enabled by cfg.
func extensions(cfg Config) []goldmark.Extender {
	var exts []goldmark.Extender
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
