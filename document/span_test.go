package document

import "testing"

func TestSpanAccessors(t *testing.T) {
	p := &Paragraph{}
	if !p.Span().IsZero() {
		t.Fatal("new node must have zero span")
	}
	s := Span{StartLine: 2, EndLine: 3, StartOffset: 10, EndOffset: 40}
	p.SetSpan(s)
	if p.Span() != s {
		t.Fatalf("got %+v", p.Span())
	}
	if s.IsZero() {
		t.Fatal("populated span must not be zero")
	}
}
