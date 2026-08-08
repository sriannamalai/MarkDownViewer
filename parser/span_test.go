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
