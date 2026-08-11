package htmlrender

import (
	"bytes"
	"strings"
	"testing"

	"github.com/sriannamalai/markdownviewer/document"
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

func TestUnsafeWikiLinkStripped(t *testing.T) {
	got := render(t, "[[javascript:alert(1)]]\n", nil)
	if strings.Contains(got, "href=") {
		t.Fatalf("blocked wikilink should have no href attribute: %q", got)
	}
	if !strings.Contains(got, `<a class="wikilink">javascript:alert(1)</a>`) {
		t.Fatalf("blocked wikilink should render without href but keep class: %q", got)
	}
}

func TestUnsafeWikiLinkKeptWhenUnsafe(t *testing.T) {
	got := render(t, "[[javascript:alert(1)]]\n", func(o *Options) { o.Unsafe = true })
	if !strings.Contains(got, `href="javascript:alert(1).md"`) {
		t.Fatalf("got %q", got)
	}
}

func TestResolverCallback(t *testing.T) {
	// Resolver output (ok=true) is trusted per its documented contract:
	// hosts fully control resolution, so a custom scheme like "vfs://"
	// bypasses safeURL and comes through untouched.
	got := render(t, "![i](pic.png) [[P]]\n", func(o *Options) {
		o.Resolver = func(kind ResolveKind, target string) (string, bool) {
			return "vfs://" + target, true
		}
	})
	if !strings.Contains(got, `src="vfs://pic.png"`) || !strings.Contains(got, `href="vfs://P"`) {
		t.Fatalf("got %q", got)
	}
}

func TestResolverDeclineFallsBackToDefaultFiltering(t *testing.T) {
	// When the Resolver returns ok=false, the default resolution path
	// applies and safeURL still filters the result — a Resolver cannot
	// accidentally weaken the policy by declining.
	got := render(t, "[x](javascript:alert(1))\n", func(o *Options) {
		o.Resolver = func(kind ResolveKind, target string) (string, bool) {
			return "", false
		}
	})
	if strings.Contains(got, "href=") {
		t.Fatalf("declined resolver should still be filtered by safeURL: %q", got)
	}
}

func TestTaskListCheckboxes(t *testing.T) {
	got := render(t, "- [x] done\n", nil)
	if !strings.Contains(got, `<input type="checkbox" checked disabled />`) {
		t.Fatalf("got %q", got)
	}
}

// TestTightTaskListInlineShape is the tight-list half of issue 4: checkbox
// stays inline, directly before the item's text, with no wrapping <p> —
// unchanged from before except for the new class attributes.
func TestTightTaskListInlineShape(t *testing.T) {
	got := render(t, "- [x] done\n- [ ] todo\n", nil)
	want := "<ul class=\"contains-task-list\">\n" +
		"<li class=\"task-list-item\"><input type=\"checkbox\" checked disabled /> done</li>\n" +
		"<li class=\"task-list-item\"><input type=\"checkbox\" disabled /> todo</li>\n" +
		"</ul>\n"
	if got != want {
		t.Fatalf("\n got %q\nwant %q", got, want)
	}
}

// TestLooseTaskListChecksboxInsideParagraph is the loose-list half of issue
// 4: previously the checkbox rendered as a sibling before the <p>, which
// pushed the item's text onto its own line/block (a bullet-shaped checkbox
// floating above disconnected text). It must render inside the first
// paragraph instead, matching cmark-gfm's shape.
func TestLooseTaskListCheckboxInsideParagraph(t *testing.T) {
	got := render(t, "- [x] done\n\n- [ ] todo\n\n  second paragraph\n", nil)
	want := "<ul class=\"contains-task-list\">\n" +
		"<li class=\"task-list-item\">\n<p><input type=\"checkbox\" checked disabled /> done</p>\n</li>\n" +
		"<li class=\"task-list-item\">\n<p><input type=\"checkbox\" disabled /> todo</p>\n<p>second paragraph</p>\n</li>\n" +
		"</ul>\n"
	if got != want {
		t.Fatalf("\n got %q\nwant %q", got, want)
	}
}

// TestTaskListClassesOnlyOnTaskItems verifies a list that mixes task and
// plain items only tags the task <li>s, and that a list with no task items
// at all gets neither class.
func TestTaskListClassesOnlyOnTaskItems(t *testing.T) {
	got := render(t, "- [x] done\n- plain item\n", nil)
	if !strings.Contains(got, `<li class="task-list-item">`) {
		t.Errorf("task item missing class: %q", got)
	}
	if !strings.Contains(got, "<li>plain item</li>") {
		t.Errorf("plain item should not get task-list-item class: %q", got)
	}

	gotPlain := render(t, "- a\n- b\n", nil)
	if strings.Contains(gotPlain, "task-list-item") || strings.Contains(gotPlain, "contains-task-list") {
		t.Errorf("plain list should carry no task-list classes: %q", gotPlain)
	}
}

// TestEmptyLooseTaskItemFallsBackToSiblingCheckbox covers the loose-item
// edge case with no paragraph to attach the checkbox to (an empty task
// item): it must still render, falling back to the tight-style sibling
// checkbox rather than being dropped or panicking.
func TestEmptyLooseTaskItemFallsBackToSiblingCheckbox(t *testing.T) {
	got := render(t, "- [ ]\n\n- next item\n", nil)
	if !strings.Contains(got, `<li class="task-list-item">`) {
		t.Fatalf("got %q", got)
	}
	if !strings.Contains(got, `<input type="checkbox" disabled />`) {
		t.Fatalf("checkbox missing: %q", got)
	}
}

// renderHandBuiltDoc renders a hand-built *document.Document directly, bypassing the
// parser entirely — for exercising Render's defensive handling of
// host-constructed trees that don't match the shapes the parser produces.
func renderHandBuiltDoc(t *testing.T, doc *document.Document) string {
	t.Helper()
	opts := DefaultOptions()
	opts.Fragment = true
	opts.Highlight = false
	var buf bytes.Buffer
	if err := Render(&buf, doc, opts); err != nil {
		t.Fatal(err)
	}
	return buf.String()
}

func TestRenderAdmonitionEmptyVariantDoesNotPanic(t *testing.T) {
	doc := &document.Document{}
	adm := &document.Admonition{} // Variant left unset
	p := &document.Paragraph{}
	p.AppendChild(&document.Text{Value: "hello"})
	adm.AppendChild(p)
	doc.AppendChild(adm)

	got := renderHandBuiltDoc(t, doc)
	want := "<div class=\"admonition admonition-note\">\n<p class=\"admonition-title\">Note</p>\n<p>hello</p>\n</div>\n"
	if got != want {
		t.Fatalf("\n got %q\nwant %q", got, want)
	}
}

func TestRenderAdmonitionHostileVariantEscaped(t *testing.T) {
	// The parser constrains admonition variants, but hand-built documents
	// (and doc JSON via RenderDoc) can carry arbitrary variants. Both the
	// class attribute and the derived title must escape it.
	doc := &document.Document{}
	adm := &document.Admonition{Variant: `x"><script>alert(1)</script>`}
	p := &document.Paragraph{}
	p.AppendChild(&document.Text{Value: "hello"})
	adm.AppendChild(p)
	doc.AppendChild(adm)

	got := renderHandBuiltDoc(t, doc)
	if strings.Contains(got, "<script") {
		t.Fatalf("hostile variant produced a live <script tag:\n%s", got)
	}
	escaped := "&quot;&gt;&lt;script&gt;alert(1)&lt;/script&gt;"
	if !strings.Contains(got, `class="admonition admonition-x`+escaped+`"`) {
		t.Errorf("class attribute not escaped:\n%s", got)
	}
	if !strings.Contains(got, `<p class="admonition-title">X`+escaped+"</p>") {
		t.Errorf("title not escaped (title-casing preserved):\n%s", got)
	}
}

func TestRenderListWithNonListItemChildDoesNotPanic(t *testing.T) {
	doc := &document.Document{}
	list := &document.List{}
	p := &document.Paragraph{}
	p.AppendChild(&document.Text{Value: "stray"})
	list.AppendChild(p) // not a *document.ListItem
	doc.AppendChild(list)

	got := renderHandBuiltDoc(t, doc)
	want := "<ul>\n<p>stray</p>\n</ul>\n"
	if got != want {
		t.Fatalf("\n got %q\nwant %q", got, want)
	}
}

func TestRenderTableWithNonRowChildDoesNotPanic(t *testing.T) {
	doc := &document.Document{}
	tbl := &document.Table{}
	p := &document.Paragraph{}
	p.AppendChild(&document.Text{Value: "stray"})
	tbl.AppendChild(p) // not a *document.TableRow
	doc.AppendChild(tbl)

	got := renderHandBuiltDoc(t, doc)
	want := "<table>\n</table>\n"
	if got != want {
		t.Fatalf("\n got %q\nwant %q", got, want)
	}
}

func TestRenderTableRowWithNonCellChildDoesNotPanic(t *testing.T) {
	doc := &document.Document{}
	tbl := &document.Table{}
	row := &document.TableRow{}
	p := &document.Paragraph{}
	p.AppendChild(&document.Text{Value: "stray"})
	row.AppendChild(p) // not a *document.TableCell
	tbl.AppendChild(row)
	doc.AppendChild(tbl)

	got := renderHandBuiltDoc(t, doc)
	want := "<table>\n<tr>\n</tr>\n</table>\n"
	if got != want {
		t.Fatalf("\n got %q\nwant %q", got, want)
	}
}
