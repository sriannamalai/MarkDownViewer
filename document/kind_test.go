package document

import "testing"

func TestKindValuesPinned(t *testing.T) {
	// The numeric values are a compatibility contract (serialization, FFI).
	// Never renumber; append only.
	pins := map[Kind]int{
		KindDocument: 0, KindHeading: 1, KindParagraph: 2, KindBlockQuote: 3,
		KindAdmonition: 4, KindList: 5, KindListItem: 6, KindCodeBlock: 7,
		KindDiagram: 8, KindMathBlock: 9, KindTable: 10, KindTableRow: 11,
		KindTableCell: 12, KindThematicBreak: 13, KindHTMLBlock: 14,
		KindDefinitionList: 15, KindDefinitionTerm: 16, KindDefinitionDesc: 17,
		KindFootnoteDef: 18, KindText: 19, KindSoftBreak: 20, KindHardBreak: 21,
		KindEmphasis: 22, KindStrong: 23, KindStrikethrough: 24, KindCodeSpan: 25,
		KindLink: 26, KindImage: 27, KindWikiLink: 28, KindMathInline: 29,
		KindHTMLInline: 30, KindFootnoteRef: 31,
	}
	for k, want := range pins {
		if int(k) != want {
			t.Errorf("%s = %d, want %d", k, int(k), want)
		}
	}
	if len(pins) != 32 {
		t.Fatalf("pin table has %d entries, want 32", len(pins))
	}
}

func TestKindStringRoundTrip(t *testing.T) {
	for k := KindDocument; k <= KindFootnoteRef; k++ {
		s := k.String()
		if s == "" || s == "unknown" {
			t.Errorf("kind %d has no name", int(k))
		}
		back, ok := KindFromString(s)
		if !ok || back != k {
			t.Errorf("round trip %d -> %q -> %d ok=%t", int(k), s, int(back), ok)
		}
	}
	if _, ok := KindFromString("nosuch"); ok {
		t.Error("unknown name must not resolve")
	}
	if got := Kind(99).String(); got != "unknown" {
		t.Errorf("out of range String() = %q", got)
	}
}
