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

func TestInlineNodesHaveZeroSpan(t *testing.T) {
	doc, _ := Parse([]byte("a *b* c\n"))
	para := doc.Children()[0].(*document.Paragraph)
	for _, inl := range para.Children() {
		if c, ok := inl.(interface{ Span() document.Span }); ok && !c.Span().IsZero() {
			t.Fatalf("inline %T has non-zero span", inl)
		}
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

func TestThematicBreakZeroSpan(t *testing.T) {
	// "***" rather than "---": with the default Config, FrontMatter is
	// enabled, and a "-"-delimited line at document line 1 is claimed by
	// the frontmatter block parser (go.abhg.dev/goldmark/frontmatter),
	// which swallows the entire input as an unterminated frontmatter
	// block when no closing delimiter follows — an unrelated, pre-existing
	// default-config interaction, not a thematic-break/span concern. "*"
	// is not a frontmatter delimiter, so it exercises the same
	// ThematicBreak path without tripping over that.
	s := spanOf(t, "***\n", 0)
	if !s.IsZero() {
		t.Fatalf("thematic break should have zero span, got %+v", s)
	}
}
