package htmlrender

import (
	"bufio"
	"fmt"
	"io"
	"strings"

	"github.com/sriannamalai/markdownviewer/document"
)

func Render(w io.Writer, doc *document.Document, opts Options) error {
	// Full-page mode arrives in Task 12; render fragment for now.
	return renderFragment(w, doc, opts)
}

func renderFragment(w io.Writer, doc *document.Document, opts Options) error {
	bw := bufio.NewWriter(w)
	r := &writer{w: bw, opts: opts}
	for _, c := range doc.Children() {
		r.block(c, false)
	}
	r.footnotes(doc)
	if r.err != nil {
		return r.err
	}
	return bw.Flush()
}

type writer struct {
	w    *bufio.Writer
	opts Options
	err  error
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

// href returns an escaped attribute value ("" if blocked).
//
// A Resolver's ok=true result is trusted per its documented contract
// (Options.Resolver): the host controls resolution, so its URL is emitted
// as-is, bypassing the safeURL scheme allowlist. Everything else — no
// Resolver installed, or the Resolver declined with ok=false — takes the
// default resolution path (wikilink targets get ".md" appended; other
// destinations pass through unchanged) and is filtered by safeURL exactly
// as before.
func (r *writer) href(kind ResolveKind, dest string) string {
	if r.opts.Resolver != nil {
		if u, ok := r.opts.Resolver(kind, dest); ok {
			return esc(u)
		}
	}
	u := dest
	if kind == ResolveWikiLink {
		u = dest + ".md"
	}
	if !r.opts.Unsafe && !safeURL(u) {
		return ""
	}
	return esc(u)
}

func (r *writer) block(n document.Node, tight bool) {
	switch n := n.(type) {
	case *document.Heading:
		tag := fmt.Sprintf("h%d", n.Level)
		if r.opts.HeadingAnchors && n.AnchorID != "" {
			r.raw("<" + tag + ` id="` + esc(n.AnchorID) + `">`)
		} else {
			r.raw("<" + tag + ">")
		}
		r.inlines(n)
		r.raw("</" + tag + ">\n")
	case *document.Paragraph:
		if tight {
			r.inlines(n)
			return
		}
		r.raw("<p>")
		r.inlines(n)
		r.raw("</p>\n")
	case *document.BlockQuote:
		r.raw("<blockquote>\n")
		r.blocks(n, false)
		r.raw("</blockquote>\n")
	case *document.Admonition:
		title := strings.ToUpper(n.Variant[:1]) + n.Variant[1:]
		r.raw(`<div class="admonition admonition-` + n.Variant + "\">\n")
		r.raw(`<p class="admonition-title">` + title + "</p>\n")
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
		r.raw("<" + tag + attrs + ">\n")
		for _, li := range n.Children() {
			r.listItem(li.(*document.ListItem), n.Tight)
		}
		r.raw("</" + tag + ">\n")
	case *document.CodeBlock:
		r.codeBlock(n)
	case *document.Diagram:
		if r.opts.Mermaid && n.Engine == "mermaid" {
			r.raw(`<pre class="mermaid">` + esc(n.Source) + "</pre>\n")
		} else {
			r.codeBlock(&document.CodeBlock{Language: n.Engine, Code: n.Source})
		}
	case *document.MathBlock:
		if r.opts.Math {
			r.raw(`<div class="math math-display">` + esc(n.Source) + "</div>\n")
		} else {
			r.codeBlock(&document.CodeBlock{Language: "math", Code: n.Source})
		}
	case *document.Table:
		r.table(n)
	case *document.ThematicBreak:
		r.raw("<hr>\n")
	case *document.HTMLBlock:
		r.rawHTML(n.HTML, true)
	case *document.DefinitionList:
		r.raw("<dl>\n")
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

func (r *writer) listItem(li *document.ListItem, tight bool) {
	kids := li.Children()
	if tight {
		r.raw("<li>")
		if li.Task {
			r.checkbox(li.Checked)
		}
		for i, c := range kids {
			if p, ok := c.(*document.Paragraph); ok && i == 0 {
				r.inlines(p)
				continue
			}
			if i == 0 {
				r.raw("\n")
			}
			r.block(c, false)
		}
		r.raw("</li>\n")
		return
	}
	r.raw("<li>\n")
	if li.Task {
		r.checkbox(li.Checked)
	}
	r.blocks(li, false)
	r.raw("</li>\n")
}

func (r *writer) checkbox(checked bool) {
	if checked {
		r.raw(`<input type="checkbox" checked disabled> `)
	} else {
		r.raw(`<input type="checkbox" disabled> `)
	}
}

func (r *writer) codeBlock(n *document.CodeBlock) {
	if r.opts.Highlight {
		var b strings.Builder
		if highlight(&b, n.Code, n.Language) {
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
	r.raw("<pre><code" + cls + ">")
	r.text(n.Code)
	r.raw("</code></pre>\n")
}

var alignAttr = map[document.Alignment]string{
	document.AlignLeft: ` align="left"`, document.AlignCenter: ` align="center"`,
	document.AlignRight: ` align="right"`,
}

func (r *writer) table(t *document.Table) {
	r.raw("<table>\n")
	rows := t.Children()
	for ri, rowNode := range rows {
		row := rowNode.(*document.TableRow)
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
		for ci, cell := range row.Children() {
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
		r.raw(fmt.Sprintf(`<li id="fn:%d">`+"\n", def.Index))
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
		r.raw("<br>\n")
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
		if u := r.href(ResolveLink, n.Destination); u != "" {
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
		u := r.href(ResolveImage, n.Destination)
		t := ""
		if n.Title != "" {
			t = ` title="` + esc(n.Title) + `"`
		}
		r.raw(`<img src="` + u + `" alt="` + esc(n.Alt) + `"` + t + ">")
	case *document.WikiLink:
		if u := r.href(ResolveWikiLink, n.Target); u != "" {
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
