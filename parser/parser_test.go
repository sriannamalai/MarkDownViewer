package parser

import (
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/sriannamalai/markdownviewer/document"
)

func assertDoc(t *testing.T, src, want string) {
	t.Helper()
	doc, err := Parse([]byte(src))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	got := strings.TrimSpace(document.Dump(doc))
	want = strings.TrimSpace(want)
	if got != want {
		t.Fatalf("tree mismatch:\n--- got ---\n%s\n--- want ---\n%s", got, want)
	}
}

func TestHeadingParagraph(t *testing.T) {
	assertDoc(t, "## Hello\n\nSome *em* **strong** `code`.\n", `
Document
  Heading[2] id="hello"
    Text "Hello"
  Paragraph
    Text "Some "
    Emphasis
      Text "em"
    Text " "
    Strong
      Text "strong"
    Text " "
    CodeSpan "code"
    Text "."
`)
}

func TestLinkImage(t *testing.T) {
	assertDoc(t, `[go](https://go.dev "Go") ![alt text](img.png)`, `
Document
  Paragraph
    Link dest="https://go.dev" title="Go"
      Text "go"
    Text " "
    Image dest="img.png" alt="alt text" title=""
      Text "alt text"
`)
}

func TestListsTightLoose(t *testing.T) {
	assertDoc(t, "- a\n- b\n", `
Document
  List ordered=false start=1 tight=true
    ListItem
      Paragraph
        Text "a"
    ListItem
      Paragraph
        Text "b"
`)
	assertDoc(t, "1. a\n\n2. b\n", `
Document
  List ordered=true start=1 tight=false
    ListItem
      Paragraph
        Text "a"
    ListItem
      Paragraph
        Text "b"
`)
}

func TestCodeBlockQuoteBreaks(t *testing.T) {
	assertDoc(t, "> quote\n\n```go\nx := 1\n```\n\n---\n", `
Document
  BlockQuote
    Paragraph
      Text "quote"
  CodeBlock lang="go" "x := 1\n"
  ThematicBreak
`)
}

func TestHardSoftBreaks(t *testing.T) {
	assertDoc(t, "line one\nline two  \nline three\n", `
Document
  Paragraph
    Text "line one"
    SoftBreak
    Text "line two"
    HardBreak
    Text "line three"
`)
}

func TestRawHTML(t *testing.T) {
	assertDoc(t, "<div>\nblock\n</div>\n\npara <em>inline</em>\n", `
Document
  HTMLBlock "<div>\nblock\n</div>\n"
  Paragraph
    Text "para "
    HTMLInline "<em>"
    Text "inline"
    HTMLInline "</em>"
`)
}

// TestBOMStripped verifies a leading UTF-8 byte order mark doesn't defeat
// block-level construct recognition (e.g. a BOM-prefixed "# Heading"
// rendering as a literal paragraph instead of a heading).
func TestBOMStripped(t *testing.T) {
	assertDoc(t, "\xEF\xBB\xBF# Heading\n", `
Document
  Heading[1] id="heading"
    Text "Heading"
`)
}

// TestInvalidUTF8Sanitized verifies invalid UTF-8 bytes in the source are
// replaced with the Unicode replacement character before any node holds
// them, rather than reaching MarshalJSON and being silently mangled there
// (see sanitizeUTF8 in parser.go). Without this, a document containing
// invalid UTF-8 fails to round-trip through the JSON codec: encoding/json
// replaces invalid bytes with U+FFFD on marshal, so the pre-JSON tree and
// the round-tripped tree diverge.
func TestInvalidUTF8Sanitized(t *testing.T) {
	src := []byte("a\xd1b\n")
	doc, err := Parse(src)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}

	text := document.PlainText(doc)
	if !utf8.ValidString(text) {
		t.Fatalf("PlainText is not valid UTF-8: %q", text)
	}
	if !strings.ContainsRune(text, utf8.RuneError) {
		t.Fatalf("expected U+FFFD replacement rune in %q", text)
	}

	data, err := document.MarshalJSON(doc)
	if err != nil {
		t.Fatalf("MarshalJSON: %v", err)
	}
	back, err := document.UnmarshalJSON(data)
	if err != nil {
		t.Fatalf("UnmarshalJSON: %v\njson: %s", err, data)
	}
	if document.Dump(doc) != document.Dump(back) {
		t.Fatalf("round trip changed tree:\n--- orig ---\n%s\n--- back ---\n%s",
			document.Dump(doc), document.Dump(back))
	}
}
