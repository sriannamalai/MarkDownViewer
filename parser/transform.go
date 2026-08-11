package parser

import (
	"sort"
	"strconv"
	"strings"

	emast "github.com/yuin/goldmark-emoji/ast"
	"github.com/yuin/goldmark/ast"
	east "github.com/yuin/goldmark/extension/ast"
	gparser "github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/util"
	"go.abhg.dev/goldmark/frontmatter"
	"go.abhg.dev/goldmark/wikilink"

	"github.com/sriannamalai/markdownviewer/document"
)

// unescapeText decodes backslash-escaped ASCII punctuation and HTML entity /
// numeric character references, e.g. `\*` -> `*`, `&amp;` -> `&`,
// `&#35;` -> `#`. It applies to plain inline text, link/image titles and
// destinations, and code-fence info strings — everywhere CommonMark treats
// backslash escapes and character references as live syntax.
//
// It deliberately does NOT apply inside code spans, code blocks, raw HTML,
// or autolink destinations/labels: the spec keeps those literal, and
// transform.go only calls it from the sites where decoding is required.
//
// This mirrors the decoding goldmark's own HTML Writer performs at render
// time (see (goldmark/renderer/html).Writer.Write), and must run as a
// single left-to-right pass rather than two independent passes (unescape,
// then resolve entities): a backslash-escaped "&" (`\&ouml;`) must produce
// a literal "&" that is NOT then reinterpreted as the start of an entity
// reference. CommonMark spec example 14 covers exactly this case. A
// two-pass unescape-then-resolve implementation gets it wrong because the
// escaped "&" is indistinguishable from a literal one by the time the
// entity pass runs.
func unescapeText(source []byte) string {
	n := len(source)
	var b strings.Builder
	b.Grow(n)
	for i := 0; i < n; {
		c := source[i]
		if c == '\\' && i+1 < n && util.IsPunct(source[i+1]) {
			b.WriteByte(source[i+1])
			i += 2
			continue
		}
		if c == '&' {
			if resolved, end, ok := resolveCharRef(source, i, n); ok {
				b.Write(resolved)
				i = end
				continue
			}
		}
		b.WriteByte(c)
		i++
	}
	return b.String()
}

// resolveCharRef attempts to parse an HTML character reference (numeric,
// e.g. "&#35;"/"&#x23;", or named, e.g. "&amp;") starting at source[start]
// (source[start] == '&'). On success it returns the decoded UTF-8 bytes and
// the index just past the reference's trailing ';'.
func resolveCharRef(source []byte, start, limit int) ([]byte, int, bool) {
	next := start + 1
	if next >= limit {
		return nil, 0, false
	}
	if source[next] == '#' {
		nnext := next + 1
		if nnext >= limit {
			return nil, 0, false
		}
		if c := source[nnext]; c == 'x' || c == 'X' {
			numStart := nnext + 1
			end, ok := util.ReadWhile(source, [2]int{numStart, limit}, util.IsHexDecimal)
			// CommonMark's numeric character reference grammar caps hex
			// references at 6 hex digits (mirroring the 7-digit cap on the
			// decimal branch below): a 7th digit means this isn't a valid
			// reference at all, so it — and the "&#x" that looked like it
			// was introducing one — stays literal.
			if !ok || end-numStart >= 7 || end >= limit || source[end] != ';' || end == numStart {
				return nil, 0, false
			}
			v, _ := strconv.ParseUint(string(source[numStart:end]), 16, 32)
			return []byte(string(util.ToValidRune(rune(v)))), end + 1, true
		}
		if c := source[nnext]; c >= '0' && c <= '9' {
			numStart := nnext
			end, ok := util.ReadWhile(source, [2]int{numStart, limit}, util.IsNumeric)
			if !ok || end-numStart >= 8 || end >= limit || source[end] != ';' {
				return nil, 0, false
			}
			v, _ := strconv.ParseUint(string(source[numStart:end]), 10, 32)
			return []byte(string(util.ToValidRune(rune(v)))), end + 1, true
		}
		return nil, 0, false
	}
	end, ok := util.ReadWhile(source, [2]int{next, limit}, util.IsAlphaNumeric)
	if !ok || end >= limit || source[end] != ';' {
		return nil, 0, false
	}
	entity, ok := util.LookUpHTML5EntityByName(string(source[next:end]))
	if !ok {
		return nil, 0, false
	}
	return entity.Characters, end + 1, true
}

type transformer struct {
	src       []byte
	cfg       Config
	lines     lineIndex
	slugs     map[string]int
	item      *document.ListItem // innermost list item, for task checkboxes
	footnotes []*document.FootnoteDef
}

// lineIndex holds the byte offset of every line start; lineIndex[0] == 0.
type lineIndex []int

func newLineIndex(src []byte) lineIndex {
	idx := lineIndex{0}
	for i, b := range src {
		if b == '\n' {
			idx = append(idx, i+1)
		}
	}
	return idx
}

// lineFor returns the 1-based line number containing byte offset off.
func (li lineIndex) lineFor(off int) int {
	return sort.Search(len(li), func(i int) bool { return li[i] > off })
}

// lineStart returns the byte offset of the start of the line containing off.
func (li lineIndex) lineStart(off int) int {
	return li[li.lineFor(off)-1]
}

// leafSpan computes a marker-inclusive span from a block node's Lines().
func (t *transformer) leafSpan(n ast.Node) document.Span {
	l := n.Lines()
	if l.Len() == 0 {
		return document.Span{}
	}
	first, last := l.At(0), l.At(l.Len()-1)
	start := t.lines.lineStart(first.Start)
	end := last.Stop
	endLine := t.lines.lineFor(end - 1)
	if end <= start {
		endLine = t.lines.lineFor(start)
	}
	return document.Span{
		StartLine: t.lines.lineFor(start), EndLine: endLine,
		StartOffset: start, EndOffset: end,
	}
}

// segSpan converts a raw source byte range [start, stop) into a Span.
// Unlike leafSpan it does NOT extend left to the start of the line: inline
// content starts mid-line. An empty or inverted range yields the zero Span.
func (t *transformer) segSpan(start, stop int) document.Span {
	if stop <= start {
		return document.Span{}
	}
	return document.Span{
		StartLine: t.lines.lineFor(start), EndLine: t.lines.lineFor(stop - 1),
		StartOffset: start, EndOffset: stop,
	}
}

// astTextUnionSpan returns the span covering n's direct *ast.Text children's
// segments — the content range of inline containers whose document
// counterpart keeps no children of its own (code spans).
func (t *transformer) astTextUnionSpan(n ast.Node) document.Span {
	start, stop := -1, -1
	for c := n.FirstChild(); c != nil; c = c.NextSibling() {
		txt, ok := c.(*ast.Text)
		if !ok || txt.Segment.Len() == 0 {
			continue
		}
		if start < 0 || txt.Segment.Start < start {
			start = txt.Segment.Start
		}
		if txt.Segment.Stop > stop {
			stop = txt.Segment.Stop
		}
	}
	if start < 0 {
		return document.Span{}
	}
	return t.segSpan(start, stop)
}

// setSpan records s on n if n supports it. All document node types embed
// Container and thus satisfy this, but n is typed as the document.Node
// interface at call sites, which does not itself expose SetSpan.
func setSpan(n document.Node, s document.Span) {
	if sp, ok := n.(interface{ SetSpan(document.Span) }); ok {
		sp.SetSpan(s)
	}
}

// unionSpan widens a to also cover b; a zero operand yields the other.
func unionSpan(a, b document.Span) document.Span {
	switch {
	case a.IsZero():
		return b
	case b.IsZero():
		return a
	}
	if b.StartOffset < a.StartOffset {
		a.StartOffset, a.StartLine = b.StartOffset, b.StartLine
	}
	if b.EndOffset > a.EndOffset {
		a.EndOffset, a.EndLine = b.EndOffset, b.EndLine
	}
	return a
}

// childUnionSpan returns the union of the direct children's non-zero spans.
func childUnionSpan(n document.Node) document.Span {
	var u document.Span
	for _, c := range n.Children() {
		if sp, ok := c.(interface{ Span() document.Span }); ok {
			u = unionSpan(u, sp.Span())
		}
	}
	return u
}

func (t *transformer) document(root ast.Node, ctx gparser.Context) *document.Document {
	doc := &document.Document{}
	t.appendChildren(doc, root)
	doc.Footnotes = t.footnotes
	if d := frontmatter.Get(ctx); d != nil {
		var m map[string]any
		if err := d.Decode(&m); err == nil {
			doc.Meta = m
		}
	}
	return doc
}

func (t *transformer) appendChildren(parent document.Node, n ast.Node) {
	for c := n.FirstChild(); c != nil; c = c.NextSibling() {
		if out := t.convert(c); out != nil {
			// The Linkify inline parser triggers on plain spaces and, when
			// no URL/email matches, leaves the surrounding prose split
			// across sibling ast.Text nodes instead of one merged run.
			// Coalesce adjacent document.Text nodes so output stays
			// identical to the non-Linkify tree. Scoped to ast.Text inputs
			// only, so it doesn't also swallow synthesized Text nodes from
			// other node kinds (e.g. emoji shortcodes) that must stay
			// distinct siblings.
			if _, isASTText := c.(*ast.Text); isASTText {
				if txt, ok := out.(*document.Text); ok {
					if kids := parent.Children(); len(kids) > 0 {
						if last, ok := kids[len(kids)-1].(*document.Text); ok {
							last.Value += txt.Value
							last.SetSpan(unionSpan(last.Span(), txt.Span()))
							out = nil
						}
					}
				}
			}
			if out != nil {
				parent.AppendChild(out)
			}
		}
		// ast.Text carries trailing break flags; emit them as siblings.
		if txt, ok := c.(*ast.Text); ok {
			if txt.HardLineBreak() {
				parent.AppendChild(&document.HardBreak{})
			} else if txt.SoftLineBreak() {
				parent.AppendChild(&document.SoftBreak{})
			}
		}
	}
}

func (t *transformer) convert(n ast.Node) document.Node {
	switch n := n.(type) {
	case *ast.Heading:
		h := &document.Heading{Level: n.Level}
		t.appendChildren(h, n)
		h.AnchorID = t.slug(document.PlainText(h))
		h.SetSpan(t.leafSpan(n))
		return h
	case *ast.Paragraph:
		if t.cfg.Math {
			if mb := mathBlockFromLines(n, t.src); mb != nil {
				mb.SetSpan(t.leafSpan(n))
				return mb
			}
		}
		p := &document.Paragraph{}
		t.appendChildren(p, n)
		p.SetSpan(t.leafSpan(n))
		if t.cfg.Math {
			if kids := p.Children(); len(kids) == 1 {
				if mi, ok := kids[0].(*document.MathInline); ok && mi.Display {
					mb := &document.MathBlock{Source: mi.Source}
					mb.SetSpan(p.Span())
					return mb
				}
			}
		}
		return p
	case *ast.TextBlock: // tight-list item content; List.Tight drives <p> omission
		p := &document.Paragraph{}
		t.appendChildren(p, n)
		p.SetSpan(t.leafSpan(n))
		return p
	case *ast.Blockquote:
		bq := &document.BlockQuote{}
		t.appendChildren(bq, n)
		bq.SetSpan(childUnionSpan(bq))
		if t.cfg.Admonitions {
			if adm := promoteAdmonition(bq); adm != nil {
				adm.SetSpan(bq.Span())
				return adm
			}
		}
		return bq
	case *ast.List:
		l := &document.List{Ordered: n.IsOrdered(), Start: 1, Tight: n.IsTight}
		if n.IsOrdered() {
			l.Start = n.Start
		}
		t.appendChildren(l, n)
		l.SetSpan(childUnionSpan(l))
		return l
	case *ast.ListItem:
		li := &document.ListItem{}
		prev := t.item
		t.item = li
		t.appendChildren(li, n)
		t.item = prev
		li.SetSpan(childUnionSpan(li))
		return li
	case *ast.FencedCodeBlock:
		lang := unescapeText(n.Language(t.src))
		out := t.codeOrSpecial(lang, blockLines(n, t.src))
		setSpan(out, t.leafSpan(n))
		return out
	case *ast.CodeBlock:
		cb := &document.CodeBlock{Code: blockLines(n, t.src)}
		cb.SetSpan(t.leafSpan(n))
		return cb
	case *ast.ThematicBreak:
		// goldmark's stock parser leaves Lines() empty; ours (tbreak.go)
		// records the break's line segment there, so leafSpan works.
		tb := &document.ThematicBreak{}
		tb.SetSpan(t.leafSpan(n))
		return tb
	case *ast.HTMLBlock:
		html := blockLines(n, t.src)
		span := t.leafSpan(n)
		if n.HasClosure() {
			html += string(n.ClosureLine.Value(t.src))
			span.EndOffset = n.ClosureLine.Stop
			span.EndLine = t.lines.lineFor(span.EndOffset - 1)
		}
		hb := &document.HTMLBlock{HTML: html}
		hb.SetSpan(span)
		return hb
	case *ast.Text:
		txt := &document.Text{Value: unescapeText(n.Segment.Value(t.src))}
		txt.SetSpan(t.segSpan(n.Segment.Start, n.Segment.Stop))
		return txt
	case *ast.String:
		// Synthesized content (e.g. link-reference labels) with no source
		// segment: span stays zero.
		return &document.Text{Value: string(n.Value)}
	case *ast.Emphasis:
		var out document.Node
		if n.Level >= 2 {
			out = &document.Strong{}
		} else {
			out = &document.Emphasis{}
		}
		t.appendChildren(out, n)
		// Content-only span: goldmark drops the */_ delimiter runs from the
		// segment list, so the union of the children is all that exists.
		setSpan(out, childUnionSpan(out))
		return out
	case *ast.CodeSpan:
		cs := &document.CodeSpan{Value: spanText(n, t.src)}
		// Content-only span (backticks excluded): the document node keeps
		// no children, so union the ast children's segments directly.
		cs.SetSpan(t.astTextUnionSpan(n))
		return cs
	case *ast.Link:
		// Destination/Title are decoded here (backslash escapes, entity and
		// numeric character references); see unescapeText. Percent-encoding
		// for the URL is deferred to the HTML renderer (render/html), which
		// is the only layer that needs a URI-safe string — other consumers
		// of the document model want the human-readable destination.
		l := &document.Link{Destination: unescapeText(n.Destination), Title: unescapeText(n.Title)}
		t.appendChildren(l, n)
		// Content-only span: covers the link text, not the []()/ reference
		// delimiters or the destination (goldmark keeps no segments for
		// them).
		l.SetSpan(childUnionSpan(l))
		return l
	case *ast.AutoLink:
		// Deliberately not passed through unescapeText: CommonMark does not
		// recognize backslash escapes or entity references inside autolinks
		// (spec.commonmark.org/0.31.2/#autolinks), so the destination and
		// visible label stay exactly as written between < and >.
		url := string(n.URL(t.src))
		if n.AutoLinkType == ast.AutoLinkEmail && !strings.HasPrefix(url, "mailto:") {
			url = "mailto:" + url
		}
		// No span: goldmark's AutoLink keeps its source segment in an
		// unexported field with no accessor, so no honest position exists
		// for either the link or its synthesized label Text.
		l := &document.Link{Destination: url}
		l.AppendChild(&document.Text{Value: string(n.Label(t.src))})
		return l
	case *ast.Image:
		img := &document.Image{Destination: unescapeText(n.Destination), Title: unescapeText(n.Title)}
		t.appendChildren(img, n)
		img.Alt = document.PlainText(img)
		// Content-only span: covers the alt text, like Link above.
		img.SetSpan(childUnionSpan(img))
		return img
	case *ast.RawHTML:
		var b strings.Builder
		for i := 0; i < n.Segments.Len(); i++ {
			seg := n.Segments.At(i)
			b.Write(seg.Value(t.src))
		}
		hi := &document.HTMLInline{HTML: b.String()}
		// Raw HTML's segments cover the markup itself, so the span is the
		// full source range — no delimiter caveat here.
		if n.Segments.Len() > 0 {
			first, last := n.Segments.At(0), n.Segments.At(n.Segments.Len()-1)
			hi.SetSpan(t.segSpan(first.Start, last.Stop))
		}
		return hi
	case *east.Table:
		tbl := &document.Table{}
		for _, a := range n.Alignments {
			tbl.Alignments = append(tbl.Alignments, map[east.Alignment]document.Alignment{
				east.AlignNone: document.AlignNone, east.AlignLeft: document.AlignLeft,
				east.AlignCenter: document.AlignCenter, east.AlignRight: document.AlignRight,
			}[a])
		}
		t.appendChildren(tbl, n)
		tbl.SetSpan(childUnionSpan(tbl))
		return tbl
	case *east.TableHeader:
		row := &document.TableRow{Header: true}
		t.appendChildren(row, n)
		row.SetSpan(childUnionSpan(row))
		return row
	case *east.TableRow:
		row := &document.TableRow{}
		t.appendChildren(row, n)
		row.SetSpan(childUnionSpan(row))
		return row
	case *east.TableCell:
		cell := &document.TableCell{}
		t.appendChildren(cell, n)
		// Unlike Table and TableRow/TableHeader, the goldmark TableCell ast
		// node carries its own Lines() segment (see extension/table.go
		// parseRow, which calls node.Lines().Append(seg)), so leafSpan
		// gives the full per-cell span — wider than a union of the inline
		// children (it includes surrounding padding and any unspanned
		// inlines); Table/TableRow then pick it up via childUnionSpan.
		cell.SetSpan(t.leafSpan(n))
		return cell
	case *east.Strikethrough:
		s := &document.Strikethrough{}
		t.appendChildren(s, n)
		// Content-only span: the ~~ delimiters leave no segments.
		s.SetSpan(childUnionSpan(s))
		return s
	case *east.TaskCheckBox:
		if t.item != nil {
			t.item.Task = true
			t.item.Checked = n.IsChecked
		}
		return nil
	case *east.FootnoteLink:
		return &document.FootnoteRef{Index: n.Index}
	case *east.FootnoteBacklink:
		return nil // renderer generates backlinks
	case *east.FootnoteList:
		for c := n.FirstChild(); c != nil; c = c.NextSibling() {
			if def, ok := t.convert(c).(*document.FootnoteDef); ok {
				t.footnotes = append(t.footnotes, def)
			}
		}
		return nil
	case *east.Footnote:
		def := &document.FootnoteDef{Index: n.Index}
		t.appendChildren(def, n)
		def.SetSpan(childUnionSpan(def))
		return def
	case *emast.Emoji:
		return &document.Text{Value: string(n.Value.Unicode)}
	case *east.DefinitionList:
		dl := &document.DefinitionList{}
		t.appendChildren(dl, n)
		dl.SetSpan(childUnionSpan(dl))
		return dl
	case *east.DefinitionTerm:
		dt := &document.DefinitionTerm{}
		t.appendChildren(dt, n)
		// Like TableCell, DefinitionTerm's own ast node carries its Lines()
		// segment directly (extension/definition_list.go: term.Lines().
		// Append(segment)), which covers the whole term line rather than
		// just the inline children's union, so leafSpan is the real source.
		dt.SetSpan(t.leafSpan(n))
		return dt
	case *east.DefinitionDescription:
		dd := &document.DefinitionDesc{}
		t.appendChildren(dd, n)
		dd.SetSpan(childUnionSpan(dd))
		return dd
	case *wikilink.Node:
		wl := &document.WikiLink{Target: string(n.Target)}
		t.appendChildren(wl, n)
		// Content-only span: covers the visible label segment (after any
		// "|"), not the [[ ]] delimiters or the target half.
		wl.SetSpan(childUnionSpan(wl))
		return wl
	case *mathNode:
		mi := &document.MathInline{Source: string(n.Source), Display: n.Display}
		// Full delimiter-inclusive span ($x$ / $$x$$): our own inline
		// parser records the segment at parse time (math.go).
		mi.SetSpan(t.segSpan(n.Segment.Start, n.Segment.Stop))
		return mi
	default:
		return nil // unknown/unwired node kinds are dropped
	}
}

var admonitionVariants = map[string]bool{
	"NOTE": true, "TIP": true, "IMPORTANT": true, "WARNING": true, "CAUTION": true,
}

// promoteAdmonition returns an Admonition if bq's first paragraph starts
// with a GitHub-style [!VARIANT] marker; the marker (and a following
// SoftBreak) is stripped. Returns nil when bq is a plain quote.
func promoteAdmonition(bq *document.BlockQuote) *document.Admonition {
	kids := bq.Children()
	if len(kids) == 0 {
		return nil
	}
	p, ok := kids[0].(*document.Paragraph)
	if !ok {
		return nil
	}
	inl := p.Children()
	if len(inl) == 0 {
		return nil
	}
	txt, ok := inl[0].(*document.Text)
	if !ok || len(txt.Value) < 4 || !strings.HasPrefix(txt.Value, "[!") || !strings.HasSuffix(txt.Value, "]") {
		return nil
	}
	name := txt.Value[2 : len(txt.Value)-1]
	if !admonitionVariants[name] {
		return nil
	}
	adm := &document.Admonition{Variant: strings.ToLower(name)}
	rest := inl[1:]
	if len(rest) > 0 {
		if _, isBreak := rest[0].(*document.SoftBreak); isBreak {
			rest = rest[1:]
		}
	}
	if len(rest) > 0 {
		np := &document.Paragraph{}
		for _, c := range rest {
			np.AppendChild(c)
		}
		// The synthesized paragraph has no goldmark lines of its own; the
		// inline children carry spans now, so union them (this excludes
		// the stripped [!VARIANT] marker, which is the point).
		np.SetSpan(childUnionSpan(np))
		adm.AppendChild(np)
	}
	for _, c := range kids[1:] {
		adm.AppendChild(c)
	}
	return adm
}

// codeOrSpecial maps special fence languages to dedicated nodes.
func (t *transformer) codeOrSpecial(lang, code string) document.Node {
	switch lang {
	case "mermaid":
		return &document.Diagram{Engine: "mermaid", Source: code}
	case "math":
		return &document.MathBlock{Source: code}
	}
	return &document.CodeBlock{Language: lang, Code: code}
}

// mathBlockFromLines promotes a paragraph shaped like
//
//	$$
//	...
//	$$
//
// into a MathBlock.
func mathBlockFromLines(n *ast.Paragraph, src []byte) *document.MathBlock {
	l := n.Lines()
	if l.Len() < 3 {
		return nil
	}
	firstSeg, lastSeg := l.At(0), l.At(l.Len()-1)
	first := strings.TrimSpace(string(firstSeg.Value(src)))
	last := strings.TrimSpace(string(lastSeg.Value(src)))
	if first != "$$" || last != "$$" {
		return nil
	}
	var b strings.Builder
	for i := 1; i < l.Len()-1; i++ {
		seg := l.At(i)
		b.Write(seg.Value(src))
	}
	return &document.MathBlock{Source: b.String()}
}

func blockLines(n ast.Node, src []byte) string {
	var b strings.Builder
	l := n.Lines()
	for i := 0; i < l.Len(); i++ {
		seg := l.At(i)
		b.Write(seg.Value(src))
	}
	return b.String()
}

// spanText concatenates a code span's raw child segments. Backslash escapes
// and entity references are deliberately left alone — code spans are the
// one CommonMark inline construct where both stay literal — but per the
// spec, "line endings are converted to spaces" within a code span (the
// leading/trailing single-space trim goldmark's own code-span parser
// already applies directly to the segments, so only the newline-to-space
// conversion is left to do here).
func spanText(n ast.Node, src []byte) string {
	var b strings.Builder
	for c := n.FirstChild(); c != nil; c = c.NextSibling() {
		if txt, ok := c.(*ast.Text); ok {
			b.Write(txt.Segment.Value(src))
		}
	}
	s := b.String()
	s = strings.ReplaceAll(s, "\r\n", "\n")
	s = strings.ReplaceAll(s, "\n", " ")
	return s
}
