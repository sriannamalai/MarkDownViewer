# Markdown Viewer Library V1 Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build the v1 Markdown viewer library per the approved spec: a stable `document` model, goldmark-based parser, themed self-contained HTML renderer, and `mdview` CLI.

**Architecture:** Three-stage pipeline — `parser` (goldmark internal) → `document` (public typed AST, zero non-stdlib imports) → `render/html` (themed, sanitized, self-contained HTML). Facade `mdviewer.go` wires them; `cmd/mdview` is the living example.

**Tech Stack:** Go ≥1.23, goldmark v1.7.x (+ extension, goldmark-emoji, go.abhg.dev/goldmark/frontmatter, go.abhg.dev/goldmark/wikilink), chroma/v2, bluemonday, embedded mermaid 11.4.1 + KaTeX 0.16.21.

## Global Constraints

- Module path: `github.com/sriannamalai/markdownviewer`. Go `1.23` in go.mod.
- `document` package imports **nothing** outside the Go standard library.
- No CDN/network references in any rendered output; all assets `go:embed`ed.
- Sanitize-by-default: raw HTML through bluemonday UGC policy; `javascript:`/`vbscript:`/`data:` URLs stripped unless `Unsafe`.
- The library never reads the filesystem during render; hosts inject a `Resolver`.
- Commits: conventional style (`feat(parser): …`). **No AI-attribution trailers of any kind** (user rule). If GPG signing fails (1Password locked), commit with `--no-gpg-sign`.
- Tests accompany every task; run `go test ./...` before every commit. Golden files regenerate with `go test ./render/html -update` — always eyeball the diff before committing regenerated goldens.
- Spec reference: `docs/superpowers/specs/2026-08-08 Markdown Viewer Library Design.md`.

---

### Task 1: Module bootstrap + `document` package

**Files:**
- Create: `go.mod`, `document/document.go`, `document/walk.go`, `document/dump.go`
- Test: `document/document_test.go`

**Interfaces:**
- Consumes: nothing (first task).
- Produces (used by every later task):
  - `type Node interface { Kind() Kind; Children() []Node; AppendChild(Node) }`
  - `type Container struct` (embeddable base implementing Children/AppendChild)
  - Concrete node structs listed below, each with `Kind() Kind`.
  - `func Walk(n Node, visit Visitor)` with `type Visitor func(n Node, entering bool) WalkStatus`; `WalkStatus` ∈ `Continue, SkipChildren, Stop`.
  - `func Dump(n Node) string` — indented tree dump used by parser tests.
  - `func PlainText(n Node) string` — concatenated Text/CodeSpan values.

- [ ] **Step 1: Init module**

```bash
cd /Users/sri/Developer/Sri/MarkDownViewer
go mod init github.com/sriannamalai/markdownviewer
```

- [ ] **Step 2: Write failing test**

`document/document_test.go`:

```go
package document

import (
	"strings"
	"testing"
)

func sample() *Document {
	doc := &Document{}
	h := &Heading{Level: 2, AnchorID: "hello"}
	h.AppendChild(&Text{Value: "Hello"})
	p := &Paragraph{}
	em := &Emphasis{}
	em.AppendChild(&Text{Value: "world"})
	p.AppendChild(em)
	doc.AppendChild(h)
	doc.AppendChild(p)
	return doc
}

func TestWalkOrder(t *testing.T) {
	var got []string
	Walk(sample(), func(n Node, entering bool) WalkStatus {
		if entering {
			got = append(got, label(n))
		}
		return Continue
	})
	want := []string{`Document`, `Heading[2] id="hello"`, `Text "Hello"`, `Paragraph`, `Emphasis`, `Text "world"`}
	if strings.Join(got, "|") != strings.Join(want, "|") {
		t.Fatalf("walk order:\n got %v\nwant %v", got, want)
	}
}

func TestWalkSkipChildren(t *testing.T) {
	var got []string
	Walk(sample(), func(n Node, entering bool) WalkStatus {
		if entering {
			got = append(got, label(n))
			if n.Kind() == KindHeading {
				return SkipChildren
			}
		}
		return Continue
	})
	for _, l := range got {
		if l == `Text "Hello"` {
			t.Fatal("SkipChildren did not skip heading text")
		}
	}
}

func TestDump(t *testing.T) {
	want := "Document\n  Heading[2] id=\"hello\"\n    Text \"Hello\"\n  Paragraph\n    Emphasis\n      Text \"world\"\n"
	if got := Dump(sample()); got != want {
		t.Fatalf("dump:\n got %q\nwant %q", got, want)
	}
}

func TestPlainText(t *testing.T) {
	if got := PlainText(sample()); got != "Helloworld" {
		t.Fatalf("plaintext: %q", got)
	}
}
```

- [ ] **Step 3: Run test, expect FAIL** — `go test ./document` → undefined types.

- [ ] **Step 4: Implement `document/document.go`**

```go
// Package document defines the stable, renderer-agnostic Markdown document
// model. It is the public contract between the parser and all renderers and
// must import nothing outside the standard library.
package document

type Kind int

const (
	KindDocument Kind = iota
	KindHeading
	KindParagraph
	KindBlockQuote
	KindAdmonition
	KindList
	KindListItem
	KindCodeBlock
	KindDiagram
	KindMathBlock
	KindTable
	KindTableRow
	KindTableCell
	KindThematicBreak
	KindHTMLBlock
	KindDefinitionList
	KindDefinitionTerm
	KindDefinitionDesc
	KindFootnoteDef
	KindText
	KindSoftBreak
	KindHardBreak
	KindEmphasis
	KindStrong
	KindStrikethrough
	KindCodeSpan
	KindLink
	KindImage
	KindWikiLink
	KindMathInline
	KindHTMLInline
	KindFootnoteRef
)

type Node interface {
	Kind() Kind
	Children() []Node
	AppendChild(Node)
}

// Container is the embeddable base for all nodes.
type Container struct{ kids []Node }

func (c *Container) Children() []Node   { return c.kids }
func (c *Container) AppendChild(n Node) { c.kids = append(c.kids, n) }

type Alignment int

const (
	AlignNone Alignment = iota
	AlignLeft
	AlignCenter
	AlignRight
)

// Block nodes.
type Document struct {
	Container
	Meta      map[string]any // front-matter, nil if absent
	Footnotes []*FootnoteDef
}
type Heading struct {
	Container
	Level    int
	AnchorID string
}
type Paragraph struct{ Container }
type BlockQuote struct{ Container }
type Admonition struct {
	Container
	Variant string // note|tip|important|warning|caution
}
type List struct {
	Container
	Ordered bool
	Start   int
	Tight   bool
}
type ListItem struct {
	Container
	Task    bool
	Checked bool
}
type CodeBlock struct {
	Container
	Language string
	Code     string
}
type Diagram struct {
	Container
	Engine string // "mermaid"
	Source string
}
type MathBlock struct {
	Container
	Source string
}
type Table struct {
	Container
	Alignments []Alignment
}
type TableRow struct {
	Container
	Header bool
}
type TableCell struct{ Container }
type ThematicBreak struct{ Container }
type HTMLBlock struct {
	Container
	HTML string
}
type DefinitionList struct{ Container }
type DefinitionTerm struct{ Container }
type DefinitionDesc struct{ Container }
type FootnoteDef struct {
	Container
	Index int
}

// Inline nodes.
type Text struct {
	Container
	Value string
}
type SoftBreak struct{ Container }
type HardBreak struct{ Container }
type Emphasis struct{ Container }
type Strong struct{ Container }
type Strikethrough struct{ Container }
type CodeSpan struct {
	Container
	Value string
}
type Link struct {
	Container
	Destination string
	Title       string
}
type Image struct {
	Container
	Destination string
	Title       string
	Alt         string
}
type WikiLink struct {
	Container
	Target string
}
type MathInline struct {
	Container
	Source  string
	Display bool
}
type HTMLInline struct {
	Container
	HTML string
}
type FootnoteRef struct {
	Container
	Index int
}

func (*Document) Kind() Kind       { return KindDocument }
func (*Heading) Kind() Kind        { return KindHeading }
func (*Paragraph) Kind() Kind      { return KindParagraph }
func (*BlockQuote) Kind() Kind     { return KindBlockQuote }
func (*Admonition) Kind() Kind     { return KindAdmonition }
func (*List) Kind() Kind           { return KindList }
func (*ListItem) Kind() Kind       { return KindListItem }
func (*CodeBlock) Kind() Kind      { return KindCodeBlock }
func (*Diagram) Kind() Kind        { return KindDiagram }
func (*MathBlock) Kind() Kind      { return KindMathBlock }
func (*Table) Kind() Kind          { return KindTable }
func (*TableRow) Kind() Kind       { return KindTableRow }
func (*TableCell) Kind() Kind      { return KindTableCell }
func (*ThematicBreak) Kind() Kind  { return KindThematicBreak }
func (*HTMLBlock) Kind() Kind      { return KindHTMLBlock }
func (*DefinitionList) Kind() Kind { return KindDefinitionList }
func (*DefinitionTerm) Kind() Kind { return KindDefinitionTerm }
func (*DefinitionDesc) Kind() Kind { return KindDefinitionDesc }
func (*FootnoteDef) Kind() Kind    { return KindFootnoteDef }
func (*Text) Kind() Kind           { return KindText }
func (*SoftBreak) Kind() Kind      { return KindSoftBreak }
func (*HardBreak) Kind() Kind      { return KindHardBreak }
func (*Emphasis) Kind() Kind       { return KindEmphasis }
func (*Strong) Kind() Kind         { return KindStrong }
func (*Strikethrough) Kind() Kind  { return KindStrikethrough }
func (*CodeSpan) Kind() Kind       { return KindCodeSpan }
func (*Link) Kind() Kind           { return KindLink }
func (*Image) Kind() Kind          { return KindImage }
func (*WikiLink) Kind() Kind       { return KindWikiLink }
func (*MathInline) Kind() Kind     { return KindMathInline }
func (*HTMLInline) Kind() Kind     { return KindHTMLInline }
func (*FootnoteRef) Kind() Kind    { return KindFootnoteRef }
```

- [ ] **Step 5: Implement `document/walk.go`**

```go
package document

type WalkStatus int

const (
	Continue WalkStatus = iota
	SkipChildren
	Stop
)

type Visitor func(n Node, entering bool) WalkStatus

// Walk visits n and its descendants depth-first, calling visit on enter and
// exit. SkipChildren (on enter) skips the subtree; Stop aborts the walk.
func Walk(n Node, visit Visitor) {
	walk(n, visit)
}

func walk(n Node, visit Visitor) WalkStatus {
	switch visit(n, true) {
	case Stop:
		return Stop
	case SkipChildren:
		return visit(n, false)
	}
	for _, c := range n.Children() {
		if walk(c, visit) == Stop {
			return Stop
		}
	}
	return visit(n, false)
}

// PlainText returns the concatenated Text and CodeSpan content of a subtree.
func PlainText(n Node) string {
	var out []byte
	Walk(n, func(n Node, entering bool) WalkStatus {
		if !entering {
			return Continue
		}
		switch n := n.(type) {
		case *Text:
			out = append(out, n.Value...)
		case *CodeSpan:
			out = append(out, n.Value...)
		}
		return Continue
	})
	return string(out)
}
```

- [ ] **Step 6: Implement `document/dump.go`**

```go
package document

import (
	"fmt"
	"strings"
)

// Dump renders a node tree as an indented text outline for tests.
func Dump(n Node) string {
	var b strings.Builder
	dump(&b, n, 0)
	return b.String()
}

func dump(b *strings.Builder, n Node, depth int) {
	b.WriteString(strings.Repeat("  ", depth))
	b.WriteString(label(n))
	b.WriteByte('\n')
	for _, c := range n.Children() {
		dump(b, c, depth+1)
	}
}

func label(n Node) string {
	switch n := n.(type) {
	case *Document:
		return "Document"
	case *Heading:
		return fmt.Sprintf("Heading[%d] id=%q", n.Level, n.AnchorID)
	case *Paragraph:
		return "Paragraph"
	case *BlockQuote:
		return "BlockQuote"
	case *Admonition:
		return fmt.Sprintf("Admonition[%s]", n.Variant)
	case *List:
		return fmt.Sprintf("List ordered=%t start=%d tight=%t", n.Ordered, n.Start, n.Tight)
	case *ListItem:
		if n.Task {
			return fmt.Sprintf("ListItem task checked=%t", n.Checked)
		}
		return "ListItem"
	case *CodeBlock:
		return fmt.Sprintf("CodeBlock lang=%q %q", n.Language, n.Code)
	case *Diagram:
		return fmt.Sprintf("Diagram[%s] %q", n.Engine, n.Source)
	case *MathBlock:
		return fmt.Sprintf("MathBlock %q", n.Source)
	case *Table:
		parts := make([]string, len(n.Alignments))
		for i, a := range n.Alignments {
			parts[i] = [...]string{"none", "left", "center", "right"}[a]
		}
		return fmt.Sprintf("Table aligns=[%s]", strings.Join(parts, ","))
	case *TableRow:
		return fmt.Sprintf("TableRow header=%t", n.Header)
	case *TableCell:
		return "TableCell"
	case *ThematicBreak:
		return "ThematicBreak"
	case *HTMLBlock:
		return fmt.Sprintf("HTMLBlock %q", n.HTML)
	case *DefinitionList:
		return "DefinitionList"
	case *DefinitionTerm:
		return "DefinitionTerm"
	case *DefinitionDesc:
		return "DefinitionDesc"
	case *FootnoteDef:
		return fmt.Sprintf("FootnoteDef[%d]", n.Index)
	case *Text:
		return fmt.Sprintf("Text %q", n.Value)
	case *SoftBreak:
		return "SoftBreak"
	case *HardBreak:
		return "HardBreak"
	case *Emphasis:
		return "Emphasis"
	case *Strong:
		return "Strong"
	case *Strikethrough:
		return "Strikethrough"
	case *CodeSpan:
		return fmt.Sprintf("CodeSpan %q", n.Value)
	case *Link:
		return fmt.Sprintf("Link dest=%q title=%q", n.Destination, n.Title)
	case *Image:
		return fmt.Sprintf("Image dest=%q alt=%q title=%q", n.Destination, n.Alt, n.Title)
	case *WikiLink:
		return fmt.Sprintf("WikiLink target=%q", n.Target)
	case *MathInline:
		return fmt.Sprintf("MathInline display=%t %q", n.Display, n.Source)
	case *HTMLInline:
		return fmt.Sprintf("HTMLInline %q", n.HTML)
	case *FootnoteRef:
		return fmt.Sprintf("FootnoteRef[%d]", n.Index)
	default:
		return fmt.Sprintf("UNKNOWN(%T)", n)
	}
}
```

- [ ] **Step 7: Run tests, expect PASS** — `go test ./document -v`

- [ ] **Step 8: Commit**

```bash
git add go.mod document/
git commit -m "feat(document): stable document model with Walk, Dump, PlainText"
```

---

### Task 2: Parser core — CommonMark → document model

**Files:**
- Create: `parser/parser.go`, `parser/config.go`, `parser/transform.go`, `parser/slug.go`
- Test: `parser/parser_test.go`, `parser/slug_test.go`

**Interfaces:**
- Consumes: everything from `document`.
- Produces:
  - `func Parse(src []byte) (*document.Document, error)` — full default config.
  - `func ParseWith(src []byte, cfg Config) (*document.Document, error)`
  - `type Config struct { Tables, Strikethrough, TaskLists, Linkify, Footnotes, DefinitionLists, FrontMatter, Emoji, WikiLinks, Math, Admonitions bool }`
  - `func Default() Config` (all true), `func CommonMarkOnly() Config` (all false).
  - Heading `AnchorID` is always populated (GitHub-style slug, deduped `-1`, `-2`…).

- [ ] **Step 1: Add dependency**

```bash
go get github.com/yuin/goldmark@latest
```

- [ ] **Step 2: Write failing tests**

`parser/slug_test.go`:

```go
package parser

import "testing"

func TestSlugify(t *testing.T) {
	cases := map[string]string{
		"Hello World":        "hello-world",
		"What's New in 2.0?": "whats-new-in-20",
		"  spaces  ":         "spaces",
		"émoji ✨ ok":         "émoji--ok",
	}
	for in, want := range cases {
		if got := slugify(in); got != want {
			t.Errorf("slugify(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestSlugDedupe(t *testing.T) {
	tr := &transformer{slugs: map[string]int{}}
	if got := tr.slug("Intro"); got != "intro" {
		t.Fatalf("first: %q", got)
	}
	if got := tr.slug("Intro"); got != "intro-1" {
		t.Fatalf("second: %q", got)
	}
	if got := tr.slug("Intro"); got != "intro-2" {
		t.Fatalf("third: %q", got)
	}
}
```

`parser/parser_test.go` (dump-based; the assertDoc helper is reused by Tasks 3–8):

```go
package parser

import (
	"strings"
	"testing"

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
```

- [ ] **Step 3: Run, expect FAIL** — `go test ./parser` → undefined.

- [ ] **Step 4: Implement `parser/config.go`**

```go
package parser

// Config selects which syntax extensions are enabled. The zero value is
// pure CommonMark; Default() enables everything.
type Config struct {
	Tables, Strikethrough, TaskLists, Linkify   bool
	Footnotes, DefinitionLists, FrontMatter     bool
	Emoji, WikiLinks, Math, Admonitions         bool
}

func Default() Config {
	return Config{
		Tables: true, Strikethrough: true, TaskLists: true, Linkify: true,
		Footnotes: true, DefinitionLists: true, FrontMatter: true,
		Emoji: true, WikiLinks: true, Math: true, Admonitions: true,
	}
}

func CommonMarkOnly() Config { return Config{} }
```

- [ ] **Step 5: Implement `parser/slug.go`**

```go
package parser

import (
	"fmt"
	"strings"
	"unicode"
)

// slugify produces a GitHub-style anchor slug: lowercase, spaces to hyphens,
// punctuation dropped, letters/digits/hyphen/underscore kept.
func slugify(s string) string {
	var b strings.Builder
	for _, r := range strings.TrimSpace(s) {
		switch {
		case unicode.IsLetter(r) || unicode.IsDigit(r) || r == '_' || r == '-':
			b.WriteRune(unicode.ToLower(r))
		case r == ' ':
			b.WriteByte('-')
		}
	}
	return b.String()
}

func (t *transformer) slug(text string) string {
	s := slugify(text)
	n, seen := t.slugs[s]
	t.slugs[s] = n + 1
	if seen {
		return fmt.Sprintf("%s-%d", s, n)
	}
	return s
}
```

- [ ] **Step 6: Implement `parser/parser.go`**

```go
// Package parser converts Markdown source into the document model. goldmark
// is an internal implementation detail and must not leak into the API.
package parser

import (
	"github.com/yuin/goldmark"
	gparser "github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/text"

	"github.com/sriannamalai/markdownviewer/document"
)

func Parse(src []byte) (*document.Document, error) {
	return ParseWith(src, Default())
}

func ParseWith(src []byte, cfg Config) (*document.Document, error) {
	md := build(cfg)
	ctx := gparser.NewContext()
	root := md.Parser().Parse(text.NewReader(src), gparser.WithContext(ctx))
	t := &transformer{src: src, cfg: cfg, slugs: map[string]int{}}
	return t.document(root, ctx), nil
}

// build assembles the goldmark instance for cfg. Extensions are appended
// here in Tasks 3–8 as they are implemented.
func build(cfg Config) goldmark.Markdown {
	var opts []goldmark.Option
	if exts := extensions(cfg); len(exts) > 0 {
		opts = append(opts, goldmark.WithExtensions(exts...))
	}
	return goldmark.New(opts...)
}

func extensions(cfg Config) []goldmark.Extender {
	var exts []goldmark.Extender
	// Populated by Tasks 3–8.
	return exts
}
```

- [ ] **Step 7: Implement `parser/transform.go`**

```go
package parser

import (
	"strings"

	"github.com/yuin/goldmark/ast"
	gparser "github.com/yuin/goldmark/parser"

	"github.com/sriannamalai/markdownviewer/document"
)

type transformer struct {
	src   []byte
	cfg   Config
	slugs map[string]int
	item  *document.ListItem // innermost list item, for task checkboxes
}

func (t *transformer) document(root ast.Node, ctx gparser.Context) *document.Document {
	doc := &document.Document{}
	t.appendChildren(doc, root)
	_ = ctx // front-matter extraction added in Task 5
	return doc
}

func (t *transformer) appendChildren(parent document.Node, n ast.Node) {
	for c := n.FirstChild(); c != nil; c = c.NextSibling() {
		if out := t.convert(c); out != nil {
			parent.AppendChild(out)
			// ast.Text carries trailing break flags; emit them as siblings.
			if txt, ok := c.(*ast.Text); ok {
				if txt.HardLineBreak() {
					parent.AppendChild(&document.HardBreak{})
				} else if txt.SoftLineBreak() {
					parent.AppendChild(&document.SoftBreak{})
				}
			}
		}
	}
}

func (t *transformer) convert(n ast.Node) document.Node {
	switch n := n.(type) {
	case *ast.Heading:
		h := &document.Heading{Level: n.Level}
		t.appendChildren(h, n)
		h.AnchorID = t.slug(document.PlainText(h))
		return h
	case *ast.Paragraph:
		p := &document.Paragraph{}
		t.appendChildren(p, n)
		return p
	case *ast.TextBlock: // tight-list item content; List.Tight drives <p> omission
		p := &document.Paragraph{}
		t.appendChildren(p, n)
		return p
	case *ast.Blockquote:
		bq := &document.BlockQuote{}
		t.appendChildren(bq, n)
		return bq // Admonition promotion added in Task 6
	case *ast.List:
		l := &document.List{Ordered: n.IsOrdered(), Start: 1, Tight: n.IsTight}
		if n.IsOrdered() {
			l.Start = n.Start
		}
		t.appendChildren(l, n)
		return l
	case *ast.ListItem:
		li := &document.ListItem{}
		prev := t.item
		t.item = li
		t.appendChildren(li, n)
		t.item = prev
		return li
	case *ast.FencedCodeBlock:
		lang := string(n.Language(t.src))
		return t.codeOrSpecial(lang, blockLines(n, t.src))
	case *ast.CodeBlock:
		return &document.CodeBlock{Code: blockLines(n, t.src)}
	case *ast.ThematicBreak:
		return &document.ThematicBreak{}
	case *ast.HTMLBlock:
		html := blockLines(n, t.src)
		if n.HasClosure() {
			html += string(n.ClosureLine.Value(t.src))
		}
		return &document.HTMLBlock{HTML: html}
	case *ast.Text:
		return &document.Text{Value: string(n.Segment.Value(t.src))}
	case *ast.String:
		return &document.Text{Value: string(n.Value)}
	case *ast.Emphasis:
		var out document.Node
		if n.Level >= 2 {
			out = &document.Strong{}
		} else {
			out = &document.Emphasis{}
		}
		t.appendChildren(out, n)
		return out
	case *ast.CodeSpan:
		return &document.CodeSpan{Value: spanText(n, t.src)}
	case *ast.Link:
		l := &document.Link{Destination: string(n.Destination), Title: string(n.Title)}
		t.appendChildren(l, n)
		return l
	case *ast.AutoLink:
		url := string(n.URL(t.src))
		if n.AutoLinkType == ast.AutoLinkEmail && !strings.HasPrefix(url, "mailto:") {
			url = "mailto:" + url
		}
		l := &document.Link{Destination: url}
		l.AppendChild(&document.Text{Value: string(n.Label(t.src))})
		return l
	case *ast.Image:
		img := &document.Image{Destination: string(n.Destination), Title: string(n.Title)}
		t.appendChildren(img, n)
		img.Alt = document.PlainText(img)
		return img
	case *ast.RawHTML:
		var b strings.Builder
		for i := 0; i < n.Segments.Len(); i++ {
			b.Write(n.Segments.At(i).Value(t.src))
		}
		return &document.HTMLInline{HTML: b.String()}
	default:
		return nil // extension nodes handled in Tasks 3-8
	}
}

// codeOrSpecial maps special fence languages to dedicated nodes.
func (t *transformer) codeOrSpecial(lang, code string) document.Node {
	switch lang {
	case "mermaid":
		return &document.Diagram{Engine: "mermaid", Source: code}
	case "math":
		return &document.MathBlock{Source: code}
	}
	return &document.CodeBlock{Language: lang, Code: code}
}

func blockLines(n ast.Node, src []byte) string {
	var b strings.Builder
	l := n.Lines()
	for i := 0; i < l.Len(); i++ {
		b.Write(l.At(i).Value(src))
	}
	return b.String()
}

func spanText(n ast.Node, src []byte) string {
	var b strings.Builder
	for c := n.FirstChild(); c != nil; c = c.NextSibling() {
		if txt, ok := c.(*ast.Text); ok {
			b.Write(txt.Segment.Value(src))
		}
	}
	return b.String()
}
```

- [ ] **Step 8: Run tests until PASS** — `go test ./parser -v`. Debug tree mismatches by printing `document.Dump`.

- [ ] **Step 9: Commit**

```bash
git add parser/ go.mod go.sum
git commit -m "feat(parser): CommonMark core transform to document model"
```

---

### Task 3: Parser GFM — tables, strikethrough, task lists, autolinks

**Files:**
- Modify: `parser/parser.go` (`extensions()`), `parser/transform.go` (new cases)
- Test: `parser/gfm_test.go`

**Interfaces:**
- Consumes: Task 2's `transformer`, `Config` flags `Tables/Strikethrough/TaskLists/Linkify`.
- Produces: `document.Table/TableRow/TableCell` (with `Alignments`, first row `Header=true`), `document.Strikethrough`, `ListItem.Task/Checked` populated.

- [ ] **Step 1: Write failing tests** — `parser/gfm_test.go`:

```go
package parser

import "testing"

func TestTable(t *testing.T) {
	assertDoc(t, "| a | b |\n|:--|--:|\n| 1 | 2 |\n", `
Document
  Table aligns=[left,right]
    TableRow header=true
      TableCell
        Text "a"
      TableCell
        Text "b"
    TableRow header=false
      TableCell
        Text "1"
      TableCell
        Text "2"
`)
}

func TestStrikethroughTaskAutolink(t *testing.T) {
	assertDoc(t, "~~gone~~ visit www.example.com\n\n- [x] done\n- [ ] todo\n", `
Document
  Paragraph
    Strikethrough
      Text "gone"
    Text " visit "
    Link dest="http://www.example.com" title=""
      Text "www.example.com"
  List ordered=false start=1 tight=true
    ListItem task checked=true
      Paragraph
        Text "done"
    ListItem task checked=false
      Paragraph
        Text "todo"
`)
}
```

- [ ] **Step 2: Run, expect FAIL** (nodes come out missing/nil).

- [ ] **Step 3: Implement** — in `parser/parser.go` `extensions()` add:

```go
import "github.com/yuin/goldmark/extension"

if cfg.Tables {
	exts = append(exts, extension.Table)
}
if cfg.Strikethrough {
	exts = append(exts, extension.Strikethrough)
}
if cfg.TaskLists {
	exts = append(exts, extension.TaskList)
}
if cfg.Linkify {
	exts = append(exts, extension.Linkify)
}
```

In `parser/transform.go` `convert()` add cases (import `east "github.com/yuin/goldmark/extension/ast"`):

```go
case *east.Table:
	tbl := &document.Table{}
	for _, a := range n.Alignments {
		tbl.Alignments = append(tbl.Alignments, map[east.Alignment]document.Alignment{
			east.AlignNone: document.AlignNone, east.AlignLeft: document.AlignLeft,
			east.AlignCenter: document.AlignCenter, east.AlignRight: document.AlignRight,
		}[a])
	}
	t.appendChildren(tbl, n)
	return tbl
case *east.TableHeader:
	row := &document.TableRow{Header: true}
	t.appendChildren(row, n)
	return row
case *east.TableRow:
	row := &document.TableRow{}
	t.appendChildren(row, n)
	return row
case *east.TableCell:
	cell := &document.TableCell{}
	t.appendChildren(cell, n)
	return cell
case *east.Strikethrough:
	s := &document.Strikethrough{}
	t.appendChildren(s, n)
	return s
case *east.TaskCheckBox:
	if t.item != nil {
		t.item.Task = true
		t.item.Checked = n.IsChecked
	}
	return nil
```

- [ ] **Step 4: Run until PASS** — `go test ./parser -v`
- [ ] **Step 5: Commit** — `git add parser/ && git commit -m "feat(parser): GFM tables, strikethrough, task lists, autolinks"`

---

### Task 4: Parser — footnotes

**Files:**
- Modify: `parser/parser.go`, `parser/transform.go`
- Test: `parser/footnote_test.go`

**Interfaces:**
- Consumes: `Config.Footnotes`.
- Produces: `document.FootnoteRef{Index}` inline; `Document.Footnotes []*FootnoteDef` (defs are **not** left in the body tree).

- [ ] **Step 1: Failing test** — `parser/footnote_test.go`:

```go
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
```

- [ ] **Step 2: Run, expect FAIL.**
- [ ] **Step 3: Implement** — `extensions()`: `if cfg.Footnotes { exts = append(exts, extension.Footnote) }`. Give `transformer` a `footnotes []*document.FootnoteDef` field; in `convert()`:

```go
case *east.FootnoteLink:
	return &document.FootnoteRef{Index: n.Index}
case *east.FootnoteBacklink:
	return nil // renderer generates backlinks
case *east.FootnoteList:
	for c := n.FirstChild(); c != nil; c = c.NextSibling() {
		if def, ok := t.convert(c).(*document.FootnoteDef); ok {
			t.footnotes = append(t.footnotes, def)
		}
	}
	return nil
case *east.Footnote:
	def := &document.FootnoteDef{Index: n.Index}
	t.appendChildren(def, n)
	return def
```

In `document()` after `appendChildren`: `doc.Footnotes = t.footnotes`.

- [ ] **Step 4: Run until PASS.**
- [ ] **Step 5: Commit** — `git commit -am "feat(parser): footnotes collected onto Document.Footnotes"`

---

### Task 5: Parser — front-matter, emoji, definition lists

**Files:**
- Modify: `parser/parser.go`, `parser/transform.go`
- Test: `parser/extras_test.go`

**Interfaces:**
- Consumes: `Config.FrontMatter/Emoji/DefinitionLists`.
- Produces: `Document.Meta map[string]any`; emoji become plain `Text` (unicode); `DefinitionList/Term/Desc` nodes.

- [ ] **Step 1: Deps**

```bash
go get go.abhg.dev/goldmark/frontmatter@latest github.com/yuin/goldmark-emoji@latest
```

- [ ] **Step 2: Failing tests** — `parser/extras_test.go`:

```go
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
```

- [ ] **Step 3: Run, expect FAIL.**
- [ ] **Step 4: Implement** — `extensions()` additions:

```go
import (
	emoji "github.com/yuin/goldmark-emoji"
	"go.abhg.dev/goldmark/frontmatter"
)

if cfg.DefinitionLists {
	exts = append(exts, extension.DefinitionList)
}
if cfg.FrontMatter {
	exts = append(exts, &frontmatter.Extender{})
}
if cfg.Emoji {
	exts = append(exts, emoji.Emoji)
}
```

`transform.go` — in `document()`:

```go
if d := frontmatter.Get(ctx); d != nil {
	var m map[string]any
	if err := d.Decode(&m); err == nil {
		doc.Meta = m
	}
}
```

`convert()` cases (import `emast "github.com/yuin/goldmark-emoji/ast"`):

```go
case *emast.Emoji:
	return &document.Text{Value: string(n.Value.Unicode)}
case *east.DefinitionList:
	dl := &document.DefinitionList{}
	t.appendChildren(dl, n)
	return dl
case *east.DefinitionTerm:
	dt := &document.DefinitionTerm{}
	t.appendChildren(dt, n)
	return dt
case *east.DefinitionDescription:
	dd := &document.DefinitionDesc{}
	t.appendChildren(dd, n)
	return dd
```

- [ ] **Step 5: Run until PASS.** (If `TextBlock` inside DefinitionDescription dumps as `Paragraph`, that is the expected mapping.)
- [ ] **Step 6: Commit** — `git commit -am "feat(parser): front-matter metadata, emoji shortcodes, definition lists"`

---

### Task 6: Parser — admonitions (`> [!NOTE]`)

**Files:**
- Modify: `parser/transform.go` (Blockquote case)
- Test: `parser/admonition_test.go`

**Interfaces:**
- Consumes: `Config.Admonitions`.
- Produces: `document.Admonition{Variant}` replacing qualifying BlockQuotes; marker text stripped. Variants: `note|tip|important|warning|caution` (lowercased).

- [ ] **Step 1: Failing test** — `parser/admonition_test.go`:

```go
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
```

- [ ] **Step 2: Run, expect FAIL.**
- [ ] **Step 3: Implement** — replace the Blockquote case:

```go
case *ast.Blockquote:
	bq := &document.BlockQuote{}
	t.appendChildren(bq, n)
	if t.cfg.Admonitions {
		if adm := promoteAdmonition(bq); adm != nil {
			return adm
		}
	}
	return bq
```

New function in `transform.go`:

```go
var admonitionVariants = map[string]bool{
	"NOTE": true, "TIP": true, "IMPORTANT": true, "WARNING": true, "CAUTION": true,
}

// promoteAdmonition returns an Admonition if bq's first paragraph starts
// with a GitHub-style [!VARIANT] marker; the marker (and a following
// SoftBreak) is stripped. Returns nil when bq is a plain quote.
func promoteAdmonition(bq *document.BlockQuote) *document.Admonition {
	kids := bq.Children()
	if len(kids) == 0 {
		return nil
	}
	p, ok := kids[0].(*document.Paragraph)
	if !ok {
		return nil
	}
	inl := p.Children()
	if len(inl) == 0 {
		return nil
	}
	txt, ok := inl[0].(*document.Text)
	if !ok || len(txt.Value) < 4 || !strings.HasPrefix(txt.Value, "[!") || !strings.HasSuffix(txt.Value, "]") {
		return nil
	}
	name := txt.Value[2 : len(txt.Value)-1]
	if !admonitionVariants[name] {
		return nil
	}
	adm := &document.Admonition{Variant: strings.ToLower(name)}
	rest := inl[1:]
	if len(rest) > 0 {
		if _, isBreak := rest[0].(*document.SoftBreak); isBreak {
			rest = rest[1:]
		}
	}
	if len(rest) > 0 {
		np := &document.Paragraph{}
		for _, c := range rest {
			np.AppendChild(c)
		}
		adm.AppendChild(np)
	}
	for _, c := range kids[1:] {
		adm.AppendChild(c)
	}
	return adm
}
```

- [ ] **Step 4: Run until PASS.**
- [ ] **Step 5: Commit** — `git commit -am "feat(parser): GitHub-style admonition promotion from blockquotes"`

---

### Task 7: Parser — wiki-links

**Files:**
- Modify: `parser/parser.go` (`extensions()`), `parser/transform.go`
- Test: `parser/wikilink_test.go`

**Interfaces:**
- Consumes: `Config.WikiLinks`.
- Produces: `document.WikiLink{Target}` with label children (label = target text when no `|label`).

- [ ] **Step 1: Dep** — `go get go.abhg.dev/goldmark/wikilink@latest`
- [ ] **Step 2: Failing test** — `parser/wikilink_test.go`:

```go
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
```

- [ ] **Step 3: Run, expect FAIL.**
- [ ] **Step 4: Implement** — `extensions()`: `if cfg.WikiLinks { exts = append(exts, &wikilink.Extender{}) }` (import `go.abhg.dev/goldmark/wikilink`). `convert()` case:

```go
case *wikilink.Node:
	wl := &document.WikiLink{Target: string(n.Target)}
	t.appendChildren(wl, n)
	return wl
```

- [ ] **Step 5: Run until PASS.** (If the extension emits the target as the child text automatically, keep the dump expectations as-is; adjust only if actual child structure differs — verify with `document.Dump` output, do not blindly rewrite.)
- [ ] **Step 6: Commit** — `git commit -am "feat(parser): wiki-link nodes"`

---

### Task 8: Parser — math (`$…$`, `$$…$$`, ```` ```math ````)

**Files:**
- Create: `parser/math.go`
- Modify: `parser/parser.go` (`extensions()`), `parser/transform.go`
- Test: `parser/math_test.go`

**Interfaces:**
- Consumes: `Config.Math`.
- Produces: `document.MathInline{Source, Display}`; `document.MathBlock{Source}`. Promotion rules: paragraph whose only child is a display MathInline → MathBlock; paragraph whose raw lines are `$$ / … / $$` → MathBlock; fence lang `math` → MathBlock (already in Task 2's `codeOrSpecial`).

- [ ] **Step 1: Failing tests** — `parser/math_test.go`:

```go
package parser

import "testing"

func TestInlineMath(t *testing.T) {
	assertDoc(t, "Euler: $e^{i\\pi}+1=0$ done\n", `
Document
  Paragraph
    Text "Euler: "
    MathInline display=false "e^{i\\pi}+1=0"
    Text " done"
`)
}

func TestDisplayMathSingleLine(t *testing.T) {
	assertDoc(t, "$$x^2$$\n", `
Document
  MathBlock "x^2"
`)
}

func TestDisplayMathMultiLine(t *testing.T) {
	assertDoc(t, "$$\nx = y\n$$\n", `
Document
  MathBlock "x = y\n"
`)
}

func TestDollarNotMath(t *testing.T) {
	assertDoc(t, "costs $5 and $10 total\n", `
Document
  Paragraph
    Text "costs $5 and $10 total"
`)
}
```

(Note: `$5 and $10` — the `$` opener is followed by a digit then no valid closer pattern before ` and $`… actually `$5 and $` closes. The guard below requires the character after the opening `$` to be non-space AND the character before the closing `$` to be non-space; `$5 and $` has a space before the closer, so it is rejected. Keep this test; it locks the guard in.)

- [ ] **Step 2: Run, expect FAIL.**
- [ ] **Step 3: Implement `parser/math.go`**

```go
package parser

import (
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	gparser "github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/text"
	"github.com/yuin/goldmark/util"
)

var mathKind = ast.NewNodeKind("Math")

type mathNode struct {
	ast.BaseInline
	Display bool
	Source  []byte
}

func (n *mathNode) Kind() ast.NodeKind { return mathKind }
func (n *mathNode) Dump(src []byte, level int) {
	ast.DumpHelper(n, src, level, nil, nil)
}

type mathExt struct{}

func (mathExt) Extend(md goldmark.Markdown) {
	md.Parser().AddOptions(gparser.WithInlineParsers(
		util.Prioritized(mathParser{}, 150),
	))
}

type mathParser struct{}

func (mathParser) Trigger() []byte { return []byte{'$'} }

// Parse recognizes $x$ and $$x$$ on a single line. Multi-line $$ blocks are
// promoted from paragraphs in the transformer. Rules: no space just inside
// either delimiter; content non-empty.
func (mathParser) Parse(parent ast.Node, block text.Reader, pc gparser.Context) ast.Node {
	line, _ := block.PeekLine()
	delim := 1
	if len(line) >= 2 && line[1] == '$' {
		delim = 2
	}
	rest := line[delim:]
	if len(rest) == 0 || rest[0] == ' ' || rest[0] == '\t' {
		return nil
	}
	end := -1
	for i := 1; i+delim <= len(rest); i++ {
		if rest[i] != '$' {
			continue
		}
		if delim == 2 && (i+1 >= len(rest) || rest[i+1] != '$') {
			continue
		}
		if rest[i-1] == ' ' || rest[i-1] == '\t' {
			continue
		}
		end = i
		break
	}
	if end <= 0 {
		return nil
	}
	src := make([]byte, end)
	copy(src, rest[:end])
	block.Advance(delim + end + delim)
	return &mathNode{Display: delim == 2, Source: src}
}
```

- [ ] **Step 4: Wire up** — `extensions()`: `if cfg.Math { exts = append(exts, mathExt{}) }`. In `convert()`:

```go
case *mathNode:
	return &document.MathInline{Source: string(n.Source), Display: n.Display}
```

Replace the Paragraph case to add promotions:

```go
case *ast.Paragraph:
	if t.cfg.Math {
		if mb := mathBlockFromLines(n, t.src); mb != nil {
			return mb
		}
	}
	p := &document.Paragraph{}
	t.appendChildren(p, n)
	if t.cfg.Math {
		if kids := p.Children(); len(kids) == 1 {
			if mi, ok := kids[0].(*document.MathInline); ok && mi.Display {
				return &document.MathBlock{Source: mi.Source}
			}
		}
	}
	return p
```

New helper:

```go
// mathBlockFromLines promotes a paragraph shaped like
//   $$
//   ...
//   $$
// into a MathBlock.
func mathBlockFromLines(n *ast.Paragraph, src []byte) *document.MathBlock {
	l := n.Lines()
	if l.Len() < 3 {
		return nil
	}
	first := strings.TrimSpace(string(l.At(0).Value(src)))
	last := strings.TrimSpace(string(l.At(l.Len() - 1).Value(src)))
	if first != "$$" || last != "$$" {
		return nil
	}
	var b strings.Builder
	for i := 1; i < l.Len()-1; i++ {
		b.Write(l.At(i).Value(src))
	}
	return &document.MathBlock{Source: b.String()}
}
```

- [ ] **Step 5: Run until PASS.**
- [ ] **Step 6: Commit** — `git commit -am "feat(parser): TeX math inline/display parsing"`

---

### Task 9: HTML renderer core

**Files:**
- Create: `render/html/options.go`, `render/html/renderer.go`, `render/html/url.go`
- Test: `render/html/renderer_test.go`, `render/html/golden_test.go`, fixtures `render/html/testdata/core.md` (+ `.golden.html`)

**Interfaces:**
- Consumes: `document` nodes, `parser.Parse` (tests only).
- Produces (package `htmlrender`, dir `render/html`):
  - `type Options struct { Fragment bool; ThemeName string; Unsafe bool; Highlight bool; Mermaid bool; Math bool; HeadingAnchors bool; Resolver Resolver }`
  - `func DefaultOptions() Options` → `{ThemeName: "auto", Highlight: true, Mermaid: true, Math: true, HeadingAnchors: true}` (Fragment=false, Unsafe=false)
  - `func Render(w io.Writer, doc *document.Document, opts Options) error`
  - `type ResolveKind int` (`ResolveLink`, `ResolveImage`, `ResolveWikiLink`); `type Resolver func(kind ResolveKind, target string) (url string, ok bool)`
  - Full-document mode is a stub until Task 12: when `!Fragment`, for now render fragment (Task 12 replaces this).

- [ ] **Step 1: Write failing unit tests** — `render/html/renderer_test.go`:

```go
package htmlrender

import (
	"bytes"
	"strings"
	"testing"

	"github.com/sriannamalai/markdownviewer/parser"
)

func render(t *testing.T, md string, mutate func(*Options)) string {
	t.Helper()
	doc, err := parser.Parse([]byte(md))
	if err != nil {
		t.Fatal(err)
	}
	opts := DefaultOptions()
	opts.Fragment = true
	opts.Highlight = false
	if mutate != nil {
		mutate(&opts)
	}
	var buf bytes.Buffer
	if err := Render(&buf, doc, opts); err != nil {
		t.Fatal(err)
	}
	return buf.String()
}

func TestCoreBlocks(t *testing.T) {
	got := render(t, "# Title\n\npara *em* **st**\n\n- a\n- b\n", nil)
	want := "<h1 id=\"title\">Title</h1>\n<p>para <em>em</em> <strong>st</strong></p>\n<ul>\n<li>a</li>\n<li>b</li>\n</ul>\n"
	if got != want {
		t.Fatalf("\n got %q\nwant %q", got, want)
	}
}

func TestAnchorsOff(t *testing.T) {
	got := render(t, "# Title\n", func(o *Options) { o.HeadingAnchors = false })
	if got != "<h1>Title</h1>\n" {
		t.Fatalf("got %q", got)
	}
}

func TestEscaping(t *testing.T) {
	got := render(t, "a < b & \"c\"\n", nil)
	if !strings.Contains(got, "a &lt; b &amp; &quot;c&quot;") {
		t.Fatalf("got %q", got)
	}
}

func TestUnsafeURLsStripped(t *testing.T) {
	got := render(t, "[x](javascript:alert(1)) ![y](data:text/html;base64,xx)\n", nil)
	if strings.Contains(got, "javascript:") || strings.Contains(got, "data:") {
		t.Fatalf("unsafe URL survived: %q", got)
	}
}

func TestUnsafeURLsKeptWhenUnsafe(t *testing.T) {
	got := render(t, "[x](javascript:alert(1))\n", func(o *Options) { o.Unsafe = true })
	if !strings.Contains(got, `href="javascript:alert(1)"`) {
		t.Fatalf("got %q", got)
	}
}

func TestWikiLinkDefaultResolution(t *testing.T) {
	got := render(t, "[[Some Page]]\n", nil)
	if !strings.Contains(got, `<a href="Some Page.md" class="wikilink">Some Page</a>`) {
		t.Fatalf("got %q", got)
	}
}

func TestResolverCallback(t *testing.T) {
	got := render(t, "![i](pic.png) [[P]]\n", func(o *Options) {
		o.Resolver = func(kind ResolveKind, target string) (string, bool) {
			return "vfs://" + target, true
		}
	})
	if !strings.Contains(got, `src="vfs://pic.png"`) || !strings.Contains(got, `href="vfs://P"`) {
		t.Fatalf("got %q", got)
	}
}

func TestTaskListCheckboxes(t *testing.T) {
	got := render(t, "- [x] done\n", nil)
	if !strings.Contains(got, `<input type="checkbox" checked disabled>`) {
		t.Fatalf("got %q", got)
	}
}
```

- [ ] **Step 2: Golden framework** — `render/html/golden_test.go`:

```go
package htmlrender

import (
	"bytes"
	"flag"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sriannamalai/markdownviewer/parser"
)

var update = flag.Bool("update", false, "rewrite golden files")

func TestGolden(t *testing.T) {
	mds, err := filepath.Glob("testdata/*.md")
	if err != nil || len(mds) == 0 {
		t.Fatalf("no fixtures: %v", err)
	}
	for _, md := range mds {
		name := strings.TrimSuffix(filepath.Base(md), ".md")
		t.Run(name, func(t *testing.T) {
			src, err := os.ReadFile(md)
			if err != nil {
				t.Fatal(err)
			}
			doc, err := parser.Parse(src)
			if err != nil {
				t.Fatal(err)
			}
			opts := DefaultOptions()
			opts.Fragment = true
			var buf bytes.Buffer
			if err := Render(&buf, doc, opts); err != nil {
				t.Fatal(err)
			}
			golden := filepath.Join("testdata", name+".golden.html")
			if *update {
				if err := os.WriteFile(golden, buf.Bytes(), 0o644); err != nil {
					t.Fatal(err)
				}
				return
			}
			want, err := os.ReadFile(golden)
			if err != nil {
				t.Fatalf("missing golden (run with -update): %v", err)
			}
			if !bytes.Equal(want, buf.Bytes()) {
				t.Fatalf("golden mismatch for %s:\n--- got ---\n%s\n--- want ---\n%s", name, buf.Bytes(), want)
			}
		})
	}
}
```

Fixture `render/html/testdata/core.md` — one document exercising: headings 1–3, paragraph with em/strong/code/strike, ordered+unordered+task lists (tight and loose), blockquote, admonition, table with alignments, fenced code (no highlight assertions yet), thematic break, footnote, definition list, wiki-link, inline + display math, mermaid fence, image, autolink.

- [ ] **Step 3: Run, expect FAIL (compile errors).**

- [ ] **Step 4: Implement `render/html/options.go`**

```go
// Package htmlrender renders the document model to themed HTML.
package htmlrender

type ResolveKind int

const (
	ResolveLink ResolveKind = iota
	ResolveImage
	ResolveWikiLink
)

// Resolver lets hosts rewrite link/image/wiki-link targets. Returning
// ok=false falls back to default handling.
type Resolver func(kind ResolveKind, target string) (url string, ok bool)

type Options struct {
	Fragment       bool   // emit body-only fragment instead of a full page
	ThemeName      string // "light", "dark", "auto"
	Unsafe         bool   // allow raw HTML and all URL schemes
	Highlight      bool   // chroma syntax highlighting
	Mermaid        bool   // mermaid diagram support
	Math           bool   // KaTeX math support
	HeadingAnchors bool   // id attributes on headings
	Resolver       Resolver
}

func DefaultOptions() Options {
	return Options{
		ThemeName: "auto", Highlight: true, Mermaid: true,
		Math: true, HeadingAnchors: true,
	}
}
```

- [ ] **Step 5: Implement `render/html/url.go`**

```go
package htmlrender

import "strings"

var badSchemes = []string{"javascript:", "vbscript:", "data:", "file:"}

// safeURL reports whether a URL is allowed under the default policy.
// Relative URLs and http(s)/mailto pass; script-ish schemes do not.
func safeURL(u string) bool {
	s := strings.ToLower(strings.TrimSpace(u))
	for _, p := range badSchemes {
		if strings.HasPrefix(s, p) {
			return false
		}
	}
	return true
}
```

- [ ] **Step 6: Implement `render/html/renderer.go`** — the core walk. Structure:

```go
package htmlrender

import (
	"bufio"
	"fmt"
	"html"
	"io"
	"strings"

	"github.com/sriannamalai/markdownviewer/document"
)

func Render(w io.Writer, doc *document.Document, opts Options) error {
	// Full-page mode arrives in Task 12; render fragment for now.
	return renderFragment(w, doc, opts)
}

func renderFragment(w io.Writer, doc *document.Document, opts Options) error {
	bw := bufio.NewWriter(w)
	r := &writer{w: bw, opts: opts}
	for _, c := range doc.Children() {
		r.block(c, false)
	}
	r.footnotes(doc)
	if r.err != nil {
		return r.err
	}
	return bw.Flush()
}

type writer struct {
	w    *bufio.Writer
	opts Options
	err  error
}

func (r *writer) raw(s string) {
	if r.err == nil {
		_, r.err = r.w.WriteString(s)
	}
}
func (r *writer) text(s string) { r.raw(html.EscapeString(s)) }
func esc(s string) string       { return html.EscapeString(s) }

func (r *writer) resolve(kind ResolveKind, dest string) string {
	if r.opts.Resolver != nil {
		if u, ok := r.opts.Resolver(kind, dest); ok {
			return u
		}
	}
	if kind == ResolveWikiLink {
		return dest + ".md"
	}
	return dest
}

// href returns an escaped, policy-checked attribute value ("" if blocked).
func (r *writer) href(kind ResolveKind, dest string) string {
	u := r.resolve(kind, dest)
	if !r.opts.Unsafe && !safeURL(u) {
		return ""
	}
	return esc(u)
}
```

Block dispatch (`tight` is true when the direct parent is a tight list item):

```go
func (r *writer) block(n document.Node, tight bool) {
	switch n := n.(type) {
	case *document.Heading:
		tag := fmt.Sprintf("h%d", n.Level)
		if r.opts.HeadingAnchors && n.AnchorID != "" {
			r.raw("<" + tag + ` id="` + esc(n.AnchorID) + `">`)
		} else {
			r.raw("<" + tag + ">")
		}
		r.inlines(n)
		r.raw("</" + tag + ">\n")
	case *document.Paragraph:
		if tight {
			r.inlines(n)
			return
		}
		r.raw("<p>")
		r.inlines(n)
		r.raw("</p>\n")
	case *document.BlockQuote:
		r.raw("<blockquote>\n")
		r.blocks(n, false)
		r.raw("</blockquote>\n")
	case *document.Admonition:
		title := strings.ToUpper(n.Variant[:1]) + n.Variant[1:]
		r.raw(`<div class="admonition admonition-` + n.Variant + "\">\n")
		r.raw(`<p class="admonition-title">` + title + "</p>\n")
		r.blocks(n, false)
		r.raw("</div>\n")
	case *document.List:
		tag, attrs := "ul", ""
		if n.Ordered {
			tag = "ol"
			if n.Start != 1 {
				attrs = fmt.Sprintf(` start="%d"`, n.Start)
			}
		}
		r.raw("<" + tag + attrs + ">\n")
		for _, li := range n.Children() {
			r.listItem(li.(*document.ListItem), n.Tight)
		}
		r.raw("</" + tag + ">\n")
	case *document.CodeBlock:
		r.codeBlock(n) // plain now; chroma path added in Task 10
	case *document.Diagram:
		if r.opts.Mermaid && n.Engine == "mermaid" {
			r.raw(`<pre class="mermaid">` + esc(n.Source) + "</pre>\n")
		} else {
			r.codeBlock(&document.CodeBlock{Language: n.Engine, Code: n.Source})
		}
	case *document.MathBlock:
		if r.opts.Math {
			r.raw(`<div class="math math-display">` + esc(n.Source) + "</div>\n")
		} else {
			r.codeBlock(&document.CodeBlock{Language: "math", Code: n.Source})
		}
	case *document.Table:
		r.table(n)
	case *document.ThematicBreak:
		r.raw("<hr>\n")
	case *document.HTMLBlock:
		r.rawHTML(n.HTML, true)
	case *document.DefinitionList:
		r.raw("<dl>\n")
		for _, c := range n.Children() {
			switch c := c.(type) {
			case *document.DefinitionTerm:
				r.raw("<dt>")
				r.inlines(c)
				r.raw("</dt>\n")
			case *document.DefinitionDesc:
				r.raw("<dd>\n")
				r.blocks(c, false)
				r.raw("</dd>\n")
			}
		}
		r.raw("</dl>\n")
	default:
		r.inline(n) // stray inline at block level
	}
}

func (r *writer) blocks(n document.Node, tight bool) {
	for _, c := range n.Children() {
		r.block(c, tight)
	}
}
func (r *writer) inlines(n document.Node) {
	for _, c := range n.Children() {
		r.inline(c)
	}
}
```

List items (CommonMark reference formatting):

```go
func (r *writer) listItem(li *document.ListItem, tight bool) {
	kids := li.Children()
	if tight {
		r.raw("<li>")
		if li.Task {
			r.checkbox(li.Checked)
		}
		for i, c := range kids {
			if p, ok := c.(*document.Paragraph); ok && i == 0 {
				r.inlines(p)
				continue
			}
			if i == 0 {
				r.raw("\n")
			}
			r.block(c, false)
		}
		r.raw("</li>\n")
		return
	}
	r.raw("<li>\n")
	if li.Task {
		r.checkbox(li.Checked)
	}
	r.blocks(li, false)
	r.raw("</li>\n")
}

func (r *writer) checkbox(checked bool) {
	if checked {
		r.raw(`<input type="checkbox" checked disabled> `)
	} else {
		r.raw(`<input type="checkbox" disabled> `)
	}
}
```

Code blocks, tables, footnotes:

```go
func (r *writer) codeBlock(n *document.CodeBlock) {
	cls := ""
	if n.Language != "" {
		cls = ` class="language-` + esc(n.Language) + `"`
	}
	r.raw("<pre><code" + cls + ">")
	r.text(n.Code)
	r.raw("</code></pre>\n")
}

var alignAttr = map[document.Alignment]string{
	document.AlignLeft: ` align="left"`, document.AlignCenter: ` align="center"`,
	document.AlignRight: ` align="right"`,
}

func (r *writer) table(t *document.Table) {
	r.raw("<table>\n")
	rows := t.Children()
	for ri, rowNode := range rows {
		row := rowNode.(*document.TableRow)
		if row.Header {
			r.raw("<thead>\n")
		} else if ri == 1 {
			r.raw("<tbody>\n")
		}
		r.raw("<tr>\n")
		tag := "td"
		if row.Header {
			tag = "th"
		}
		for ci, cell := range row.Children() {
			attr := ""
			if ci < len(t.Alignments) {
				attr = alignAttr[t.Alignments[ci]]
			}
			r.raw("<" + tag + attr + ">")
			r.inlines(cell)
			r.raw("</" + tag + ">\n")
		}
		r.raw("</tr>\n")
		if row.Header {
			r.raw("</thead>\n")
		}
	}
	if len(rows) > 1 {
		r.raw("</tbody>\n")
	}
	r.raw("</table>\n")
}

func (r *writer) footnotes(doc *document.Document) {
	if len(doc.Footnotes) == 0 {
		return
	}
	r.raw("<section class=\"footnotes\">\n<ol>\n")
	for _, def := range doc.Footnotes {
		r.raw(fmt.Sprintf(`<li id="fn:%d">`+"\n", def.Index))
		r.blocks(def, false)
		r.raw(fmt.Sprintf(`<a href="#fnref:%d" class="footnote-backref">↩</a>`+"\n</li>\n", def.Index))
	}
	r.raw("</ol>\n</section>\n")
}
```

Inline dispatch:

```go
func (r *writer) inline(n document.Node) {
	switch n := n.(type) {
	case *document.Text:
		r.text(n.Value)
	case *document.SoftBreak:
		r.raw("\n")
	case *document.HardBreak:
		r.raw("<br>\n")
	case *document.Emphasis:
		r.raw("<em>")
		r.inlines(n)
		r.raw("</em>")
	case *document.Strong:
		r.raw("<strong>")
		r.inlines(n)
		r.raw("</strong>")
	case *document.Strikethrough:
		r.raw("<del>")
		r.inlines(n)
		r.raw("</del>")
	case *document.CodeSpan:
		r.raw("<code>")
		r.text(n.Value)
		r.raw("</code>")
	case *document.Link:
		if u := r.href(ResolveLink, n.Destination); u != "" {
			t := ""
			if n.Title != "" {
				t = ` title="` + esc(n.Title) + `"`
			}
			r.raw(`<a href="` + u + `"` + t + ">")
		} else {
			r.raw("<a>")
		}
		r.inlines(n)
		r.raw("</a>")
	case *document.Image:
		u := r.href(ResolveImage, n.Destination)
		t := ""
		if n.Title != "" {
			t = ` title="` + esc(n.Title) + `"`
		}
		r.raw(`<img src="` + u + `" alt="` + esc(n.Alt) + `"` + t + ">")
	case *document.WikiLink:
		u := r.href(ResolveWikiLink, n.Target)
		r.raw(`<a href="` + u + `" class="wikilink">`)
		r.inlines(n)
		r.raw("</a>")
	case *document.MathInline:
		if r.opts.Math {
			cls := "math math-inline"
			if n.Display {
				cls = "math math-display"
			}
			r.raw(`<span class="` + cls + `">` + esc(n.Source) + "</span>")
		} else {
			r.raw("<code>")
			r.text(n.Source)
			r.raw("</code>")
		}
	case *document.HTMLInline:
		r.rawHTML(n.HTML, false)
	case *document.FootnoteRef:
		r.raw(fmt.Sprintf(`<sup id="fnref:%d"><a href="#fn:%d">%d</a></sup>`, n.Index, n.Index, n.Index))
	}
}

// rawHTML passes raw markdown HTML through; sanitization lands in Task 11.
func (r *writer) rawHTML(s string, block bool) {
	r.raw(s) // Task 11 replaces this body
	if block && !strings.HasSuffix(s, "\n") {
		r.raw("\n")
	}
}
```

- [ ] **Step 7: Run unit tests until PASS** — `go test ./render/html -run 'Test[^G]' -v`
- [ ] **Step 8: Generate goldens** — `go test ./render/html -run TestGolden -update`, then **read `core.golden.html` end-to-end** and verify every construct renders sanely (checkboxes, admonition div, table aligns, mermaid pre, math spans, footnote section). Re-run without `-update`: PASS.
- [ ] **Step 9: Commit**

```bash
git add render/
git commit -m "feat(render/html): core HTML fragment renderer with URL policy and resolver"
```

---

### Task 10: Syntax highlighting (chroma)

**Files:**
- Create: `render/html/highlight.go`
- Modify: `render/html/renderer.go` (`codeBlock`)
- Test: `render/html/highlight_test.go`, fixture `render/html/testdata/code.md`

**Interfaces:**
- Consumes: `Options.Highlight`.
- Produces: `func chromaCSS(styleName string) (string, error)` (used by Task 12's page assembly); highlighted `<pre class="chroma">…` output for known languages; plain `codeBlock` fallback for unknown.

- [ ] **Step 1: Dep** — `go get github.com/alecthomas/chroma/v2@latest`
- [ ] **Step 2: Failing test** — `render/html/highlight_test.go`:

```go
package htmlrender

import (
	"strings"
	"testing"
)

func TestHighlightGo(t *testing.T) {
	got := render(t, "```go\nfmt.Println(\"hi\")\n```\n", func(o *Options) { o.Highlight = true })
	if !strings.Contains(got, `class="chroma"`) || !strings.Contains(got, "<span") {
		t.Fatalf("not highlighted: %q", got)
	}
	if strings.Contains(got, "style=") {
		t.Fatalf("inline styles leaked (classes mode expected): %q", got)
	}
}

func TestHighlightUnknownLangFallsBack(t *testing.T) {
	got := render(t, "```nosuchlang\nzzz\n```\n", func(o *Options) { o.Highlight = true })
	if !strings.Contains(got, `<pre><code class="language-nosuchlang">zzz`) {
		t.Fatalf("fallback broken: %q", got)
	}
}

func TestChromaCSS(t *testing.T) {
	css, err := chromaCSS("github")
	if err != nil || !strings.Contains(css, ".chroma") {
		t.Fatalf("css: %v %q", err, css)
	}
}
```

- [ ] **Step 3: Run, expect FAIL.**
- [ ] **Step 4: Implement `render/html/highlight.go`**

```go
package htmlrender

import (
	"io"
	"strings"

	"github.com/alecthomas/chroma/v2"
	chromahtml "github.com/alecthomas/chroma/v2/formatters/html"
	"github.com/alecthomas/chroma/v2/lexers"
	"github.com/alecthomas/chroma/v2/styles"
)

var chromaFormatter = chromahtml.New(chromahtml.WithClasses(true))

// highlight writes chroma-highlighted HTML for code; returns false when the
// language is unknown so the caller can fall back to plain rendering.
func highlight(w io.Writer, code, lang string) bool {
	if lang == "" {
		return false
	}
	lexer := lexers.Get(lang)
	if lexer == nil {
		return false
	}
	lexer = chroma.Coalesce(lexer)
	it, err := lexer.Tokenise(nil, code)
	if err != nil {
		return false
	}
	return chromaFormatter.Format(w, styles.Get("github"), it) == nil
}

// chromaCSS returns class-based CSS for a chroma style ("github",
// "github-dark"). Used by full-page assembly.
func chromaCSS(styleName string) (string, error) {
	style := styles.Get(styleName)
	var b strings.Builder
	if err := chromaFormatter.WriteCSS(&b, style); err != nil {
		return "", err
	}
	return b.String(), nil
}
```

Modify `codeBlock`:

```go
func (r *writer) codeBlock(n *document.CodeBlock) {
	if r.opts.Highlight {
		var b strings.Builder
		if highlight(&b, n.Code, n.Language) {
			r.raw(b.String())
			if !strings.HasSuffix(b.String(), "\n") {
				r.raw("\n")
			}
			return
		}
	}
	cls := ""
	if n.Language != "" {
		cls = ` class="language-` + esc(n.Language) + `"`
	}
	r.raw("<pre><code" + cls + ">")
	r.text(n.Code)
	r.raw("</code></pre>\n")
}
```

- [ ] **Step 5: Run until PASS.** Add fixture `testdata/code.md` (Go + Python + unknown-lang fences), regenerate goldens with `-update`, eyeball, re-run.
- [ ] **Step 6: Commit** — `git add render/ go.mod go.sum && git commit -m "feat(render/html): chroma syntax highlighting with class-based CSS"`

---

### Task 11: Raw-HTML sanitization

**Files:**
- Create: `render/html/sanitize.go`, `render/html/testdata/xss.txt`
- Modify: `render/html/renderer.go` (`rawHTML`)
- Test: `render/html/sanitize_test.go`

**Interfaces:**
- Consumes: `Options.Unsafe`.
- Produces: default-path raw HTML runs through bluemonday UGC policy; `Unsafe` passes through verbatim.

- [ ] **Step 1: Dep** — `go get github.com/microcosm-cc/bluemonday@latest`
- [ ] **Step 2: Failing tests** — `render/html/sanitize_test.go`:

```go
package htmlrender

import (
	"bufio"
	"os"
	"strings"
	"testing"
)

func TestScriptStripped(t *testing.T) {
	got := render(t, "<script>alert(1)</script>\n\nsafe <b>bold</b>\n", nil)
	if strings.Contains(got, "<script") {
		t.Fatalf("script survived: %q", got)
	}
	if !strings.Contains(got, "<b>bold</b>") {
		t.Fatalf("benign HTML over-stripped: %q", got)
	}
}

func TestUnsafePassthrough(t *testing.T) {
	got := render(t, "<script>alert(1)</script>\n", func(o *Options) { o.Unsafe = true })
	if !strings.Contains(got, "<script>alert(1)</script>") {
		t.Fatalf("unsafe mode should pass through: %q", got)
	}
}

// xss.txt: one hostile markdown snippet per line (\n escapes as \n).
func TestXSSCorpus(t *testing.T) {
	f, err := os.Open("testdata/xss.txt")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := strings.ReplaceAll(sc.Text(), `\n`, "\n")
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		got := strings.ToLower(render(t, line, nil))
		for _, bad := range []string{"<script", "javascript:", "onerror=", "onload=", "srcdoc="} {
			if strings.Contains(got, bad) {
				t.Errorf("payload %q leaked %q into output %q", line, bad, got)
			}
		}
	}
}
```

`render/html/testdata/xss.txt`:

```
# classic vectors — one markdown snippet per line
<script>alert(1)</script>
<img src=x onerror=alert(1)>
<svg onload=alert(1)>
[click](javascript:alert(1))
[click](JaVaScRiPt:alert(1))
![img](javascript:alert(1))
<iframe srcdoc="<script>alert(1)</script>"></iframe>
<a href="vbscript:msgbox(1)">x</a>
<details open ontoggle=alert(1)>
<body onload=alert(1)>
[x]: javascript:alert(1)\n\n[x]
<math><mtext></form><form><mglyph><svg><mtext><style><path id="</style><img onerror=alert(1) src>">
```

- [ ] **Step 3: Run, expect FAIL** (script currently passes through).
- [ ] **Step 4: Implement `render/html/sanitize.go`**

```go
package htmlrender

import "github.com/microcosm-cc/bluemonday"

// sanitizePolicy is the default policy for raw HTML embedded in markdown:
// bluemonday's UGC policy (basic formatting, links, images; no scripts,
// event handlers, or iframes).
var sanitizePolicy = bluemonday.UGCPolicy()
```

Replace `rawHTML`:

```go
func (r *writer) rawHTML(s string, block bool) {
	out := s
	if !r.opts.Unsafe {
		out = sanitizePolicy.Sanitize(s)
	}
	r.raw(out)
	if block && !strings.HasSuffix(out, "\n") {
		r.raw("\n")
	}
}
```

- [ ] **Step 5: Run until PASS** (`go test ./render/html -v`). Note: sanitizing inline HTML tag-by-tag means unmatched `<em>`/`</em>` fragments may each sanitize independently — if bluemonday drops lone closing tags and a corpus/unit test fails on benign inline HTML, sanitize inline runs as-is and accept that bluemonday balances tags; adjust the `TestScriptStripped` benign assertion to whatever correct sanitized form appears, as long as bold text content survives.
- [ ] **Step 6: Regenerate goldens if raw-HTML fixtures changed; eyeball; commit** — `git add render/ go.mod go.sum && git commit -m "feat(render/html): sanitize raw HTML by default with XSS corpus tests"`

---

### Task 12: Theme package + full-document page assembly

**Files:**
- Create: `theme/theme.go`, `theme/base.css`, `render/html/page.go`
- Test: `theme/theme_test.go`, `render/html/page_test.go`

**Interfaces:**
- Consumes: `chromaCSS` (Task 10), `Options.Fragment/ThemeName`.
- Produces:
  - `type Theme struct { Name string; Vars map[string]string; ChromaStyle string }`
  - `func Light() Theme`, `func Dark() Theme`, `func Get(name string) (Theme, error)` (`"light"|"dark"|"auto"` — auto returns Light; page assembly adds the dark media query), `func BaseCSS() string`, `func (t Theme) CSS(selector string) string`
  - `Render` with `Fragment=false` emits a complete standalone HTML page (doctype, meta, inlined CSS; JS wiring in Task 13).

- [ ] **Step 1: Failing tests** — `theme/theme_test.go`:

```go
package theme

import (
	"strings"
	"testing"
)

func TestThemesComplete(t *testing.T) {
	required := []string{"--md-bg", "--md-fg", "--md-accent", "--md-code-bg", "--md-border", "--md-quote-fg"}
	for _, th := range []Theme{Light(), Dark()} {
		for _, v := range required {
			if _, ok := th.Vars[v]; !ok {
				t.Errorf("theme %s missing %s", th.Name, v)
			}
		}
	}
}

func TestCSSEmission(t *testing.T) {
	css := Light().CSS(":root")
	if !strings.HasPrefix(css, ":root{") || !strings.Contains(css, "--md-bg:") {
		t.Fatalf("css: %q", css)
	}
}

func TestBaseCSSUsesVars(t *testing.T) {
	if !strings.Contains(BaseCSS(), "var(--md-bg)") {
		t.Fatal("base.css must consume theme variables")
	}
}

func TestGet(t *testing.T) {
	for _, name := range []string{"light", "dark", "auto"} {
		if _, err := Get(name); err != nil {
			t.Errorf("Get(%q): %v", name, err)
		}
	}
	if _, err := Get("neon"); err == nil {
		t.Error("Get(neon) should fail")
	}
}
```

`render/html/page_test.go`:

```go
package htmlrender

import (
	"strings"
	"testing"
)

func TestFullPage(t *testing.T) {
	got := render(t, "# Hi\n", func(o *Options) { o.Fragment = false; o.ThemeName = "auto" })
	for _, want := range []string{
		"<!doctype html>", "<meta charset=\"utf-8\">", "<style>",
		"markdown-body", "--md-bg:", "@media (prefers-color-scheme: dark)",
		"<h1 id=\"hi\">Hi</h1>",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("page missing %q", want)
		}
	}
	if strings.Contains(got, "http://") || strings.Contains(got, "https://cdn") {
		t.Error("page must not reference the network")
	}
}

func TestFullPageLightHasNoDarkBlock(t *testing.T) {
	got := render(t, "# Hi\n", func(o *Options) { o.Fragment = false; o.ThemeName = "light" })
	if strings.Contains(got, "prefers-color-scheme") {
		t.Error("explicit light theme should not embed dark override")
	}
}
```

- [ ] **Step 2: Run, expect FAIL.**
- [ ] **Step 3: Implement `theme/theme.go`**

```go
// Package theme provides the built-in visual themes as CSS custom-property
// sets over one base stylesheet.
package theme

import (
	_ "embed"
	"fmt"
	"sort"
	"strings"
)

//go:embed base.css
var baseCSS string

func BaseCSS() string { return baseCSS }

type Theme struct {
	Name        string
	Vars        map[string]string
	ChromaStyle string // paired chroma syntax-highlight style
}

func Light() Theme {
	return Theme{Name: "light", ChromaStyle: "github", Vars: map[string]string{
		"--md-bg": "#ffffff", "--md-fg": "#1f2328", "--md-accent": "#0969da",
		"--md-code-bg": "#f6f8fa", "--md-border": "#d1d9e0", "--md-quote-fg": "#59636e",
	}}
}

func Dark() Theme {
	return Theme{Name: "dark", ChromaStyle: "github-dark", Vars: map[string]string{
		"--md-bg": "#0d1117", "--md-fg": "#e6edf3", "--md-accent": "#4493f8",
		"--md-code-bg": "#161b22", "--md-border": "#30363d", "--md-quote-fg": "#8d96a0",
	}}
}

// Get resolves a theme name. "auto" returns Light; page assembly layers the
// dark override in a prefers-color-scheme media query.
func Get(name string) (Theme, error) {
	switch name {
	case "light", "auto", "":
		return Light(), nil
	case "dark":
		return Dark(), nil
	}
	return Theme{}, fmt.Errorf("theme: unknown theme %q", name)
}

// CSS emits the variable declarations under the given selector.
func (t Theme) CSS(selector string) string {
	keys := make([]string, 0, len(t.Vars))
	for k := range t.Vars {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	var b strings.Builder
	b.WriteString(selector + "{")
	for _, k := range keys {
		b.WriteString(k + ":" + t.Vars[k] + ";")
	}
	b.WriteString("}")
	return b.String()
}
```

- [ ] **Step 4: Write `theme/base.css`** — a real stylesheet consuming the variables. Required coverage (write it fully; ~120 lines): `body.markdown-body` (bg/fg/font-stack/line-height/max-width 860px/margin auto/padding), headings with border-bottom on h1/h2, links via `--md-accent`, `code`/`pre` via `--md-code-bg` (pre: overflow-x auto, border-radius), `blockquote` (left border `--md-border`, color `--md-quote-fg`), tables (collapsed borders, th bg `--md-code-bg`, cell padding, borders `--md-border`), `hr`, images `max-width:100%`, task-list checkboxes margin, `.admonition` (left accent border, padding; per-variant accent colors: note `#0969da`, tip `#1a7f37`, important `#8250df`, warning `#9a6700`, caution `#cf222e`; `.admonition-title` bold), `.footnotes` (smaller font, top border), `.wikilink` (dotted underline), `.math-display` (centered, margin), `.mermaid` (centered), `dl/dt/dd` (dt bold, dd indented). Every color must come from a `var(--md-*)` or the fixed admonition accents.

- [ ] **Step 5: Implement `render/html/page.go`**

```go
package htmlrender

import (
	"fmt"
	"io"

	"github.com/sriannamalai/markdownviewer/document"
	"github.com/sriannamalai/markdownviewer/theme"
)

func renderPage(w io.Writer, doc *document.Document, opts Options) error {
	th, err := theme.Get(opts.ThemeName)
	if err != nil {
		return err
	}
	lightChroma, err := chromaCSS(theme.Light().ChromaStyle)
	if err != nil {
		return err
	}
	fmt.Fprint(w, "<!doctype html>\n<html>\n<head>\n<meta charset=\"utf-8\">\n")
	fmt.Fprint(w, "<meta name=\"viewport\" content=\"width=device-width, initial-scale=1\">\n<style>\n")
	if opts.ThemeName == "dark" {
		darkChroma, err := chromaCSS(theme.Dark().ChromaStyle)
		if err != nil {
			return err
		}
		fmt.Fprint(w, theme.Dark().CSS(":root")+"\n"+darkChroma)
	} else {
		fmt.Fprint(w, th.CSS(":root")+"\n"+lightChroma)
	}
	if opts.ThemeName == "auto" || opts.ThemeName == "" {
		darkChroma, err := chromaCSS(theme.Dark().ChromaStyle)
		if err != nil {
			return err
		}
		fmt.Fprint(w, "\n@media (prefers-color-scheme: dark){\n"+theme.Dark().CSS(":root")+"\n"+darkChroma+"}\n")
	}
	fmt.Fprint(w, theme.BaseCSS())
	fmt.Fprint(w, "</style>\n</head>\n<body class=\"markdown-body\">\n")
	if err := renderFragment(w, doc, opts); err != nil {
		return err
	}
	// JS assets (mermaid/KaTeX) are appended here by Task 13.
	fmt.Fprint(w, "</body>\n</html>\n")
	return nil
}
```

And switch `Render`:

```go
func Render(w io.Writer, doc *document.Document, opts Options) error {
	if opts.Fragment {
		return renderFragment(w, doc, opts)
	}
	return renderPage(w, doc, opts)
}
```

- [ ] **Step 6: Run until PASS** — `go test ./theme ./render/html -v`
- [ ] **Step 7: Visual smoke check** — render `render/html/testdata/core.md` to a temp full page and open it:

```bash
go run ./cmd/mdview 2>/dev/null || true  # CLI doesn't exist yet; use a scratch main
cat > /tmp/smoke.go <<'EOF'
package main

import (
	"os"

	htmlrender "github.com/sriannamalai/markdownviewer/render/html"
	"github.com/sriannamalai/markdownviewer/parser"
)

func main() {
	src, _ := os.ReadFile("render/html/testdata/core.md")
	doc, _ := parser.Parse(src)
	f, _ := os.Create("/tmp/smoke.html")
	defer f.Close()
	_ = htmlrender.Render(f, doc, htmlrender.DefaultOptions())
}
EOF
go run /tmp/smoke.go && open /tmp/smoke.html
```

Verify: readable typography, both light/dark (toggle OS theme or use dev tools emulation), highlighted code matches theme, admonitions colored.

- [ ] **Step 8: Commit** — `git add theme/ render/ && git commit -m "feat(theme,render/html): themed standalone page output with light/dark/auto"`

---

### Task 13: Embedded mermaid + KaTeX assets

**Files:**
- Create: `scripts/fetch-assets.sh`, `scripts/inlinefonts/main.go`, `internal/assets/embed.go`, `third_party/README.md`
- Modify: `render/html/page.go`
- Test: `render/html/assets_test.go`

**Interfaces:**
- Consumes: `Options.Mermaid/Math`, page assembly from Task 12.
- Produces: `internal/assets` exposes `MermaidJS()`, `KatexJS()`, `KatexCSS() string`; pages embed JS/CSS **only when the document actually contains** Diagram/Math nodes.

- [ ] **Step 1: Write `scripts/fetch-assets.sh`**

```bash
#!/usr/bin/env bash
# Fetches pinned viewer assets into internal/assets/ and license texts into
# third_party/. Run manually on upgrades; outputs are committed.
set -euo pipefail
cd "$(dirname "$0")/.."

MERMAID_VERSION=11.4.1
KATEX_VERSION=0.16.21
CDN=https://cdn.jsdelivr.net/npm

mkdir -p internal/assets/raw third_party/mermaid third_party/katex

curl -fsSL "$CDN/mermaid@${MERMAID_VERSION}/dist/mermaid.min.js" -o internal/assets/mermaid.min.js
curl -fsSL "$CDN/mermaid@${MERMAID_VERSION}/LICENSE" -o third_party/mermaid/LICENSE

curl -fsSL "$CDN/katex@${KATEX_VERSION}/dist/katex.min.js" -o internal/assets/katex.min.js
curl -fsSL "$CDN/katex@${KATEX_VERSION}/dist/katex.min.css" -o internal/assets/raw/katex.min.css
curl -fsSL "$CDN/katex@${KATEX_VERSION}/LICENSE" -o third_party/katex/LICENSE

fonts=(
  KaTeX_AMS-Regular KaTeX_Caligraphic-Bold KaTeX_Caligraphic-Regular
  KaTeX_Fraktur-Bold KaTeX_Fraktur-Regular KaTeX_Main-Bold
  KaTeX_Main-BoldItalic KaTeX_Main-Italic KaTeX_Main-Regular
  KaTeX_Math-BoldItalic KaTeX_Math-Italic KaTeX_SansSerif-Bold
  KaTeX_SansSerif-Italic KaTeX_SansSerif-Regular KaTeX_Script-Regular
  KaTeX_Size1-Regular KaTeX_Size2-Regular KaTeX_Size3-Regular
  KaTeX_Size4-Regular KaTeX_Typewriter-Regular
)
mkdir -p internal/assets/raw/fonts
for f in "${fonts[@]}"; do
  curl -fsSL "$CDN/katex@${KATEX_VERSION}/dist/fonts/${f}.woff2" -o "internal/assets/raw/fonts/${f}.woff2"
done

go run ./scripts/inlinefonts internal/assets/raw/katex.min.css internal/assets/raw/fonts internal/assets/katex.inline.css
rm -rf internal/assets/raw
echo "mermaid ${MERMAID_VERSION} + katex ${KATEX_VERSION} fetched and inlined."
```

- [ ] **Step 2: Write `scripts/inlinefonts/main.go`**

```go
// Command inlinefonts rewrites KaTeX's CSS to carry its woff2 fonts as
// data: URIs so rendered pages are fully self-contained.
package main

import (
	"encoding/base64"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"regexp"
)

func main() {
	if len(os.Args) != 4 {
		log.Fatal("usage: inlinefonts <katex.min.css> <fontsdir> <out.css>")
	}
	css, err := os.ReadFile(os.Args[1])
	if err != nil {
		log.Fatal(err)
	}
	// Drop non-woff2 fallback sources.
	drop := regexp.MustCompile(`,\s*url\(fonts/[^)]+\.(woff|ttf)\)\s*format\((?:'|")?(?:woff|truetype)(?:'|")?\)`)
	out := drop.ReplaceAll(css, nil)
	// Inline woff2 references.
	woff2 := regexp.MustCompile(`url\(fonts/([^)]+\.woff2)\)`)
	out = woff2.ReplaceAllFunc(out, func(m []byte) []byte {
		name := woff2.FindSubmatch(m)[1]
		data, err := os.ReadFile(filepath.Join(os.Args[2], string(name)))
		if err != nil {
			log.Fatalf("font %s: %v", name, err)
		}
		return []byte(fmt.Sprintf("url(data:font/woff2;base64,%s)", base64.StdEncoding.EncodeToString(data)))
	})
	if err := os.WriteFile(os.Args[3], out, 0o644); err != nil {
		log.Fatal(err)
	}
}
```

- [ ] **Step 3: Run the fetch** — `chmod +x scripts/fetch-assets.sh && ./scripts/fetch-assets.sh`. Verify: `internal/assets/{mermaid.min.js,katex.min.js,katex.inline.css}` exist and `grep -c "url(fonts/" internal/assets/katex.inline.css` prints `0`.

- [ ] **Step 4: Implement `internal/assets/embed.go`**

```go
// Package assets embeds the vendored third-party viewer assets. See
// third_party/README.md for versions and licenses.
package assets

import _ "embed"

//go:embed mermaid.min.js
var mermaidJS string

//go:embed katex.min.js
var katexJS string

//go:embed katex.inline.css
var katexCSS string

func MermaidJS() string { return mermaidJS }
func KatexJS() string   { return katexJS }
func KatexCSS() string  { return katexCSS }
```

- [ ] **Step 5: Failing test** — `render/html/assets_test.go`:

```go
package htmlrender

import (
	"strings"
	"testing"
)

func TestMermaidEmbeddedOnlyWhenUsed(t *testing.T) {
	with := render(t, "```mermaid\ngraph TD; A-->B\n```\n", func(o *Options) { o.Fragment = false })
	if !strings.Contains(with, "mermaid.initialize") {
		t.Error("mermaid page missing init")
	}
	without := render(t, "plain text\n", func(o *Options) { o.Fragment = false })
	if strings.Contains(without, "mermaid.initialize") {
		t.Error("mermaid embedded without diagrams")
	}
}

func TestKatexEmbeddedOnlyWhenUsed(t *testing.T) {
	with := render(t, "$x^2$\n", func(o *Options) { o.Fragment = false })
	if !strings.Contains(with, "katex.render") {
		t.Error("math page missing katex init")
	}
	without := render(t, "plain\n", func(o *Options) { o.Fragment = false })
	if strings.Contains(without, "katex") {
		t.Error("katex embedded without math")
	}
}
```

- [ ] **Step 6: Wire into `render/html/page.go`** — before `</body>`:

```go
import "github.com/sriannamalai/markdownviewer/internal/assets"

func usesFeatures(doc *document.Document) (mermaid, math bool) {
	document.Walk(doc, func(n document.Node, entering bool) document.WalkStatus {
		if !entering {
			return document.Continue
		}
		switch n.(type) {
		case *document.Diagram:
			mermaid = true
		case *document.MathBlock, *document.MathInline:
			math = true
		}
		return document.Continue
	})
	return mermaid, math
}
```

In `renderPage`, after the fragment (mermaid theme follows the page theme; KaTeX CSS goes into the `<style>` block in `<head>`, so compute `usesFeatures` before writing the head):

```go
mermaidUsed, mathUsed := usesFeatures(doc)
// in <style>: if mathUsed && opts.Math → fmt.Fprint(w, assets.KatexCSS())
// before </body>:
if mermaidUsed && opts.Mermaid {
	mtheme := "default"
	if opts.ThemeName == "dark" {
		mtheme = "dark"
	}
	fmt.Fprint(w, "<script>"+assets.MermaidJS()+"</script>\n")
	fmt.Fprintf(w, "<script>mermaid.initialize({startOnLoad:true,theme:%q});</script>\n", mtheme)
}
if mathUsed && opts.Math {
	fmt.Fprint(w, "<script>"+assets.KatexJS()+"</script>\n")
	fmt.Fprint(w, "<script>document.querySelectorAll('.math').forEach(function(el){"+
		"katex.render(el.textContent,el,{displayMode:el.classList.contains('math-display'),throwOnError:false});});</script>\n")
}
```

- [ ] **Step 7: Run until PASS**, then repeat the Task 12 visual smoke check with a mermaid diagram and math in the fixture — the diagram must draw and the math must typeset offline (disconnect network or check devtools: zero network requests).
- [ ] **Step 8: Write `third_party/README.md`** — a table: component, version, license, upstream URL, local license path (mermaid 11.4.1 MIT, KaTeX 0.16.21 MIT, and note goldmark/chroma/bluemonday as Go module deps with their licenses).
- [ ] **Step 9: Commit** — `git add scripts/ internal/ third_party/ render/ && git commit -m "feat(assets): embed pinned mermaid and KaTeX, wired conditionally into pages"`

---

### Task 14: Facade API (`mdviewer.go`)

**Files:**
- Create: `mdviewer.go`
- Test: `mdviewer_test.go`

**Interfaces:**
- Consumes: `parser`, `render/html`, `document`.
- Produces (package `markdownviewer` — the import root):
  - `func Parse(src []byte) (*document.Document, error)`
  - `func Render(src []byte, opts ...Option) ([]byte, error)`
  - `func RenderTo(w io.Writer, src []byte, opts ...Option) error`
  - Options: `WithTheme(name string)`, `Fragment()`, `AllowRawHTML()`, `DisableMermaid()`, `DisableMath()`, `DisableHighlighting()`, `WithResolver(Resolver)`
  - Re-exports: `type Resolver = htmlrender.Resolver`, `ResolveLink/ResolveImage/ResolveWikiLink` consts.

- [ ] **Step 1: Failing test** — `mdviewer_test.go`:

```go
package markdownviewer

import (
	"strings"
	"testing"
)

func TestRenderDefaults(t *testing.T) {
	out, err := Render([]byte("# Hi\n"))
	if err != nil {
		t.Fatal(err)
	}
	s := string(out)
	if !strings.Contains(s, "<!doctype html>") || !strings.Contains(s, `<h1 id="hi">Hi</h1>`) {
		t.Fatalf("got %q", s)
	}
}

func TestFragmentOption(t *testing.T) {
	out, _ := Render([]byte("# Hi\n"), Fragment())
	if strings.Contains(string(out), "<!doctype") {
		t.Fatal("fragment should not be a full page")
	}
}

func TestOptionStacking(t *testing.T) {
	out, _ := Render([]byte("<script>x</script>\n\n$a$\n"), Fragment(), AllowRawHTML(), DisableMath())
	s := string(out)
	if !strings.Contains(s, "<script>x</script>") {
		t.Fatal("AllowRawHTML not applied")
	}
	if strings.Contains(s, `class="math`) {
		t.Fatal("DisableMath not applied")
	}
}

func TestParse(t *testing.T) {
	doc, err := Parse([]byte("# Hi\n"))
	if err != nil || len(doc.Children()) != 1 {
		t.Fatalf("doc: %v %v", doc, err)
	}
}
```

- [ ] **Step 2: Run, expect FAIL.**
- [ ] **Step 3: Implement `mdviewer.go`**

```go
// Package markdownviewer is an embeddable Markdown previewer: it parses
// Markdown (CommonMark + GFM + modern extensions) into a stable document
// model and renders themed, self-contained, sanitized HTML.
//
// Quick start:
//
//	html, err := markdownviewer.Render(src)
//
// See the document package for the AST and render/html for renderer options.
package markdownviewer

import (
	"bytes"
	"io"

	"github.com/sriannamalai/markdownviewer/document"
	"github.com/sriannamalai/markdownviewer/parser"
	htmlrender "github.com/sriannamalai/markdownviewer/render/html"
)

type Resolver = htmlrender.Resolver

const (
	ResolveLink     = htmlrender.ResolveLink
	ResolveImage    = htmlrender.ResolveImage
	ResolveWikiLink = htmlrender.ResolveWikiLink
)

type config struct {
	render htmlrender.Options
}

type Option func(*config)

func WithTheme(name string) Option     { return func(c *config) { c.render.ThemeName = name } }
func Fragment() Option                 { return func(c *config) { c.render.Fragment = true } }
func AllowRawHTML() Option             { return func(c *config) { c.render.Unsafe = true } }
func DisableMermaid() Option           { return func(c *config) { c.render.Mermaid = false } }
func DisableMath() Option              { return func(c *config) { c.render.Math = false } }
func DisableHighlighting() Option      { return func(c *config) { c.render.Highlight = false } }
func WithResolver(r Resolver) Option   { return func(c *config) { c.render.Resolver = r } }

// Parse returns the document model for src.
func Parse(src []byte) (*document.Document, error) { return parser.Parse(src) }

// Render parses src and renders HTML with the given options.
func Render(src []byte, opts ...Option) ([]byte, error) {
	var buf bytes.Buffer
	if err := RenderTo(&buf, src, opts...); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// RenderTo renders into w.
func RenderTo(w io.Writer, src []byte, opts ...Option) error {
	cfg := &config{render: htmlrender.DefaultOptions()}
	for _, o := range opts {
		o(cfg)
	}
	doc, err := parser.Parse(src)
	if err != nil {
		return err
	}
	return htmlrender.Render(w, doc, cfg.render)
}
```

- [ ] **Step 4: Run until PASS.**
- [ ] **Step 5: Commit** — `git add mdviewer.go mdviewer_test.go && git commit -m "feat: facade API with functional options"`

---

### Task 15: `mdview` CLI

**Files:**
- Create: `cmd/mdview/main.go`
- Test: `cmd/mdview/main_test.go`

**Interfaces:**
- Consumes: facade API only (dogfooding the public surface).
- Produces: `mdview [flags] [file.md]` — stdin when no file; flags `-o out.html` (default stdout), `-theme light|dark|auto`, `-fragment`, `-unsafe`, `-no-mermaid`, `-no-math`, `-no-highlight`.

- [ ] **Step 1: Failing test** — `cmd/mdview/main_test.go` (test the run function, not main):

```go
package main

import (
	"bytes"
	"strings"
	"testing"
)

func TestRunStdinToStdout(t *testing.T) {
	var out bytes.Buffer
	err := run([]string{"-fragment"}, strings.NewReader("# Hi\n"), &out)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), `<h1 id="hi">Hi</h1>`) {
		t.Fatalf("got %q", out.String())
	}
}

func TestRunBadTheme(t *testing.T) {
	var out bytes.Buffer
	if err := run([]string{"-theme", "neon"}, strings.NewReader("x"), &out); err == nil {
		t.Fatal("expected error for unknown theme")
	}
}
```

- [ ] **Step 2: Run, expect FAIL.**
- [ ] **Step 3: Implement `cmd/mdview/main.go`**

```go
// Command mdview renders Markdown to self-contained HTML.
//
//	mdview README.md -o readme.html
//	cat notes.md | mdview -theme dark > notes.html
package main

import (
	"flag"
	"fmt"
	"io"
	"os"

	viewer "github.com/sriannamalai/markdownviewer"
)

func main() {
	if err := run(os.Args[1:], os.Stdin, os.Stdout); err != nil {
		fmt.Fprintln(os.Stderr, "mdview:", err)
		os.Exit(1)
	}
}

func run(args []string, stdin io.Reader, stdout io.Writer) error {
	fs := flag.NewFlagSet("mdview", flag.ContinueOnError)
	out := fs.String("o", "", "output file (default stdout)")
	themeName := fs.String("theme", "auto", "theme: light, dark, auto")
	fragment := fs.Bool("fragment", false, "emit body-only HTML fragment")
	unsafe := fs.Bool("unsafe", false, "allow raw HTML and all URL schemes")
	noMermaid := fs.Bool("no-mermaid", false, "disable mermaid diagrams")
	noMath := fs.Bool("no-math", false, "disable KaTeX math")
	noHighlight := fs.Bool("no-highlight", false, "disable syntax highlighting")
	if err := fs.Parse(args); err != nil {
		return err
	}

	var src []byte
	var err error
	if fs.NArg() > 0 {
		src, err = os.ReadFile(fs.Arg(0))
	} else {
		src, err = io.ReadAll(stdin)
	}
	if err != nil {
		return err
	}

	opts := []viewer.Option{viewer.WithTheme(*themeName)}
	if *fragment {
		opts = append(opts, viewer.Fragment())
	}
	if *unsafe {
		opts = append(opts, viewer.AllowRawHTML())
	}
	if *noMermaid {
		opts = append(opts, viewer.DisableMermaid())
	}
	if *noMath {
		opts = append(opts, viewer.DisableMath())
	}
	if *noHighlight {
		opts = append(opts, viewer.DisableHighlighting())
	}

	w := stdout
	if *out != "" {
		f, err := os.Create(*out)
		if err != nil {
			return err
		}
		defer f.Close()
		w = f
	}
	return viewer.RenderTo(w, src, opts...)
}
```

(Note: unknown theme must fail — verify `theme.Get` error propagates through `RenderTo`; the Task 12 implementation returns it from `renderPage`.)

- [ ] **Step 4: Run until PASS**, then end-to-end: `go run ./cmd/mdview README.md -o /tmp/readme.html && open /tmp/readme.html`.
- [ ] **Step 5: Commit** — `git add cmd/ && git commit -m "feat(mdview): CLI renderer over the facade API"`

---

### Task 16: Conformance suites + fuzzing

**Files:**
- Create: `parser/spec_test.go`, `parser/testdata/commonmark-0.31.2.json` (fetched), `parser/testdata/gfm-extras.json` (hand-written), `parser/fuzz_test.go`
- Modify: possibly `render/html/renderer.go` for formatting fixes the suite reveals.

**Interfaces:**
- Consumes: `parser.ParseWith(CommonMarkOnly())`, `htmlrender.Render`.
- Produces: conformance report; documented skip list; fuzz targets `FuzzParse`, `FuzzParseRender`.

- [ ] **Step 1: Fetch the CommonMark suite**

```bash
curl -fsSL https://spec.commonmark.org/0.31.2/spec.json -o parser/testdata/commonmark-0.31.2.json
```

- [ ] **Step 2: Hand-write `parser/testdata/gfm-extras.json`** — 12+ cases covering tables (alignment row, escaped pipes), strikethrough, task lists, autolinks (www., http, email), in the same `{"markdown","html","example"}` shape, with expected HTML matching **our** renderer conventions (use the golden outputs from Tasks 9–10 as the source of truth for formatting).

- [ ] **Step 3: Write `parser/spec_test.go`**

```go
package parser_test

import (
	"bytes"
	"encoding/json"
	"os"
	"testing"

	"github.com/sriannamalai/markdownviewer/parser"
	htmlrender "github.com/sriannamalai/markdownviewer/render/html"
)

type specCase struct {
	Markdown string `json:"markdown"`
	HTML     string `json:"html"`
	Example  int    `json:"example"`
}

// skips documents examples our pipeline intentionally renders differently.
// Every entry needs a reason. Adding entries requires justification in the
// commit message; the count is asserted so it cannot silently grow.
var skips = map[int]string{}

const maxSkips = 15

func TestCommonMarkSpec(t *testing.T) {
	runSpec(t, "testdata/commonmark-0.31.2.json", parser.CommonMarkOnly())
}

func TestGFMExtras(t *testing.T) {
	runSpec(t, "testdata/gfm-extras.json", parser.Default())
}

func runSpec(t *testing.T, path string, cfg parser.Config) {
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var cases []specCase
	if err := json.Unmarshal(data, &cases); err != nil {
		t.Fatal(err)
	}
	if len(skips) > maxSkips {
		t.Fatalf("skip list has %d entries; cap is %d", len(skips), maxSkips)
	}
	opts := htmlrender.DefaultOptions()
	opts.Fragment = true
	opts.Unsafe = true // spec output includes raw HTML verbatim
	opts.HeadingAnchors = false
	opts.Highlight = false
	opts.Mermaid = false
	opts.Math = false
	failed := 0
	for _, c := range cases {
		if reason, ok := skips[c.Example]; ok {
			t.Logf("example %d skipped: %s", c.Example, reason)
			continue
		}
		doc, err := parser.ParseWith([]byte(c.Markdown), cfg)
		if err != nil {
			t.Errorf("example %d: parse error %v", c.Example, err)
			continue
		}
		var buf bytes.Buffer
		if err := htmlrender.Render(&buf, doc, opts); err != nil {
			t.Errorf("example %d: render error %v", c.Example, err)
			continue
		}
		if buf.String() != c.HTML {
			failed++
			if failed <= 20 {
				t.Errorf("example %d:\ninput  %q\ngot    %q\nwant   %q", c.Example, c.Markdown, buf.String(), c.HTML)
			}
		}
	}
	if failed > 0 {
		t.Errorf("%d spec examples failed", failed)
	}
}
```

- [ ] **Step 4: Run and burn down failures** — `go test ./parser -run Spec -v`. Expect an initial cluster of formatting diffs (tight/loose list newlines, code-fence info strings, nested blockquote spacing). Fix them **in the renderer** to match reference output. Legitimate semantic differences (e.g. our `Diagram`/`math` fence handling is disabled here, so there should be very few) go into `skips` with reasons, hard-capped at 15. After renderer fixes, regenerate render goldens (`-update`), eyeball, and re-run the full tree: `go test ./...`.

- [ ] **Step 5: Write `parser/fuzz_test.go`**

```go
package parser_test

import (
	"bytes"
	"testing"

	"github.com/sriannamalai/markdownviewer/parser"
	htmlrender "github.com/sriannamalai/markdownviewer/render/html"
)

func FuzzParseRender(f *testing.F) {
	seeds := []string{
		"# h\n\npara\n", "- [x] t\n", "| a |\n|---|\n| b |\n",
		"```go\nx\n```\n", "$a$ $$b$$\n", "> [!NOTE]\n> x\n",
		"[[w]] :smile: ~~s~~ [l](u) ![i](u)\n", "Term\n: def\n",
		"---\nk: v\n---\nx[^1]\n\n[^1]: n\n", "$$\nx\n$$\n",
		"<div><script>x</script></div>\n",
	}
	for _, s := range seeds {
		f.Add([]byte(s))
	}
	f.Fuzz(func(t *testing.T, data []byte) {
		doc, err := parser.Parse(data)
		if err != nil {
			return
		}
		var buf bytes.Buffer
		opts := htmlrender.DefaultOptions()
		opts.Fragment = true
		_ = htmlrender.Render(&buf, doc, opts)
	})
}
```

- [ ] **Step 6: Fuzz for 60 seconds** — `go test ./parser -fuzz FuzzParseRender -fuzztime 60s`. Fix any panic found (nil children assumptions, index-out-of-range in admonition/table paths are the likely suspects); add the crasher as a regression seed.
- [ ] **Step 7: Commit** — `git add parser/ render/ && git commit -m "test: CommonMark 0.31.2 + GFM conformance suites and fuzz targets"`

---

### Task 17: CI, OSS hygiene, README

**Files:**
- Create: `.github/workflows/ci.yml`, `scripts/check-coverage.sh`, `NOTICE`, `CONTRIBUTING.md`, `CODE_OF_CONDUCT.md`, `SECURITY.md`
- Modify: `README.md`

**Interfaces:** consumes everything; produces the public face of the repo.

- [ ] **Step 1: `.github/workflows/ci.yml`**

```yaml
name: CI
on:
  push: {branches: [main]}
  pull_request:
jobs:
  test:
    strategy:
      matrix:
        os: [ubuntu-latest, macos-latest, windows-latest]
    runs-on: ${{ matrix.os }}
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with: {go-version: '1.23.x'}
      - run: go vet ./...
      - run: go test -race ./...
  lint:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with: {go-version: '1.23.x'}
      - run: go install honnef.co/go/tools/cmd/staticcheck@latest
      - run: staticcheck ./...
  coverage:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with: {go-version: '1.23.x'}
      - run: go test -coverprofile=cover.out ./...
      - run: ./scripts/check-coverage.sh cover.out 75
```

- [ ] **Step 2: `scripts/check-coverage.sh`**

```bash
#!/usr/bin/env bash
# Fails if total coverage in $1 is below $2 percent.
set -euo pipefail
total=$(go tool cover -func="$1" | awk '/^total:/ {gsub(/%/,"",$3); print $3}')
echo "total coverage: ${total}%"
awk -v t="$total" -v min="$2" 'BEGIN {exit (t+0 < min+0) ? 1 : 0}'
```

`chmod +x scripts/check-coverage.sh` and run locally: `go test -coverprofile=/tmp/c.out ./... && ./scripts/check-coverage.sh /tmp/c.out 75`. If below 75, add tests before proceeding (likely gaps: `theme.Get` error path, CLI flags, dump labels).

- [ ] **Step 3: Hygiene files.**

`NOTICE`:

```
MarkDownViewer
Copyright 2026 Sri Annamalai

This product includes software developed by third parties:
- goldmark (MIT) — https://github.com/yuin/goldmark
- chroma (MIT) — https://github.com/alecthomas/chroma
- bluemonday (BSD-3-Clause) — https://github.com/microcosm-cc/bluemonday
- mermaid (MIT), embedded — see third_party/mermaid/LICENSE
- KaTeX (MIT), embedded (including KaTeX fonts, SIL OFL 1.1) — see third_party/katex/LICENSE
```

`CONTRIBUTING.md`: how to run tests (`go test ./...`), golden-file workflow (`-update` + eyeball), asset upgrade flow (`scripts/fetch-assets.sh`), and DCO: contributions require `git commit -s` sign-off certifying the Developer Certificate of Origin (link developercertificate.org); PRs must pass CI; conventional-commit style requested.

`SECURITY.md`: supported versions (latest v0.x), private reports to `mail@sriannamalai.com`, 90-day coordinated disclosure, explicit note that sanitizer bypasses are in scope.

`CODE_OF_CONDUCT.md`:

```bash
curl -fsSL https://www.contributor-covenant.org/version/2/1/code_of_conduct/code_of_conduct.md -o CODE_OF_CONDUCT.md
```

Then edit the contact placeholder to `mail@sriannamalai.com`.

- [ ] **Step 4: Rewrite `README.md`** with: one-paragraph pitch (embeddable, layered document model, self-contained themed HTML, safe by default); feature table; install (`go get github.com/sriannamalai/markdownviewer`); quick-start code (facade `Render` with options — copy the working example from `mdviewer_test.go`); CLI usage block; theming section (built-ins + variable override example); security model summary; roadmap section from the spec (FFI/WASM → mobile bindings → native renderer, all over the frozen `document` model); license + NOTICE pointer. Keep it plain pipe-table Markdown (it is itself a test document for the viewer — `go run ./cmd/mdview README.md` should look good).

- [ ] **Step 5: Full verification**

```bash
go vet ./... && go test -race ./... && go build ./...
go run ./cmd/mdview README.md -o /tmp/readme.html && open /tmp/readme.html
```

- [ ] **Step 6: Commit**

```bash
git add .github/ scripts/ NOTICE CONTRIBUTING.md CODE_OF_CONDUCT.md SECURITY.md README.md
git commit -m "chore: CI matrix, coverage gate, OSS hygiene files, README"
```

---

## Completion checklist (maps to spec's v1 deliverables)

- [ ] `document`, `parser`, `render/html`, `theme` packages — Tasks 1–13
- [ ] `mdview` CLI — Task 15
- [ ] Golden files, spec suites, fuzz, sanitizer corpus, CI — Tasks 9–13, 16, 17
- [ ] README with embedding examples — Task 17
- [ ] `NOTICE`, `CONTRIBUTING.md`, `CODE_OF_CONDUCT.md`, `SECURITY.md`, `third_party/README.md` — Tasks 13, 17
