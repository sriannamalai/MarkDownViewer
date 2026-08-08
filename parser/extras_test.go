package parser

import "testing"

func TestFrontMatter(t *testing.T) {
	doc, err := Parse([]byte("---\ntitle: Hi\ntags: [a, b]\n---\n\nbody\n"))
	if err != nil {
		t.Fatal(err)
	}
	if doc.Meta["title"] != "Hi" {
		t.Fatalf("meta: %#v", doc.Meta)
	}
}

func TestEmoji(t *testing.T) {
	assertDoc(t, "hi :smile:\n", `
Document
  Paragraph
    Text "hi "
    Text "😄"
`)
}

func TestDefinitionList(t *testing.T) {
	assertDoc(t, "Term\n: definition\n", `
Document
  DefinitionList
    DefinitionTerm
      Text "Term"
    DefinitionDesc
      Paragraph
        Text "definition"
`)
}
