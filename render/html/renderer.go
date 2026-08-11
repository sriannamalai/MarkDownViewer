package htmlrender

import (
	"bufio"
	"fmt"
	"io"
	"strings"

	"github.com/yuin/goldmark/util"

	"github.com/sriannamalai/markdownviewer/document"
	"github.com/sriannamalai/markdownviewer/resolve"
)

// Render writes doc to w as HTML per opts, either a full page (the default)
// or a body-only fragment (Options.Fragment).
func Render(w io.Writer, doc *document.Document, opts Options) error {
	if opts.Fragment {
		return renderFragment(w, doc, opts)
	}
	return renderPage(w, doc, opts)
}

func renderFragment(w io.Writer, doc *document.Document, opts Options) error {
	bw := bufio.NewWriter(w)
	r := &writer{w: bw, opts: opts}
	for _, c := range doc.Children() {
		if opts.SourceMap {
			if sp, ok := c.(interface{ Span() document.Span }); ok {
				if s := sp.Span(); !s.IsZero() {
					r.lineAttr = fmt.Sprintf(` data-md-line="%d"`, s.StartLine)
				}
			}
		}
		r.block(c, false)
	}
	r.footnotes(doc)
	if r.err != nil {
		return r.err
	}
	return bw.Flush()
}

type writer struct {
	w        *bufio.Writer
	opts     Options
	err      error
	lineAttr string
}

func (r *writer) raw(s string) {
	if r.err == nil {
		_, r.err = r.w.WriteString(s)
	}
}
func (r *writer) text(s string) { r.raw(esc(s)) }

// esc escapes the characters that are unsafe in both HTML text content and
// double-quoted attribute values: & < > ".
func esc(s string) string {
	if !strings.ContainsAny(s, "&<>\"") {
		return s
	}
	var b strings.Builder
	b.Grow(len(s))
	for _, r := range s {
		switch r {
		case '&':
			b.WriteString("&amp;")
		case '<':
			b.WriteString("&lt;")
		case '>':
			b.WriteString("&gt;")
		case '"':
			b.WriteString("&quot;")
		default:
			b.WriteRune(r)
		}
	}
	return b.String()
}

// href resolves dest to an escaped attribute value. The second return value
// reports whether the href/src attribute should be emitted at all: false
// means the destination was blocked by policy, which is a different outcome
// from an empty-but-allowed destination (e.g. "[foo]: <>", CommonMark spec
// example 200) — the former omits the attribute entirely, the latter emits
// `href=""`. Callers must not conflate the two by testing the string alone.
//
// A Resolver's ok=true result is trusted per its documented contract
// (Options.Resolver): the host controls resolution, so its URL is emitted
// as-is, bypassing the resolve.SafeURL scheme allowlist. Everything else —
// no Resolver installed, or the Resolver declined with ok=false — takes
// the default resolution path (resolve.DefaultResolution: wikilink targets
// get ".md" appended; other destinations pass through unchanged) and is
// filtered by resolve.SafeURL exactly as before. Both policies live in the
// renderer-agnostic resolve package so every renderer shares them.
func (r *writer) href(kind ResolveKind, dest string) (string, bool) {
	if r.opts.Resolver != nil {
		if u, ok := r.opts.Resolver(kind, dest); ok {
			return esc(u), true
		}
	}
	u := resolve.DefaultResolution(kind, dest)
	if !r.opts.Unsafe && !resolve.SafeURL(u) {
		return "", false
	}
	if kind != ResolveWikiLink {
		// Percent-encode the destination to match the CommonMark reference
		// HTML output. Wikilink targets are excluded: they're a filesystem
		// path resolved by the host, not a URI, and encoding would corrupt
		// e.g. spaces in page names (see TestWikiLinkDefaultResolution).
		// Backslash-escapes and entity references in ordinary link/image
		// destinations are already decoded by the parser (transform.go); for
		// autolinks — where CommonMark forbids that decoding — the raw text
		// flows through untouched, so this call only ever adds percent-
		// encoding, never a second decode pass.
		u = string(util.URLEscape([]byte(u), false))
	}
	return esc(u), true
}

func (r *writer) block(n document.Node, tight bool) {
	attr := r.lineAttr
	r.lineAttr = ""
	switch n := n.(type) {
	case *document.Heading:
		tag := fmt.Sprintf("h%d", n.Level)
		if r.opts.HeadingAnchors && n.AnchorID != "" {
			r.raw("<" + tag + attr + ` id="` + esc(n.AnchorID) + `">`)
		} else {
			r.raw("<" + tag + attr + ">")
		}
		r.inlines(n)
		r.raw("</" + tag + ">\n")
	case *document.Paragraph:
		if tight {
			r.inlines(n)
			return
		}
		r.raw("<p" + attr + ">")
		r.inlines(n)
		r.raw("</p>\n")
	case *document.BlockQuote:
		r.raw("<blockquote" + attr + ">\n")
		r.blocks(n, false)
		r.raw("</blockquote>\n")
	case *document.Admonition:
		variant := n.Variant
		if variant == "" {
			variant = "note"
		}
		// Title-case first, then escape: the parser constrains variants,
		// but hand-built documents and doc JSON via RenderDoc can carry
		// arbitrary bytes here, so both attribute and title positions must
		// be escaped.
		title := strings.ToUpper(variant[:1]) + variant[1:]
		r.raw(`<div` + attr + ` class="admonition admonition-` + esc(variant) + "\">\n")
		r.raw(`<p class="admonition-title">` + esc(title) + "</p>\n")
		r.blocks(n, false)
		r.raw("</div>\n")
	case *document.List:
		tag, attrs := "ul", ""
		if n.Ordered {
			tag = "ol"
			if n.Start != 1 {
				attrs = fmt.Sprintf(` start="%d"`, n.Start)
			}
		}
		cls := ""
		if listHasTaskItem(n) {
			cls = ` class="contains-task-list"`
		}
		r.raw("<" + tag + attr + cls + attrs + ">\n")
		for _, li := range n.Children() {
			if item, ok := li.(*document.ListItem); ok {
				r.listItem(item, n.Tight)
			} else {
				// A List should only ever contain ListItem children; a
				// hand-built document that violates this renders the
				// stray node through the normal block path rather than
				// panicking on the type assertion.
				r.block(li, false)
			}
		}
		r.raw("</" + tag + ">\n")
	case *document.CodeBlock:
		r.codeBlock(n, attr)
	case *document.Diagram:
		if r.opts.Mermaid && n.Engine == "mermaid" {
			r.raw(`<pre` + attr + ` class="mermaid">` + esc(n.Source) + "</pre>\n")
		} else {
			r.codeBlock(&document.CodeBlock{Language: n.Engine, Code: n.Source}, attr)
		}
	case *document.MathBlock:
		if r.opts.Math {
			r.raw(`<div` + attr + ` class="math math-display">` + esc(n.Source) + "</div>\n")
		} else {
			r.codeBlock(&document.CodeBlock{Language: "math", Code: n.Source}, attr)
		}
	case *document.Table:
		r.table(n, attr)
	case *document.ThematicBreak:
		r.raw("<hr" + attr + " />\n")
	case *document.HTMLBlock:
		r.rawHTML(n.HTML, true)
	case *document.DefinitionList:
		r.raw("<dl" + attr + ">\n")
		for _, c := range n.Children() {
			switch c := c.(type) {
			case *document.DefinitionTerm:
				r.raw("<dt>")
				r.inlines(c)
				r.raw("</dt>\n")
			case *document.DefinitionDesc:
				r.raw("<dd>\n")
				r.blocks(c, false)
				r.raw("</dd>\n")
			}
		}
		r.raw("</dl>\n")
	default:
		r.inline(n) // stray inline at block level
	}
}

func (r *writer) blocks(n document.Node, tight bool) {
	for _, c := range n.Children() {
		r.block(c, tight)
	}
}
func (r *writer) inlines(n document.Node) {
	for _, c := range n.Children() {
		r.inline(c)
	}
}

// listHasTaskItem reports whether any direct ListItem child of n is a task
// item, so the enclosing <ul>/<ol> can carry class="contains-task-list"
// (base.css uses it to zero the list's own indent, since task items supply
// their own via a negative checkbox margin).
func listHasTaskItem(n *document.List) bool {
	for _, c := range n.Children() {
		if li, ok := c.(*document.ListItem); ok && li.Task {
			return true
		}
	}
	return false
}

func (r *writer) listItem(li *document.ListItem, tight bool) {
	kids := li.Children()
	liClass := ""
	if li.Task {
		liClass = ` class="task-list-item"`
	}
	// A list item with no block content at all renders as the bare
	// "<li></li>", with none of the internal newlines either branch below
	// would otherwise add (CommonMark spec example 315).
	if len(kids) == 0 && !li.Task {
		r.raw("<li></li>\n")
		return
	}
	if tight {
		r.raw("<li" + liClass + ">")
		if li.Task {
			r.checkbox(li.Checked)
		}
		// In a tight list, every Paragraph child (not just the first — a
		// heading or nested list may precede a trailing paragraph, spec
		// example 300) is inlined without a <p> wrapper. Every other block
		// type renders normally and, like all of r.block's output, ends
		// with its own trailing "\n".
		//
		// A leading "\n" is needed before child i exactly when the
		// previously written output doesn't already end in one: that's the
		// case for i==0 when the first child isn't a paragraph (nothing
		// has been written since "<li>" yet), and for i>0 when the
		// previous child WAS an inlined paragraph (which, unlike a block,
		// leaves no trailing newline behind).
		prevInlinedParagraph := false
		for i, c := range kids {
			p, isParagraph := c.(*document.Paragraph)
			if i == 0 {
				if !isParagraph {
					r.raw("\n")
				}
			} else if prevInlinedParagraph {
				r.raw("\n")
			}
			if isParagraph {
				r.inlines(p)
				prevInlinedParagraph = true
				continue
			}
			r.block(c, false)
			prevInlinedParagraph = false
		}
		r.raw("</li>\n")
		return
	}
	// Loose item. A loose task item nests its checkbox inside the first
	// paragraph's <p> — <li><p><input.../> text</p>…</li>, matching
	// cmark-gfm — rather than as a sibling before it: a checkbox sibling
	// before the <p> pushes the paragraph's text onto its own line/block,
	// which reads as the checkbox floating above disconnected text instead
	// of leading it.
	r.raw("<li" + liClass + ">\n")
	if li.Task {
		if len(kids) > 0 {
			if p, ok := kids[0].(*document.Paragraph); ok {
				pAttr := r.lineAttr
				r.lineAttr = ""
				r.raw("<p" + pAttr + ">")
				r.checkbox(li.Checked)
				r.inlines(p)
				r.raw("</p>\n")
				for _, c := range kids[1:] {
					r.block(c, false)
				}
				r.raw("</li>\n")
				return
			}
		}
		// No paragraph to attach the checkbox to (an empty task item, or
		// one whose first block isn't a paragraph) — fall back to emitting
		// it before the item's blocks, same as the tight shape.
		r.checkbox(li.Checked)
	}
	r.blocks(li, false)
	r.raw("</li>\n")
}

func (r *writer) checkbox(checked bool) {
	if checked {
		r.raw(`<input type="checkbox" checked disabled /> `)
	} else {
		r.raw(`<input type="checkbox" disabled /> `)
	}
}

func (r *writer) codeBlock(n *document.CodeBlock, attr string) {
	if r.opts.CodeHeader {
		// Header row above the code: language label (display casing is
		// CSS's job) + copy button. Both emit paths below sit inside the
		// wrapper; data-md-line stays on the block element it is on
		// today — the wrapper carries no attributes.
		lang := n.Language
		if lang == "" {
			lang = "code"
		}
		r.raw(`<div class="md-code"><div class="md-code-header"><span class="md-code-lang">` +
			esc(lang) + `</span><button type="button" class="md-code-copy">Copy</button></div>`)
		defer r.raw("</div>\n")
	}
	if r.opts.Highlight {
		var b strings.Builder
		if highlight(&b, n.Code, n.Language) {
			// Chroma-highlighted output is unowned markup (its own <pre>/
			// <span> structure) — no data-md-line attribute is threaded in.
			r.raw(b.String())
			if !strings.HasSuffix(b.String(), "\n") {
				r.raw("\n")
			}
			return
		}
	}
	cls := ""
	if n.Language != "" {
		cls = ` class="language-` + esc(n.Language) + `"`
	}
	r.raw("<pre" + attr + "><code" + cls + ">")
	r.text(n.Code)
	r.raw("</code></pre>\n")
}

var alignAttr = map[document.Alignment]string{
	document.AlignLeft: ` align="left"`, document.AlignCenter: ` align="center"`,
	document.AlignRight: ` align="right"`,
}

func (r *writer) table(t *document.Table, attr string) {
	r.raw("<table" + attr + ">\n")
	rows := t.Children()
	for ri, rowNode := range rows {
		row, ok := rowNode.(*document.TableRow)
		if !ok {
			// A Table should only ever contain TableRow children; a
			// hand-built document that violates this drops the stray
			// node rather than panicking on the type assertion.
			continue
		}
		if row.Header {
			r.raw("<thead>\n")
		} else if ri == 1 {
			r.raw("<tbody>\n")
		}
		r.raw("<tr>\n")
		tag := "td"
		if row.Header {
			tag = "th"
		}
		for ci, cellNode := range row.Children() {
			cell, ok := cellNode.(*document.TableCell)
			if !ok {
				// Likewise for TableRow's children: skip anything that
				// isn't a TableCell instead of asserting.
				continue
			}
			attr := ""
			if ci < len(t.Alignments) {
				attr = alignAttr[t.Alignments[ci]]
			}
			r.raw("<" + tag + attr + ">")
			r.inlines(cell)
			r.raw("</" + tag + ">\n")
		}
		r.raw("</tr>\n")
		if row.Header {
			r.raw("</thead>\n")
		}
	}
	if len(rows) > 1 {
		r.raw("</tbody>\n")
	}
	r.raw("</table>\n")
}

func (r *writer) footnotes(doc *document.Document) {
	if len(doc.Footnotes) == 0 {
		return
	}
	r.raw("<section class=\"footnotes\">\n<ol>\n")
	for _, def := range doc.Footnotes {
		attr := ""
		if r.opts.SourceMap {
			if s := def.Span(); !s.IsZero() {
				attr = fmt.Sprintf(` data-md-line="%d"`, s.StartLine)
			}
		}
		r.raw(fmt.Sprintf(`<li id="fn:%d"%s>`+"\n", def.Index, attr))
		r.blocks(def, false)
		r.raw(fmt.Sprintf(`<a href="#fnref:%d" class="footnote-backref">↩</a>`+"\n</li>\n", def.Index))
	}
	r.raw("</ol>\n</section>\n")
}

func (r *writer) inline(n document.Node) {
	switch n := n.(type) {
	case *document.Text:
		r.text(n.Value)
	case *document.SoftBreak:
		r.raw("\n")
	case *document.HardBreak:
		r.raw("<br />\n")
	case *document.Emphasis:
		r.raw("<em>")
		r.inlines(n)
		r.raw("</em>")
	case *document.Strong:
		r.raw("<strong>")
		r.inlines(n)
		r.raw("</strong>")
	case *document.Strikethrough:
		r.raw("<del>")
		r.inlines(n)
		r.raw("</del>")
	case *document.CodeSpan:
		r.raw("<code>")
		r.text(n.Value)
		r.raw("</code>")
	case *document.Link:
		if u, ok := r.href(ResolveLink, n.Destination); ok {
			t := ""
			if n.Title != "" {
				t = ` title="` + esc(n.Title) + `"`
			}
			r.raw(`<a href="` + u + `"` + t + ">")
		} else {
			r.raw("<a>")
		}
		r.inlines(n)
		r.raw("</a>")
	case *document.Image:
		u, _ := r.href(ResolveImage, n.Destination) // blocked destinations still get a (empty) src, like a real <img>
		t := ""
		if n.Title != "" {
			t = ` title="` + esc(n.Title) + `"`
		}
		r.raw(`<img src="` + u + `" alt="` + esc(n.Alt) + `"` + t + " />")
	case *document.WikiLink:
		if u, ok := r.href(ResolveWikiLink, n.Target); ok {
			r.raw(`<a href="` + u + `" class="wikilink">`)
		} else {
			r.raw(`<a class="wikilink">`)
		}
		r.inlines(n)
		r.raw("</a>")
	case *document.MathInline:
		if r.opts.Math {
			cls := "math math-inline"
			if n.Display {
				cls = "math math-display"
			}
			r.raw(`<span class="` + cls + `">` + esc(n.Source) + "</span>")
		} else {
			r.raw("<code>")
			r.text(n.Source)
			r.raw("</code>")
		}
	case *document.HTMLInline:
		r.rawHTML(n.HTML, false)
	case *document.FootnoteRef:
		r.raw(fmt.Sprintf(`<sup id="fnref:%d"><a href="#fn:%d">%d</a></sup>`, n.Index, n.Index, n.Index))
	}
}

// rawHTML sanitizes raw markdown HTML with bluemonday's UGC policy by
// default, stripping scripts, event handlers, and other XSS vectors.
// Unsafe mode passes it through verbatim.
func (r *writer) rawHTML(s string, block bool) {
	out := s
	if !r.opts.Unsafe {
		out = sanitizePolicy.Sanitize(s)
	}
	r.raw(out)
	if block && !strings.HasSuffix(out, "\n") {
		r.raw("\n")
	}
}
