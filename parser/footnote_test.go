package parser

import (
	"testing"

	"github.com/sriannamalai/markdownviewer/document"
)

func TestFootnotes(t *testing.T) {
	doc, err := Parse([]byte("text[^1]\n\n[^1]: the note\n"))
	if err != nil {
		t.Fatal(err)
	}
	got := document.Dump(doc)
	want := "Document\n  Paragraph\n    Text \"text\"\n    FootnoteRef[1]\n"
	if got != want {
		t.Fatalf("body:\n got %q\nwant %q", got, want)
	}
	if len(doc.Footnotes) != 1 || doc.Footnotes[0].Index != 1 {
		t.Fatalf("footnotes: %#v", doc.Footnotes)
	}
	if document.PlainText(doc.Footnotes[0]) != "the note" {
		t.Fatalf("footnote text: %q", document.PlainText(doc.Footnotes[0]))
	}
}
