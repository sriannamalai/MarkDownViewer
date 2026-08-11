package tree_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"html"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/sriannamalai/markdownviewer/document"
	"github.com/sriannamalai/markdownviewer/parser"
	htmlrender "github.com/sriannamalai/markdownviewer/render/html"
	"github.com/sriannamalai/markdownviewer/render/tree"
)

// The fidelity rule (spec §1): the render tree's plain-text content
// must equal the HTML renderer's text content. These tests enforce it
// differentially — same document into both renderers, extract the text
// from each output, compare — across the local fixture corpus, the full
// CommonMark 0.31.2 suite, and the GFM-extras suite.
//
// Extraction: the HTML output is stripped of tags and entity-unescaped
// (the inverse of the renderer's esc); the tree contributes its literal
// text fields, with raw-HTML fields given the same strip+unescape
// treatment their markup gets on the HTML side, and the footnote
// backref glyph (an HTML-only decoration) removed. Whitespace is
// normalized on both sides, since block structure renders as
// tags+newlines in HTML but as tree nesting here.

func TestTextParityFixtures(t *testing.T) {
	mds, err := filepath.Glob("testdata/*.md")
	if err != nil || len(mds) == 0 {
		t.Fatalf("no fixtures: %v", err)
	}
	// Also sweep the HTML renderer's own fixtures.
	htmlMds, err := filepath.Glob(filepath.Join("..", "html", "testdata", "*.md"))
	if err != nil {
		t.Fatal(err)
	}
	for _, md := range append(mds, htmlMds...) {
		t.Run(filepath.Base(md), func(t *testing.T) {
			src, err := os.ReadFile(md)
			if err != nil {
				t.Fatal(err)
			}
			doc, err := parser.Parse(src)
			if err != nil {
				t.Fatal(err)
			}
			checkParity(t, doc, src, "")
		})
	}
}

func TestTextParityCommonMark(t *testing.T) {
	runParityCorpus(t, filepath.Join("..", "..", "parser", "testdata", "commonmark-0.31.2.json"),
		parser.CommonMarkOnly(), 652)
}

func TestTextParityGFMExtras(t *testing.T) {
	runParityCorpus(t, filepath.Join("..", "..", "parser", "testdata", "gfm-extras.json"),
		parser.Default(), 12)
}

func runParityCorpus(t *testing.T, path string, cfg parser.Config, wantCount int) {
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var cases []struct {
		Markdown string `json:"markdown"`
		Example  int    `json:"example"`
	}
	if err := json.Unmarshal(data, &cases); err != nil {
		t.Fatal(err)
	}
	if len(cases) != wantCount {
		t.Fatalf("%s: loaded %d cases, want %d", path, len(cases), wantCount)
	}
	for _, c := range cases {
		doc, err := parser.ParseWith([]byte(c.Markdown), cfg)
		if err != nil {
			t.Fatalf("example %d: parse: %v", c.Example, err)
		}
		checkParity(t, doc, []byte(c.Markdown), fmt.Sprintf("example %d: ", c.Example))
	}
}

func checkParity(t *testing.T, doc *document.Document, src []byte, label string) {
	t.Helper()
	hOpts := htmlrender.DefaultOptions()
	hOpts.Fragment = true
	hOpts.Highlight = false // chroma output interleaves its own markup; plain <pre><code> keeps text extraction exact
	var buf bytes.Buffer
	if err := htmlrender.Render(&buf, doc, hOpts); err != nil {
		t.Fatalf("%srender: %v", label, err)
	}
	tOpts := tree.DefaultOptions()
	tOpts.Source = src
	tr, err := tree.Build(doc, tOpts)
	if err != nil {
		t.Fatalf("%sbuild: %v", label, err)
	}
	htmlText := normalize(strings.ReplaceAll(textFromHTML(buf.String()), "↩", ""))
	treeText := normalize(textFromTree(tr))
	if htmlText != treeText {
		t.Errorf("%stext parity broken:\n--- html ---\n%q\n--- tree ---\n%q", label, htmlText, treeText)
	}
}

// textFromHTML strips tags (quote-aware inside tags) and unescapes
// entities — the inverse of the renderer's esc.
func textFromHTML(s string) string {
	var b strings.Builder
	inTag := false
	var quote byte
	for i := 0; i < len(s); i++ {
		c := s[i]
		if inTag {
			switch {
			case quote != 0:
				if c == quote {
					quote = 0
				}
			case c == '"' || c == '\'':
				quote = c
			case c == '>':
				inTag = false
			}
			continue
		}
		if c == '<' {
			inTag = true
			continue
		}
		b.WriteByte(c)
	}
	return html.UnescapeString(b.String())
}

func normalize(s string) string { return strings.Join(strings.Fields(s), " ") }

func textFromTree(tr *tree.Tree) string {
	var b strings.Builder
	for _, blk := range tr.Blocks {
		blockText(&b, blk)
	}
	for _, fn := range tr.Footnotes {
		for _, blk := range fn.Blocks {
			blockText(&b, blk)
		}
	}
	return b.String()
}

func blockText(b *strings.Builder, blk tree.Block) {
	b.WriteByte('\n')
	switch blk := blk.(type) {
	case *tree.Heading:
		inlinesText(b, blk.Children)
	case *tree.Paragraph:
		inlinesText(b, blk.Children)
	case *tree.BlockQuote:
		for _, c := range blk.Blocks {
			blockText(b, c)
		}
	case *tree.Admonition:
		b.WriteString(blk.Title) // the HTML shape shows the derived title
		for _, c := range blk.Blocks {
			blockText(b, c)
		}
	case *tree.List:
		for _, it := range blk.Items {
			for _, c := range it.Blocks {
				blockText(b, c)
			}
		}
	case *tree.CodeBlock:
		b.WriteString(blk.Text)
	case *tree.MathBlock:
		b.WriteString(blk.Source)
	case *tree.Diagram:
		b.WriteString(blk.Source)
	case *tree.Table:
		for _, cell := range blk.Header {
			b.WriteByte('\n')
			inlinesText(b, cell)
		}
		for _, row := range blk.Rows {
			for _, cell := range row {
				b.WriteByte('\n')
				inlinesText(b, cell)
			}
		}
	case *tree.ThematicBreak:
	case *tree.HTMLBlock:
		// The HTML side embeds this markup verbatim, so its text shows
		// through the same strip+unescape the whole output gets.
		b.WriteString(textFromHTML(blk.HTML))
	case *tree.DefinitionList:
		for _, c := range blk.Blocks {
			blockText(b, c)
		}
	case *tree.DefinitionTerm:
		inlinesText(b, blk.Children)
	case *tree.DefinitionDesc:
		for _, c := range blk.Blocks {
			blockText(b, c)
		}
	}
	b.WriteByte('\n')
}

func inlinesText(b *strings.Builder, ins []tree.Inline) {
	for _, in := range ins {
		switch in := in.(type) {
		case *tree.Text:
			b.WriteString(in.Value)
		case *tree.SoftBreak, *tree.HardBreak:
			b.WriteByte('\n')
		case *tree.Emphasis:
			inlinesText(b, in.Children)
		case *tree.Strong:
			inlinesText(b, in.Children)
		case *tree.Strikethrough:
			inlinesText(b, in.Children)
		case *tree.CodeSpan:
			b.WriteString(in.Value)
		case *tree.Link:
			inlinesText(b, in.Children)
		case *tree.Image: // alt renders as an attribute: no text either side
		case *tree.MathInline:
			b.WriteString(in.Source)
		case *tree.HTMLInline:
			b.WriteString(textFromHTML(in.HTML))
		case *tree.FootnoteRef:
			b.WriteString(strconv.Itoa(in.Index))
		}
	}
}
