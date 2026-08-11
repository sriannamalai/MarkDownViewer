package tree_test

import (
	"crypto/sha256"
	"encoding/hex"
	"strings"
	"testing"

	"github.com/sriannamalai/markdownviewer/document"
	"github.com/sriannamalai/markdownviewer/parser"
	"github.com/sriannamalai/markdownviewer/render/tree"
	"github.com/sriannamalai/markdownviewer/resolve"
)

// build parses md and builds its tree with opts (Source wired to md).
func build(t *testing.T, md string, opts tree.Options) *tree.Tree {
	t.Helper()
	doc, err := parser.Parse([]byte(md))
	if err != nil {
		t.Fatal(err)
	}
	opts.Source = []byte(md)
	tr, err := tree.Build(doc, opts)
	if err != nil {
		t.Fatal(err)
	}
	return tr
}

func buildDefault(t *testing.T, md string) *tree.Tree {
	return build(t, md, tree.DefaultOptions())
}

func TestBuildNilDocument(t *testing.T) {
	if _, err := tree.Build(nil, tree.DefaultOptions()); err == nil {
		t.Fatal("Build(nil) succeeded, want error")
	}
}

func TestHeadingAnchors(t *testing.T) {
	tr := buildDefault(t, "# Hello World\n")
	h := tr.Blocks[0].(*tree.Heading)
	if h.Level != 1 || h.AnchorID != "hello-world" {
		t.Fatalf("got level=%d anchor=%q", h.Level, h.AnchorID)
	}
	opts := tree.DefaultOptions()
	opts.HeadingAnchors = false
	tr = build(t, "# Hello World\n", opts)
	if a := tr.Blocks[0].(*tree.Heading).AnchorID; a != "" {
		t.Fatalf("anchors off: got anchor %q, want empty", a)
	}
}

func TestListShapes(t *testing.T) {
	tr := buildDefault(t, "5. five\n6. six\n\n- [x] done\n- [ ] todo\n- plain\n")
	ol := tr.Blocks[0].(*tree.List)
	if !ol.Ordered || ol.Start != 5 || !ol.Tight || len(ol.Items) != 2 {
		t.Fatalf("ordered list: %+v", ol)
	}
	if ol.Items[0].Task != nil {
		t.Fatal("ordered item is not a task")
	}
	ul := tr.Blocks[1].(*tree.List)
	if ul.Ordered || len(ul.Items) != 3 {
		t.Fatalf("unordered list: %+v", ul)
	}
	if v := ul.Items[0].Task; v == nil || !*v {
		t.Fatalf("done task: got %v, want true", v)
	}
	if v := ul.Items[1].Task; v == nil || *v {
		t.Fatalf("todo task: got %v, want false", v)
	}
	if ul.Items[2].Task != nil {
		t.Fatal("plain item carries a task state")
	}
}

func TestCodeBlockLabelAndRunsStub(t *testing.T) {
	tr := buildDefault(t, "```go\npackage x\n```\n\n```\nbare\n```\n")
	cb := tr.Blocks[0].(*tree.CodeBlock)
	if cb.Language != "go" || cb.Label != "go" || cb.Text != "package x\n" || cb.Runs != nil {
		t.Fatalf("go block: %+v", cb)
	}
	bare := tr.Blocks[1].(*tree.CodeBlock)
	if bare.Language != "" || bare.Label != "code" || bare.Runs != nil {
		t.Fatalf("bare block: %+v", bare)
	}
}

func TestMathToggle(t *testing.T) {
	const md = "$$\nx^2\n$$\n\nInline $y$ math.\n"
	tr := buildDefault(t, md)
	mb := tr.Blocks[0].(*tree.MathBlock)
	if mb.Source != "x^2\n" {
		t.Fatalf("math source %q", mb.Source)
	}
	para := tr.Blocks[1].(*tree.Paragraph)
	found := false
	for _, in := range para.Children {
		if mi, ok := in.(*tree.MathInline); ok {
			found = true
			if mi.Source != "y" || mi.Display {
				t.Fatalf("inline math: %+v", mi)
			}
		}
	}
	if !found {
		t.Fatal("no MathInline in paragraph")
	}

	// Math off: same fallbacks the HTML renderer applies — a code block
	// with language "math", and a code span of the source.
	opts := tree.DefaultOptions()
	opts.Math = false
	tr = build(t, md, opts)
	cb := tr.Blocks[0].(*tree.CodeBlock)
	if cb.Language != "math" || cb.Text != "x^2\n" || cb.Runs != nil {
		t.Fatalf("math-off block: %+v", cb)
	}
	para = tr.Blocks[1].(*tree.Paragraph)
	for _, in := range para.Children {
		if _, ok := in.(*tree.MathInline); ok {
			t.Fatal("math off still produced MathInline")
		}
	}
	foundSpan := false
	for _, in := range para.Children {
		if cs, ok := in.(*tree.CodeSpan); ok && cs.Value == "y" {
			foundSpan = true
		}
	}
	if !foundSpan {
		t.Fatal("math-off inline did not fall back to CodeSpan")
	}
}

func TestMathInlineDisplay(t *testing.T) {
	tr := buildDefault(t, "Mid $$z^3$$ sentence.\n")
	para := tr.Blocks[0].(*tree.Paragraph)
	for _, in := range para.Children {
		if mi, ok := in.(*tree.MathInline); ok {
			if !mi.Display || mi.Source != "z^3" {
				t.Fatalf("display math: %+v", mi)
			}
			return
		}
	}
	t.Fatal("no MathInline found")
}

func TestMermaidToggle(t *testing.T) {
	const md = "```mermaid\ngraph TD\n```\n"
	tr := buildDefault(t, md)
	d := tr.Blocks[0].(*tree.Diagram)
	if d.Engine != "mermaid" || d.Source != "graph TD\n" {
		t.Fatalf("diagram: %+v", d)
	}
	opts := tree.DefaultOptions()
	opts.Mermaid = false
	tr = build(t, md, opts)
	cb := tr.Blocks[0].(*tree.CodeBlock)
	if cb.Language != "mermaid" || cb.Text != "graph TD\n" {
		t.Fatalf("mermaid-off block: %+v", cb)
	}
}

func TestAdmonitionSharedDerivation(t *testing.T) {
	tr := buildDefault(t, "> [!TIP]\n> Handy.\n")
	ad := tr.Blocks[0].(*tree.Admonition)
	if ad.Variant != "tip" || ad.Title != "Tip" {
		t.Fatalf("admonition: variant=%q title=%q", ad.Variant, ad.Title)
	}
	// Hand-built empty variant takes the shared "note" default.
	doc := &document.Document{}
	doc.AppendChild(&document.Admonition{})
	tr2, err := tree.Build(doc, tree.DefaultOptions())
	if err != nil {
		t.Fatal(err)
	}
	ad = tr2.Blocks[0].(*tree.Admonition)
	if ad.Variant != "note" || ad.Title != "Note" {
		t.Fatalf("default admonition: variant=%q title=%q", ad.Variant, ad.Title)
	}
}

func TestTable(t *testing.T) {
	tr := buildDefault(t, "| L | C | R |\n|:--|:-:|--:|\n| a | b | c |\n| d | e | f |\n")
	tb := tr.Blocks[0].(*tree.Table)
	want := []document.Alignment{document.AlignLeft, document.AlignCenter, document.AlignRight}
	if len(tb.Alignments) != 3 || tb.Alignments[0] != want[0] || tb.Alignments[1] != want[1] || tb.Alignments[2] != want[2] {
		t.Fatalf("alignments: %v", tb.Alignments)
	}
	if len(tb.Header) != 3 || len(tb.Rows) != 2 || len(tb.Rows[1]) != 3 {
		t.Fatalf("shape: header=%d rows=%d", len(tb.Header), len(tb.Rows))
	}
	if v := tb.Rows[0][1][0].(*tree.Text).Value; v != "b" {
		t.Fatalf("cell text %q", v)
	}
}

func TestRawHTMLPolicy(t *testing.T) {
	const md = "<div onclick=\"x()\"><script>a()</script><em>ok</em></div>\n\npara <b onmouseover=\"y()\">b</b> tail\n"
	tr := buildDefault(t, md)
	hb := tr.Blocks[0].(*tree.HTMLBlock)
	if hb.Unsafe {
		t.Fatal("sanitized block flagged unsafe")
	}
	if strings.Contains(hb.HTML, "script") || strings.Contains(hb.HTML, "onclick") {
		t.Fatalf("sanitize leak: %q", hb.HTML)
	}
	if !strings.Contains(hb.HTML, "<em>ok</em>") {
		t.Fatalf("sanitize dropped safe markup: %q", hb.HTML)
	}
	para := tr.Blocks[1].(*tree.Paragraph)
	for _, in := range para.Children {
		if hi, ok := in.(*tree.HTMLInline); ok {
			if hi.Unsafe || strings.Contains(hi.HTML, "onmouseover") {
				t.Fatalf("inline sanitize leak: %+v", hi)
			}
		}
	}

	opts := tree.DefaultOptions()
	opts.AllowRawHTML = true
	tr = build(t, md, opts)
	hb = tr.Blocks[0].(*tree.HTMLBlock)
	if !hb.Unsafe || !strings.Contains(hb.HTML, "<script>") {
		t.Fatalf("raw mode: unsafe=%v html=%q", hb.Unsafe, hb.HTML)
	}
}

// firstLink returns the n-th (0-based) Link inline in the tree's
// paragraphs.
func nthLink(t *testing.T, tr *tree.Tree, n int) *tree.Link {
	t.Helper()
	i := 0
	for _, b := range tr.Blocks {
		p, ok := b.(*tree.Paragraph)
		if !ok {
			continue
		}
		for _, in := range p.Children {
			if l, ok := in.(*tree.Link); ok {
				if i == n {
					return l
				}
				i++
			}
		}
	}
	t.Fatalf("link %d not found", n)
	return nil
}

func TestLinkPolicy(t *testing.T) {
	md := "[ok](https://e.com/a%20b) then [bad](javascript:alert(1)) then [empty]() done.\n"
	tr := buildDefault(t, md)
	if l := nthLink(t, tr, 0); l.URL != "https://e.com/a%20b" || l.Blocked {
		t.Fatalf("safe link: %+v", l)
	}
	// Blocked scheme: url emptied AND blocked marked — distinct from the
	// empty-but-allowed destination below.
	if l := nthLink(t, tr, 1); l.URL != "" || !l.Blocked {
		t.Fatalf("blocked link: %+v", l)
	}
	if l := nthLink(t, tr, 2); l.URL != "" || l.Blocked {
		t.Fatalf("empty link: %+v", l)
	}
}

func TestResolverContract(t *testing.T) {
	md := "[a](page-a) and [[Wiki]] and ![i](img)\n"
	var seen []string
	opts := tree.DefaultOptions()
	opts.Resolver = func(kind resolve.ResolveKind, target string) (string, bool) {
		seen = append(seen, target)
		switch kind {
		case resolve.ResolveLink:
			// Resolver-returned URLs are host-trusted and carried
			// verbatim — even a scheme SafeURL would block.
			return "app://open/" + target, true
		case resolve.ResolveWikiLink:
			return "", false // decline → default ".md" resolution
		default:
			return "asset://img", true
		}
	}
	tr := build(t, md, opts)
	l := nthLink(t, tr, 0)
	if l.URL != "app://open/page-a" || l.Blocked || l.Source != "" {
		t.Fatalf("resolved link: %+v", l)
	}
	w := nthLink(t, tr, 1)
	if w.URL != "Wiki.md" || w.Blocked || w.Source != "wikiLink" {
		t.Fatalf("declined wiki link: %+v", w)
	}
	p := tr.Blocks[0].(*tree.Paragraph)
	var img *tree.Image
	for _, in := range p.Children {
		if i, ok := in.(*tree.Image); ok {
			img = i
		}
	}
	if img == nil || img.URL != "asset://img" || img.Blocked {
		t.Fatalf("resolved image: %+v", img)
	}
	if len(seen) != 3 {
		t.Fatalf("resolver calls: %v", seen)
	}
}

func TestImageBlockedKeepsAlt(t *testing.T) {
	tr := buildDefault(t, "![alt text](javascript:x) end\n")
	p := tr.Blocks[0].(*tree.Paragraph)
	img := p.Children[0].(*tree.Image)
	if img.URL != "" || !img.Blocked || img.Alt != "alt text" {
		t.Fatalf("blocked image: %+v", img)
	}
}

func TestFootnotePairing(t *testing.T) {
	tr := buildDefault(t, "one[^a] two[^a] three[^b]\n\n[^a]: A body.\n[^b]: B body.\n")
	if len(tr.Footnotes) != 2 {
		t.Fatalf("footnotes: %d", len(tr.Footnotes))
	}
	a, b := tr.Footnotes[0], tr.Footnotes[1]
	if a.Index != 1 || a.RefCount != 2 || b.Index != 2 || b.RefCount != 1 {
		t.Fatalf("pairing: a=%+v b=%+v", a, b)
	}
	if len(a.Blocks) == 0 {
		t.Fatal("footnote body empty")
	}
	// Reference sites appear inline with the matching index.
	p := tr.Blocks[0].(*tree.Paragraph)
	var refs []int
	for _, in := range p.Children {
		if r, ok := in.(*tree.FootnoteRef); ok {
			refs = append(refs, r.Index)
		}
	}
	if len(refs) != 3 || refs[0] != 1 || refs[1] != 1 || refs[2] != 2 {
		t.Fatalf("ref sites: %v", refs)
	}
}

func TestDefinitionList(t *testing.T) {
	tr := buildDefault(t, "Term\n: Description one.\n: Description two.\n")
	dl := tr.Blocks[0].(*tree.DefinitionList)
	if len(dl.Blocks) != 3 {
		t.Fatalf("dl blocks: %d", len(dl.Blocks))
	}
	term := dl.Blocks[0].(*tree.DefinitionTerm)
	if term.Children[0].(*tree.Text).Value != "Term" {
		t.Fatalf("term: %+v", term)
	}
	if _, ok := dl.Blocks[1].(*tree.DefinitionDesc); !ok {
		t.Fatalf("desc: %T", dl.Blocks[1])
	}
}

func TestBlockIDContentHash(t *testing.T) {
	const md = "# Hi\n\npara one\n\n---\n\n---\n"
	doc, err := parser.Parse([]byte(md))
	if err != nil {
		t.Fatal(err)
	}
	opts := tree.DefaultOptions()
	opts.Source = []byte(md)
	tr, err := tree.Build(doc, opts)
	if err != nil {
		t.Fatal(err)
	}
	// id = hex(sha256(source bytes of the block's span))[:16].
	h := tr.Blocks[0].(*tree.Heading)
	sum := sha256.Sum256([]byte(md)[h.Span.StartOffset:h.Span.EndOffset])
	if want := hex.EncodeToString(sum[:8]); h.ID != want {
		t.Fatalf("heading id %q, want %q", h.ID, want)
	}
	// Content hash: byte-identical blocks share an id.
	hr1 := tr.Blocks[2].(*tree.ThematicBreak)
	hr2 := tr.Blocks[3].(*tree.ThematicBreak)
	if hr1.ID != hr2.ID {
		t.Fatalf("identical blocks got distinct ids %q %q", hr1.ID, hr2.ID)
	}
	// And a distinct block gets a distinct id.
	if p := tr.Blocks[1].(*tree.Paragraph); p.ID == h.ID || p.ID == hr1.ID {
		t.Fatal("distinct blocks share an id")
	}
	// Stability: an edit elsewhere must not move content-hashed ids.
	const md2 = "# Hi\n\npara CHANGED\n\n---\n\n---\n"
	doc2, err := parser.Parse([]byte(md2))
	if err != nil {
		t.Fatal(err)
	}
	opts.Source = []byte(md2)
	tr2, err := tree.Build(doc2, opts)
	if err != nil {
		t.Fatal(err)
	}
	if got := tr2.Blocks[0].(*tree.Heading).ID; got != h.ID {
		t.Fatalf("heading id moved on unrelated edit: %q vs %q", got, h.ID)
	}
	if got := tr2.Blocks[1].(*tree.Paragraph).ID; got == tr.Blocks[1].(*tree.Paragraph).ID {
		t.Fatal("edited paragraph kept its id")
	}
}

func TestBlockIDFallback(t *testing.T) {
	const md = "# Hi\n\npara\n"
	doc, err := parser.Parse([]byte(md))
	if err != nil {
		t.Fatal(err)
	}
	// No Source: every block takes the domain-separated kind+ordinal
	// fallback, hex(sha256("\x00mdv-fallback\x00" + kind + ":" +
	// ordinal))[:16] over the document-order block count.
	tr, err := tree.Build(doc, tree.DefaultOptions())
	if err != nil {
		t.Fatal(err)
	}
	sum := sha256.Sum256([]byte("\x00mdv-fallback\x00heading:0"))
	if want := hex.EncodeToString(sum[:8]); tr.Blocks[0].(*tree.Heading).ID != want {
		t.Fatalf("fallback id %q, want %q", tr.Blocks[0].(*tree.Heading).ID, want)
	}
	sum = sha256.Sum256([]byte("\x00mdv-fallback\x00paragraph:1"))
	if want := hex.EncodeToString(sum[:8]); tr.Blocks[1].(*tree.Paragraph).ID != want {
		t.Fatalf("fallback id %q, want %q", tr.Blocks[1].(*tree.Paragraph).ID, want)
	}
	// Determinism: same document, same ids.
	tr2, err := tree.Build(doc, tree.DefaultOptions())
	if err != nil {
		t.Fatal(err)
	}
	if tr2.Blocks[0].(*tree.Heading).ID != tr.Blocks[0].(*tree.Heading).ID {
		t.Fatal("fallback ids not deterministic")
	}
	// A zero-span block falls back even when Source is present.
	hand := &document.Document{}
	hand.AppendChild(&document.Paragraph{})
	opts := tree.DefaultOptions()
	opts.Source = []byte(md)
	tr3, err := tree.Build(hand, opts)
	if err != nil {
		t.Fatal(err)
	}
	sum = sha256.Sum256([]byte("\x00mdv-fallback\x00paragraph:0"))
	if want := hex.EncodeToString(sum[:8]); tr3.Blocks[0].(*tree.Paragraph).ID != want {
		t.Fatalf("zero-span id %q, want %q", tr3.Blocks[0].(*tree.Paragraph).ID, want)
	}
	// Domain separation: a block whose SOURCE literally spells the
	// fallback's kind+ordinal must not collide with a fallback id.
	const trap = "paragraph:0"
	tr4 := buildDefault(t, trap+"\n") // content-hashed: source present, span valid
	sum = sha256.Sum256([]byte(trap))
	if got := tr4.Blocks[0].(*tree.Paragraph).ID; got != hex.EncodeToString(sum[:8]) {
		t.Fatalf("trap paragraph not content-hashed: %q", got)
	}
	tr5, err := tree.Build(hand, tree.DefaultOptions()) // fallback: paragraph ordinal 0
	if err != nil {
		t.Fatal(err)
	}
	if tr4.Blocks[0].(*tree.Paragraph).ID == tr5.Blocks[0].(*tree.Paragraph).ID {
		t.Fatal("content id collides with fallback id for preimage \"paragraph:0\"")
	}
}

func TestInlineSpansCarried(t *testing.T) {
	tr := buildDefault(t, "plain *emphasis* tail\n")
	p := tr.Blocks[0].(*tree.Paragraph)
	if p.Span.IsZero() {
		t.Fatal("paragraph span missing")
	}
	txt := p.Children[0].(*tree.Text)
	if txt.Span.IsZero() {
		t.Fatal("text span missing")
	}
	em := p.Children[1].(*tree.Emphasis)
	if em.Span.IsZero() {
		t.Fatal("emphasis (content) span missing")
	}
}

func TestStrayInlineAtBlockLevel(t *testing.T) {
	// Hand-built documents may put an inline at block level; the tree
	// wraps it in an implicit paragraph (the HTML renderer emits it
	// bare).
	doc := &document.Document{}
	doc.AppendChild(&document.Text{Value: "stray"})
	tr, err := tree.Build(doc, tree.DefaultOptions())
	if err != nil {
		t.Fatal(err)
	}
	p, ok := tr.Blocks[0].(*tree.Paragraph)
	if !ok || p.Children[0].(*tree.Text).Value != "stray" {
		t.Fatalf("stray inline: %#v", tr.Blocks[0])
	}
}
