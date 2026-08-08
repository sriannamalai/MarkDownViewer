package parser

import "testing"

func TestWikiLinks(t *testing.T) {
	assertDoc(t, "See [[Design Notes]] and [[Other Page|that page]].\n", `
Document
  Paragraph
    Text "See "
    WikiLink target="Design Notes"
      Text "Design Notes"
    Text " and "
    WikiLink target="Other Page"
      Text "that page"
    Text "."
`)
}
