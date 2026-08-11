package parser

import (
	"testing"

	"github.com/sriannamalai/markdownviewer/document"
)

// spanOf returns the span of the i-th top-level child.
func spanOf(t *testing.T, src string, i int) document.Span {
	t.Helper()
	doc, err := Parse([]byte(src))
	if err != nil {
		t.Fatal(err)
	}
	kids := doc.Children()
	if i >= len(kids) {
		t.Fatalf("only %d children", len(kids))
	}
	c, ok := kids[i].(interface{ Span() document.Span })
	if !ok {
		t.Fatalf("child %d has no Span()", i)
	}
	return c.Span()
}

func TestLeafBlockSpans(t *testing.T) {
	src := "# Title\n\npara one\nline two\n\n```go\ncode\n```\n"
	// offsets: "# Title\n"=0..7(nl at 7), blank 8, "para one\n"=9..17,
	// "line two\n"=18..26, blank 27, "```go\n"=28, "code\n"=34, "```\n"=39
	h := spanOf(t, src, 0)
	if h.StartLine != 1 || h.EndLine != 1 || h.StartOffset != 0 {
		t.Fatalf("heading span %+v", h)
	}
	p := spanOf(t, src, 1)
	if p.StartLine != 3 || p.EndLine != 4 || p.StartOffset != 9 {
		t.Fatalf("para span %+v", p)
	}
	cb := spanOf(t, src, 2)
	// fenced code Lines() cover only the body; span start extends to the
	// start of the line containing the first body segment is NOT the fence
	// line — accept body-based span but the line must be the code body line.
	if cb.StartLine != 7 {
		t.Fatalf("code span %+v", cb)
	}
}

func TestSpanMarkerInclusiveStart(t *testing.T) {
	// Heading segment starts after "## "; span must extend left to the
	// start of the line so the markers are included.
	s := spanOf(t, "## Hello\n", 0)
	if s.StartOffset != 0 {
		t.Fatalf("marker not included: %+v", s)
	}
}

func TestSpanBOMBase(t *testing.T) {
	// BOM is stripped before parsing; offsets are relative to post-BOM src.
	s := spanOf(t, "\xEF\xBB\xBF# Hi\n", 0)
	if s.StartOffset != 0 || s.StartLine != 1 {
		t.Fatalf("BOM-relative span wrong: %+v", s)
	}
}

// inlineAt parses src and returns the i-th inline child of the first
// top-level block.
func inlineAt(t *testing.T, src string, i int) document.Node {
	t.Helper()
	doc, err := Parse([]byte(src))
	if err != nil {
		t.Fatal(err)
	}
	inl := doc.Children()[0].Children()
	if i >= len(inl) {
		t.Fatalf("only %d inline children", len(inl))
	}
	return inl[i]
}

// inlineSpanAt is inlineAt followed by the node's Span accessor.
func inlineSpanAt(t *testing.T, src string, i int) document.Span {
	t.Helper()
	n := inlineAt(t, src, i)
	sp, ok := n.(interface{ Span() document.Span })
	if !ok {
		t.Fatalf("inline %T has no Span()", n)
	}
	return sp.Span()
}

func wantSpan(t *testing.T, got document.Span, startLine, endLine, startOff, endOff int) {
	t.Helper()
	want := document.Span{StartLine: startLine, EndLine: endLine, StartOffset: startOff, EndOffset: endOff}
	if got != want {
		t.Fatalf("span = %+v, want %+v", got, want)
	}
}

func TestInlineTextSpans(t *testing.T) {
	// "a *b* c\n": Text "a " 0..2, Emphasis(Text "b" 3..4), Text " c" 5..7.
	wantSpan(t, inlineSpanAt(t, "a *b* c\n", 0), 1, 1, 0, 2)
	wantSpan(t, inlineSpanAt(t, "a *b* c\n", 2), 1, 1, 5, 7)
}

func TestEmphasisContentSpan(t *testing.T) {
	// Emphasis spans its CONTENT only — goldmark drops the * delimiters
	// from the segment list, so they are not recoverable honestly.
	em := inlineAt(t, "a *b* c\n", 1).(*document.Emphasis)
	wantSpan(t, em.Span(), 1, 1, 3, 4)
	// The inner Text carries the same content range.
	wantSpan(t, em.Children()[0].(*document.Text).Span(), 1, 1, 3, 4)
}

func TestStrongContentSpan(t *testing.T) {
	// "**bold word** end\n": content "bold word" at 2..11.
	st := inlineAt(t, "**bold word** end\n", 0).(*document.Strong)
	wantSpan(t, st.Span(), 1, 1, 2, 11)
}

func TestEmphasisMultiLineUnion(t *testing.T) {
	// "*two\nlines* x\n": content spans lines 1-2, 1..10 (softbreak inside).
	em := inlineAt(t, "*two\nlines* x\n", 0).(*document.Emphasis)
	wantSpan(t, em.Span(), 1, 2, 1, 10)
}

func TestLinkWithNestedEmphasisUnion(t *testing.T) {
	// "[a *b* c](u)\n": label content "a *b* c" spans 1..8 (delimiters and
	// destination excluded — content-only).
	l := inlineAt(t, "[a *b* c](u)\n", 0).(*document.Link)
	wantSpan(t, l.Span(), 1, 1, 1, 8)
	em := l.Children()[1].(*document.Emphasis)
	wantSpan(t, em.Span(), 1, 1, 4, 5)
}

func TestCodeSpanContentSpan(t *testing.T) {
	// "x `co de` y\n": code content "co de" at 3..8; backticks excluded.
	cs := inlineAt(t, "x `co de` y\n", 1).(*document.CodeSpan)
	wantSpan(t, cs.Span(), 1, 1, 3, 8)
}

func TestStrikethroughContentSpan(t *testing.T) {
	s := inlineAt(t, "~~gone~~\n", 0).(*document.Strikethrough)
	wantSpan(t, s.Span(), 1, 1, 2, 6)
}

func TestImageContentSpan(t *testing.T) {
	// "![alt text](i.png)\n": alt content at 2..10.
	img := inlineAt(t, "![alt text](i.png)\n", 0).(*document.Image)
	wantSpan(t, img.Span(), 1, 1, 2, 10)
}

func TestWikiLinkLabelSpan(t *testing.T) {
	// "[[Page]]\n": label segment 2..6 ([[ ]] excluded).
	wl := inlineAt(t, "[[Page]]\n", 0).(*document.WikiLink)
	wantSpan(t, wl.Span(), 1, 1, 2, 6)
	// "[[T|label]]\n": span covers the label part, 4..9.
	wl2 := inlineAt(t, "[[T|label]]\n", 0).(*document.WikiLink)
	wantSpan(t, wl2.Span(), 1, 1, 4, 9)
}

func TestMathInlineDelimiterInclusiveSpan(t *testing.T) {
	// Math is parsed by our own inline parser, which records the full
	// delimiter-inclusive range: "$x+y$" at 0..5.
	mi := inlineAt(t, "$x+y$ end\n", 0).(*document.MathInline)
	wantSpan(t, mi.Span(), 1, 1, 0, 5)
	// Display math inline in text: "a $$z$$ b\n" -> $$z$$ at 2..7.
	mi2 := inlineAt(t, "a $$z$$ b\n", 1).(*document.MathInline)
	wantSpan(t, mi2.Span(), 1, 1, 2, 7)
}

func TestHTMLInlineSpan(t *testing.T) {
	// "a <b>x</b> c\n": raw segments carry the full markup ranges.
	open := inlineAt(t, "a <b>x</b> c\n", 1).(*document.HTMLInline)
	wantSpan(t, open.Span(), 1, 1, 2, 5)
	cl := inlineAt(t, "a <b>x</b> c\n", 3).(*document.HTMLInline)
	wantSpan(t, cl.Span(), 1, 1, 6, 10)
}

func TestAutoLinkSpanIsZero(t *testing.T) {
	// goldmark's AutoLink node keeps its source segment in an unexported
	// field with no accessor, so no honest span exists; it stays zero.
	l := inlineAt(t, "<http://x.y>\n", 0).(*document.Link)
	if !l.Span().IsZero() {
		t.Fatalf("autolink span should be zero, got %+v", l.Span())
	}
}

func TestInlineSpansAfterFrontMatter(t *testing.T) {
	// Front-matter bytes stay in src (goldmark consumes the fence lines but
	// offsets are absolute), so inline spans must point at the post-fence
	// source lines with no extra adjustment.
	src := "---\ntitle: x\n---\n\nhello *em*\n"
	// Lines: "---\n"=0..4, "title: x\n"=4..13, "---\n"=13..17, "\n"=17..18,
	// "hello *em*\n"=18..29.
	wantSpan(t, inlineSpanAt(t, src, 0), 5, 5, 18, 24) // Text "hello "
	em := inlineAt(t, src, 1).(*document.Emphasis)
	wantSpan(t, em.Span(), 5, 5, 25, 27)
}

func TestInlineSpanJSONRoundTrip(t *testing.T) {
	doc, err := Parse([]byte("a *b* c\n"))
	if err != nil {
		t.Fatal(err)
	}
	data, err := document.MarshalJSON(doc)
	if err != nil {
		t.Fatal(err)
	}
	back, err := document.UnmarshalJSON(data)
	if err != nil {
		t.Fatal(err)
	}
	orig := doc.Children()[0].Children()[1].(*document.Emphasis)
	got := back.Children()[0].Children()[1].(*document.Emphasis)
	if got.Span() != orig.Span() || orig.Span().IsZero() {
		t.Fatalf("inline span lost in round trip: %+v vs %+v", got.Span(), orig.Span())
	}
}

func TestContainerSpansUnion(t *testing.T) {
	src := "- one\n- two\n  - nested\n"
	doc, _ := Parse([]byte(src))
	list := doc.Children()[0].(*document.List)
	ls := list.Span()
	if ls.StartLine != 1 || ls.EndLine != 3 {
		t.Fatalf("list span %+v", ls)
	}
	item2 := list.Children()[1].(*document.ListItem)
	is := item2.Span()
	if is.StartLine != 2 || is.EndLine != 3 {
		t.Fatalf("item span %+v", is)
	}
}

func TestBlockquoteAndAdmonitionSpans(t *testing.T) {
	bq := spanOf(t, "> quoted\n> more\n", 0)
	if bq.StartLine != 1 || bq.EndLine != 2 {
		t.Fatalf("blockquote span %+v", bq)
	}
	adm := spanOf(t, "> [!NOTE]\n> body\n", 0)
	if adm.StartLine == 0 || adm.EndLine < adm.StartLine {
		t.Fatalf("admonition span %+v", adm)
	}
}

func TestTableSpan(t *testing.T) {
	s := spanOf(t, "| a |\n|---|\n| b |\n", 0)
	if s.StartLine != 1 || s.EndLine != 3 {
		t.Fatalf("table span %+v", s)
	}
}

func TestThematicBreakSpan(t *testing.T) {
	// "***" rather than "---": with the default Config, FrontMatter is
	// enabled, and a "-"-delimited line at document line 1 is claimed by
	// the frontmatter block parser (go.abhg.dev/goldmark/frontmatter),
	// which swallows the entire input as an unterminated frontmatter
	// block when no closing delimiter follows — an unrelated, pre-existing
	// default-config interaction, not a thematic-break/span concern. "*"
	// is not a frontmatter delimiter, so it exercises the ThematicBreak
	// path without tripping over that.
	wantSpan(t, spanOf(t, "***\n", 0), 1, 1, 0, 4)
	// Indented break: span extends left to the start of the line,
	// marker-inclusive like every other leaf block.
	wantSpan(t, spanOf(t, "  ***  \n", 0), 1, 1, 0, 8)
	// "---" beyond line 1 is a plain thematic break even with FrontMatter
	// enabled; lines: "para\n"=0..5, "\n"=5..6, "---\n"=6..10.
	wantSpan(t, spanOf(t, "para\n\n---\n\nafter\n", 1), 3, 3, 6, 10)
}

func TestThematicBreakSpanAfterFrontMatter(t *testing.T) {
	// The span-recording thematic-break parser must not fire on the
	// front-matter fence lines (the frontmatter block parser runs at
	// priority 0, before it), and the break's offsets are absolute:
	// "---\n"=0..4, "a: 1\n"=4..9, "---\n"=9..13, "***\n"=13..17.
	wantSpan(t, spanOf(t, "---\na: 1\n---\n***\n", 0), 4, 4, 13, 17)
}
