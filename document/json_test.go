package document

import (
	"bytes"
	"strings"
	"testing"
)

// fullDoc builds a document containing every node kind exactly once.
func fullDoc() *Document {
	doc := &Document{Meta: map[string]any{"title": "T", "n": float64(3)}}
	h := &Heading{Level: 2, AnchorID: "h"}
	h.AppendChild(&Text{Value: "H"})
	h.SetSpan(Span{StartLine: 1, EndLine: 1, StartOffset: 0, EndOffset: 4})
	p := &Paragraph{}
	em := &Emphasis{}
	em.AppendChild(&Text{Value: "e"})
	st := &Strong{}
	st.AppendChild(&Text{Value: "s"})
	del := &Strikethrough{}
	del.AppendChild(&Text{Value: "d"})
	lnk := &Link{Destination: "https://x", Title: "t"}
	lnk.AppendChild(&Text{Value: "l"})
	img := &Image{Destination: "i.png", Alt: "a", Title: "it"}
	wl := &WikiLink{Target: "Page"}
	wl.AppendChild(&Text{Value: "Page"})
	for _, n := range []Node{em, st, del, &CodeSpan{Value: "c"}, lnk, img, wl,
		&MathInline{Source: "x", Display: false}, &HTMLInline{HTML: "<b>"},
		&FootnoteRef{Index: 1}, &SoftBreak{}, &HardBreak{}} {
		p.AppendChild(n)
	}
	bq := &BlockQuote{}
	bq.AppendChild(&Paragraph{})
	adm := &Admonition{Variant: "note"}
	adm.AppendChild(&Paragraph{})
	list := &List{Ordered: true, Start: 3, Tight: true}
	li := &ListItem{Task: true, Checked: true}
	li.AppendChild(&Paragraph{})
	list.AppendChild(li)
	tbl := &Table{Alignments: []Alignment{AlignLeft, AlignNone, AlignRight, AlignCenter}}
	row := &TableRow{Header: true}
	cell := &TableCell{}
	cell.AppendChild(&Text{Value: "c"})
	row.AppendChild(cell)
	tbl.AppendChild(row)
	dl := &DefinitionList{}
	dt := &DefinitionTerm{}
	dt.AppendChild(&Text{Value: "t"})
	dd := &DefinitionDesc{}
	dd.AppendChild(&Paragraph{})
	dl.AppendChild(dt)
	dl.AppendChild(dd)
	def := &FootnoteDef{Index: 1}
	def.AppendChild(&Paragraph{})
	doc.Footnotes = []*FootnoteDef{def}
	for _, n := range []Node{h, p, bq, adm, list,
		&CodeBlock{Language: "go", Code: "x\n"}, &Diagram{Engine: "mermaid", Source: "g"},
		&MathBlock{Source: "m"}, tbl, &ThematicBreak{}, &HTMLBlock{HTML: "<hr>\n"}, dl} {
		doc.AppendChild(n)
	}
	return doc
}

func TestJSONRoundTripAllKinds(t *testing.T) {
	orig := fullDoc()
	data, err := MarshalJSON(orig)
	if err != nil {
		t.Fatal(err)
	}
	back, err := UnmarshalJSON(data)
	if err != nil {
		t.Fatalf("unmarshal: %v\njson: %s", err, data)
	}
	if Dump(orig) != Dump(back) {
		t.Fatalf("dump mismatch:\n--- orig ---\n%s\n--- back ---\n%s", Dump(orig), Dump(back))
	}
	if len(back.Footnotes) != 1 || back.Footnotes[0].Index != 1 {
		t.Fatalf("footnotes lost: %+v", back.Footnotes)
	}
	if back.Meta["title"] != "T" {
		t.Fatalf("meta lost: %+v", back.Meta)
	}
	if back.Children()[0].(*Heading).Span().StartOffset != 0 ||
		back.Children()[0].(*Heading).Span().EndOffset != 4 {
		t.Fatalf("span lost: %+v", back.Children()[0].(*Heading).Span())
	}
	again, err := MarshalJSON(back)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(data, again) {
		t.Fatal("re-marshal not byte-stable")
	}
}

func TestJSONWireShape(t *testing.T) {
	data, _ := MarshalJSON(fullDoc())
	s := string(data)
	for _, want := range []string{`"version":1`, `"kind":"document"`,
		`"kind":"heading"`, `"anchorId":"h"`, `"alignments":["left","none","right","center"]`,
		`"span":{"startLine":1,"endLine":1,"startOffset":0,"endOffset":4}`} {
		if !strings.Contains(s, want) {
			t.Errorf("wire missing %s", want)
		}
	}
	if strings.Contains(s, `"span":{}`) {
		t.Error("zero spans must be omitted")
	}
}

func TestJSONErrors(t *testing.T) {
	if _, err := UnmarshalJSON([]byte(`{"version":2,"kind":"document"}`)); err == nil {
		t.Error("version 2 must be rejected")
	}
	if _, err := UnmarshalJSON([]byte(`{"version":1,"kind":"document","children":[{"kind":"nosuch"}]}`)); err == nil {
		t.Error("unknown kind must be rejected")
	}
	if _, err := UnmarshalJSON([]byte(`not json`)); err == nil {
		t.Error("malformed input must be rejected")
	}
}
