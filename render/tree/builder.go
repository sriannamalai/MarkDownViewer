package tree

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"strconv"

	"github.com/sriannamalai/markdownviewer/document"
	htmlrender "github.com/sriannamalai/markdownviewer/render/html"
	"github.com/sriannamalai/markdownviewer/render/internal/derive"
	"github.com/sriannamalai/markdownviewer/resolve"
)

// Options configures Build. The toggles mirror the HTML renderer's
// semantics exactly:
//
//   - Math/Mermaid off: math and mermaid nodes fall back to the same
//     shapes the HTML path falls back to — a MathBlock becomes a
//     CodeBlock with language "math", a Diagram becomes a CodeBlock
//     with the engine as its language, and a MathInline becomes a
//     CodeSpan of its source.
//   - HeadingAnchors off: headings carry no anchor id.
//   - AllowRawHTML: raw HTML nodes carry the raw markup with
//     Unsafe=true; otherwise the bluemonday-sanitized form with
//     Unsafe=false. Unlike the HTML renderer's single Unsafe flag,
//     AllowRawHTML governs ONLY raw HTML: URL policy (resolve.SafeURL)
//     always applies — a blocked destination is reported as url:"" +
//     blocked:true and the host decides what to do with it.
//   - Highlighting: code blocks carry token runs (CodeBlock.Runs) from
//     the same chroma tokenise seam the HTML renderer highlights
//     through (htmlrender.TokenRuns, cached there). Off — or on with an
//     unknown/missing language, or when chroma cannot reproduce the
//     code text exactly — leaves Runs nil (wire: null) and the host
//     renders Text plain, the tree analogue of the HTML renderer's
//     fallback-to-plain path.
type Options struct {
	Resolver       resolve.Resolver // optional hook to rewrite link/image/wiki-link targets; nil uses default resolution
	HeadingAnchors bool             // anchor ids on headings
	Highlighting   bool             // code token runs (see note above)
	Math           bool             // math nodes; off falls back to code shapes
	Mermaid        bool             // mermaid diagram nodes; off falls back to code blocks
	AllowRawHTML   bool             // carry raw HTML unsanitized (unsafe:true)

	// Source is the markdown source doc was parsed from, used to derive
	// content-hash block ids from node spans (see "Block identity" in
	// the package documentation). Empty or nil is valid — e.g. when
	// building from decoded doc JSON with no source at hand — and
	// switches every block to the deterministic kind+ordinal fallback
	// id.
	Source []byte
}

// DefaultOptions returns the recommended Options, matching the HTML
// renderer's defaults: highlighting, mermaid, math, and heading anchors
// enabled; raw HTML sanitized.
func DefaultOptions() Options {
	return Options{Highlighting: true, Mermaid: true, Math: true, HeadingAnchors: true}
}

// Build produces the resolved render tree for doc. It never mutates
// doc. All policy is applied here, through the same shared code paths
// the HTML renderer uses: URL resolution/filtering via derive.Href
// (resolve.Resolver trust contract included), raw-HTML sanitizing via
// derive.SanitizeHTML, admonition titles via derive.AdmonitionTitle,
// and footnote pairing via derive.Footnotes.
func Build(doc *document.Document, opts Options) (*Tree, error) {
	if doc == nil {
		return nil, errors.New("tree: nil document")
	}
	fns := derive.Footnotes(doc)
	b := &builder{opts: opts, footnoteDefID: previewFootnoteDefIDs(opts.Source, fns)}
	t := &Tree{Blocks: b.blocks(doc)}
	for _, fn := range fns {
		def := fn.Def
		t.Footnotes = append(t.Footnotes, Footnote{
			Span:     def.Span(),
			ID:       b.id(document.KindFootnoteDef, def.Span()),
			Index:    def.Index,
			RefCount: fn.RefCount,
			Blocks:   b.blocks(def),
		})
	}
	return t, nil
}

type builder struct {
	opts    Options
	ordinal int // document-order count of every emitted block (items and footnotes included)

	// footnoteDefID maps a footnote's Index to the block id its
	// Footnote entry will carry, so FootnoteRef inlines — built before
	// the trailing footnotes loop assigns real ids — can carry a
	// same-value DefID. See previewFootnoteDefIDs.
	footnoteDefID map[int]string
}

// contentHashID computes the content-hash form of a block id —
// hex(sha256(src[span.StartOffset:span.EndOffset]))[:16] — when src and
// span support it. It is a pure function of src+span (no ordinal
// dependency), which is what lets previewFootnoteDefIDs compute it
// ahead of the document-order traversal that assigns fallback ids.
func contentHashID(src []byte, span document.Span) (string, bool) {
	if len(src) > 0 && !span.IsZero() &&
		span.StartOffset >= 0 && span.StartOffset <= span.EndOffset && span.EndOffset <= len(src) {
		sum := sha256.Sum256(src[span.StartOffset:span.EndOffset])
		return hex.EncodeToString(sum[:8]), true
	}
	return "", false
}

// previewFootnoteDefIDs computes, ahead of the main build traversal,
// the block id each footnote definition's eventual Footnote entry will
// carry — so FootnoteRef.DefID (set while walking the document body,
// before any footnote definition is actually emitted) can carry the
// matching value. Only resolvable in the content-hash case (Source
// provided and the definition has a real span): the fallback
// kind+ordinal id depends on the ordinal position definitions are
// emitted at in the trailing footnotes loop, which isn't known yet
// here, so those entries are simply omitted — FootnoteRef.DefID stays
// "" for them, and hosts fall back to Tree.FootnoteByIndex. This never
// touches builder.ordinal, so it cannot perturb any other block's
// fallback id.
func previewFootnoteDefIDs(src []byte, fns []derive.Footnote) map[int]string {
	if len(fns) == 0 {
		return nil
	}
	ids := make(map[int]string, len(fns))
	for _, fn := range fns {
		if id, ok := contentHashID(src, fn.Def.Span()); ok {
			ids[fn.Def.Index] = id
		}
	}
	return ids
}

// id derives a block's identity: the content hash of its span's source
// bytes, or the kind+ordinal fallback when no source bytes are
// available for it. See "Block identity" in the package documentation.
// Every call advances the ordinal, so fallback ids stay aligned with
// the block's document-order position whether or not preceding blocks
// hashed by content.
func (b *builder) id(kind document.Kind, span document.Span) string {
	ord := b.ordinal
	b.ordinal++
	if id, ok := contentHashID(b.opts.Source, span); ok {
		return id
	}
	// The fallback preimage is domain-separated from the content-hash
	// form (which hashes raw span bytes): without the prefix, a block
	// whose source bytes literally spell e.g. "list:2" would collide
	// with a fallback id. NUL cannot appear in decoded markdown source
	// (invalid UTF-8 is replaced before parsing), so the prefix is
	// unreachable from content.
	sum := sha256.Sum256([]byte("\x00mdv-fallback\x00" + kind.String() + ":" + strconv.Itoa(ord)))
	return hex.EncodeToString(sum[:8])
}

func nodeSpan(n document.Node) document.Span {
	if sp, ok := n.(interface{ Span() document.Span }); ok {
		return sp.Span()
	}
	return document.Span{}
}

func (b *builder) blocks(n document.Node) []Block {
	out := make([]Block, 0, len(n.Children()))
	for _, c := range n.Children() {
		if blk := b.block(c); blk != nil {
			out = append(out, blk)
		}
	}
	return out
}

func (b *builder) block(n document.Node) Block {
	span := nodeSpan(n)
	switch n := n.(type) {
	case *document.Heading:
		anchor := ""
		if b.opts.HeadingAnchors {
			anchor = n.AnchorID
		}
		return &Heading{Span: span, ID: b.id(document.KindHeading, span),
			Level: n.Level, AnchorID: anchor, Children: b.inlines(n)}
	case *document.Paragraph:
		return &Paragraph{Span: span, ID: b.id(document.KindParagraph, span), Children: b.inlines(n)}
	case *document.BlockQuote:
		return &BlockQuote{Span: span, ID: b.id(document.KindBlockQuote, span), Blocks: b.blocks(n)}
	case *document.Admonition:
		variant, title := derive.AdmonitionTitle(n.Variant)
		return &Admonition{Span: span, ID: b.id(document.KindAdmonition, span),
			Variant: variant, Title: title, Blocks: b.blocks(n)}
	case *document.List:
		l := &List{Span: span, ID: b.id(document.KindList, span),
			Ordered: n.Ordered, Start: n.Start, Tight: n.Tight, Items: []ListItem{}}
		for _, c := range n.Children() {
			if li, ok := c.(*document.ListItem); ok {
				l.Items = append(l.Items, b.listItem(li))
				continue
			}
			// A List should only ever contain ListItem children; a
			// hand-built document that violates this keeps the stray
			// node as an untasked item wrapping it (the HTML renderer
			// likewise renders it through the normal block path).
			liSpan := nodeSpan(c)
			itemID := b.id(document.KindListItem, liSpan)
			if blk := b.block(c); blk != nil {
				l.Items = append(l.Items, ListItem{Span: liSpan, ID: itemID, Blocks: []Block{blk}})
			}
		}
		return l
	case *document.CodeBlock:
		return b.codeBlock(span, n.Language, n.Code)
	case *document.Diagram:
		if b.opts.Mermaid && n.Engine == "mermaid" {
			return &Diagram{Span: span, ID: b.id(document.KindDiagram, span),
				Engine: n.Engine, Source: n.Source}
		}
		return b.codeBlock(span, n.Engine, n.Source)
	case *document.MathBlock:
		if b.opts.Math {
			return &MathBlock{Span: span, ID: b.id(document.KindMathBlock, span), Source: n.Source}
		}
		return b.codeBlock(span, "math", n.Source)
	case *document.Table:
		return b.table(n, span)
	case *document.ThematicBreak:
		return &ThematicBreak{Span: span, ID: b.id(document.KindThematicBreak, span)}
	case *document.HTMLBlock:
		html, unsafe := b.rawHTML(n.HTML)
		return &HTMLBlock{Span: span, ID: b.id(document.KindHTMLBlock, span), HTML: html, Unsafe: unsafe}
	case *document.DefinitionList:
		dl := &DefinitionList{Span: span, ID: b.id(document.KindDefinitionList, span), Blocks: []Block{}}
		for _, c := range n.Children() {
			// Mirroring the HTML renderer, anything that isn't a
			// term/description is dropped.
			switch c := c.(type) {
			case *document.DefinitionTerm:
				cs := nodeSpan(c)
				dl.Blocks = append(dl.Blocks, &DefinitionTerm{Span: cs,
					ID: b.id(document.KindDefinitionTerm, cs), Children: b.inlines(c)})
			case *document.DefinitionDesc:
				cs := nodeSpan(c)
				dl.Blocks = append(dl.Blocks, &DefinitionDesc{Span: cs,
					ID: b.id(document.KindDefinitionDesc, cs), Blocks: b.blocks(c)})
			}
		}
		return dl
	default:
		// A stray inline at block level (possible in hand-built
		// documents; the HTML renderer emits it bare). Wrap it in an
		// implicit paragraph so the tree stays uniformly typed; unknown
		// nodes are dropped.
		if inl := b.inline(n); inl != nil {
			return &Paragraph{Span: span, ID: b.id(document.KindParagraph, span), Children: []Inline{inl}}
		}
		return nil
	}
}

// codeBlock builds a CodeBlock (directly, or as the math/mermaid-off
// fallback — the tree analogue of the HTML renderer's fallback to a
// plain code block). With Options.Highlighting and a known language it
// carries token runs from the shared tokenise seam; the cached slice is
// copied into the tree's own type so no Build output aliases the cache.
func (b *builder) codeBlock(span document.Span, language, code string) *CodeBlock {
	cb := &CodeBlock{Span: span, ID: b.id(document.KindCodeBlock, span),
		Language: language, Label: derive.CodeLabel(language), Text: code}
	if b.opts.Highlighting {
		if runs, ok := htmlrender.TokenRuns(code, language); ok {
			cb.Runs = make([]TokenRun, len(runs))
			for i, r := range runs {
				cb.Runs[i] = TokenRun{Text: r.Text, TokenType: r.TokenType}
			}
		}
	}
	return cb
}

func (b *builder) listItem(li *document.ListItem) ListItem {
	span := nodeSpan(li)
	item := ListItem{Span: span, ID: b.id(document.KindListItem, span), Blocks: b.blocks(li)}
	if li.Task {
		checked := li.Checked
		item.Task = &checked
	}
	return item
}

func (b *builder) table(t *document.Table, span document.Span) *Table {
	out := &Table{Span: span, ID: b.id(document.KindTable, span),
		Alignments: t.Alignments, Rows: [][]Cell{}}
	for _, rowNode := range t.Children() {
		row, ok := rowNode.(*document.TableRow)
		if !ok {
			continue // mirror the HTML renderer: skip non-row children
		}
		cells := make([]Cell, 0, len(row.Children()))
		for _, cellNode := range row.Children() {
			cell, ok := cellNode.(*document.TableCell)
			if !ok {
				continue // likewise skip non-cell children
			}
			cells = append(cells, Cell(b.inlines(cell)))
		}
		if row.Header {
			out.Header = cells
		} else {
			out.Rows = append(out.Rows, cells)
		}
	}
	return out
}

func (b *builder) inlines(n document.Node) []Inline {
	out := make([]Inline, 0, len(n.Children()))
	for _, c := range n.Children() {
		if inl := b.inline(c); inl != nil {
			out = append(out, inl)
		}
	}
	return out
}

func (b *builder) inline(n document.Node) Inline {
	span := nodeSpan(n)
	switch n := n.(type) {
	case *document.Text:
		return &Text{Span: span, Value: n.Value}
	case *document.SoftBreak:
		return &SoftBreak{}
	case *document.HardBreak:
		return &HardBreak{}
	case *document.Emphasis:
		return &Emphasis{Span: span, Children: b.inlines(n)}
	case *document.Strong:
		return &Strong{Span: span, Children: b.inlines(n)}
	case *document.Strikethrough:
		return &Strikethrough{Span: span, Children: b.inlines(n)}
	case *document.CodeSpan:
		return &CodeSpan{Span: span, Value: n.Value}
	case *document.Link:
		u, ok := b.href(resolve.ResolveLink, n.Destination)
		return &Link{Span: span, URL: u, Blocked: !ok, Title: n.Title, Children: b.inlines(n)}
	case *document.Image:
		u, ok := b.href(resolve.ResolveImage, n.Destination)
		return &Image{Span: span, URL: u, Blocked: !ok, Alt: n.Alt, Title: n.Title}
	case *document.WikiLink:
		u, ok := b.href(resolve.ResolveWikiLink, n.Target)
		return &Link{Span: span, URL: u, Blocked: !ok, Source: "wikiLink", Children: b.inlines(n)}
	case *document.MathInline:
		if b.opts.Math {
			return &MathInline{Span: span, Source: n.Source, Display: n.Display}
		}
		return &CodeSpan{Span: span, Value: n.Source}
	case *document.HTMLInline:
		html, unsafe := b.rawHTML(n.HTML)
		return &HTMLInline{Span: span, HTML: html, Unsafe: unsafe}
	case *document.FootnoteRef:
		return &FootnoteRef{Index: n.Index, DefID: b.footnoteDefID[n.Index]}
	default:
		return nil // unknown node: dropped
	}
}

// href runs the shared resolution pipeline (derive.Href). URL policy is
// always enforced here — the tree has no unsafe-URL mode; blocked
// destinations surface as ("", false) → url:"" + blocked:true.
func (b *builder) href(kind resolve.ResolveKind, dest string) (string, bool) {
	return derive.Href(b.opts.Resolver, false, kind, dest)
}

func (b *builder) rawHTML(s string) (html string, unsafe bool) {
	if b.opts.AllowRawHTML {
		return s, true
	}
	return derive.SanitizeHTML(s), false
}
