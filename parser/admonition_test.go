package parser

import "testing"

func TestAdmonition(t *testing.T) {
	assertDoc(t, "> [!NOTE]\n> Useful info.\n", `
Document
  Admonition[note]
    Paragraph
      Text "Useful info."
`)
}

func TestAdmonitionMarkerOnlyParagraph(t *testing.T) {
	assertDoc(t, "> [!WARNING]\n>\n> Body here.\n", `
Document
  Admonition[warning]
    Paragraph
      Text "Body here."
`)
}

func TestPlainBlockquoteUntouched(t *testing.T) {
	assertDoc(t, "> [!UNKNOWN]\n> nope\n", `
Document
  BlockQuote
    Paragraph
      Text "[!UNKNOWN]"
      SoftBreak
      Text "nope"
`)
}
