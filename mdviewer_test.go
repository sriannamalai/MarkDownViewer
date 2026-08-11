package markdownviewer

import (
	"bytes"
	"context"
	"strings"
	"testing"
	"time"

	"github.com/sriannamalai/markdownviewer/document"
	"github.com/sriannamalai/markdownviewer/parser"
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

func TestWithSourceMap(t *testing.T) {
	out, err := Render([]byte("# Hi\n"), Fragment(), WithSourceMap())
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(out), `data-md-line="1"`) {
		t.Fatalf("got %q", out)
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

// TestResolverKindExported is a compile-level guard: ResolveKind must stay
// exported from the facade so hosts can write an explicitly-typed resolver
// func literal (Go func literals require named parameter types — a bare
// package-qualified constant isn't enough). If ResolveKind regresses to
// unexported, this test file fails to compile.
func TestResolverKindExported(t *testing.T) {
	var called bool
	resolver := func(kind ResolveKind, target string) (string, bool) {
		called = true
		if kind == ResolveImage {
			return "https://cdn.example.com/" + target, true
		}
		return "", false
	}
	out, err := Render([]byte("![alt](pic.png)\n"), Fragment(), WithResolver(resolver))
	if err != nil {
		t.Fatal(err)
	}
	if !called {
		t.Fatal("resolver was not invoked")
	}
	if !strings.Contains(string(out), `src="https://cdn.example.com/pic.png"`) {
		t.Fatalf("resolver rewrite not applied, got %q", out)
	}
}

func TestThemeCustomization(t *testing.T) {
	out, _ := Render(
		[]byte("# Hi\n"),
		WithThemeOverrides(map[string]string{"--md-bg": "#f0f0f0"}),
		WithStylesheet(".custom { color: blue; }"),
	)
	s := string(out)
	if !strings.Contains(s, ":root{--md-bg:#f0f0f0;") {
		t.Fatal("WithThemeOverrides not applied")
	}
	if !strings.Contains(s, ".custom { color: blue; }") {
		t.Fatal("WithStylesheet not applied")
	}
}

func TestWithExtraCSS(t *testing.T) {
	out, _ := Render(
		[]byte("# Hi\n"),
		WithExtraCSS(".host { font-size: 117%; }"),
	)
	s := string(out)
	extra := strings.Index(s, ".host { font-size: 117%; }")
	if extra == -1 {
		t.Fatal("WithExtraCSS not applied")
	}
	base := strings.Index(s, "body.markdown-body {")
	if base == -1 || extra < base {
		t.Fatalf("WithExtraCSS must append after base CSS, not replace it (base=%d extra=%d)", base, extra)
	}
}

// TestDefaultLayoutIsFluid is the v0.3 behavior-change guard: the page no
// longer carries a fixed max-width by default (theme/base.css now reads
// "max-width: var(--md-max-width, none)"), so a plain Render has no
// --md-max-width override and the property resolves to "none".
func TestDefaultLayoutIsFluid(t *testing.T) {
	out, err := Render([]byte("# Hi\n"))
	if err != nil {
		t.Fatal(err)
	}
	s := string(out)
	if strings.Contains(s, "--md-max-width:") {
		t.Fatalf("default render should not set --md-max-width, got %q", s)
	}
	if !strings.Contains(s, "max-width: var(--md-max-width, none)") {
		t.Fatalf("base.css should fall back to a fluid (none) max-width, got %q", s)
	}
}

func TestWithMaxWidth(t *testing.T) {
	out, err := Render([]byte("# Hi\n"), WithMaxWidth("700px"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(out), "--md-max-width:700px;") {
		t.Fatalf("got %q", out)
	}
}

func TestWithMaxWidthEmptyStaysFluid(t *testing.T) {
	out, err := Render([]byte("# Hi\n"), WithMaxWidth(""))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(out), "--md-max-width:") {
		t.Fatalf("empty WithMaxWidth should leave the page fluid, got %q", out)
	}
}

// TestWithMaxWidthRejectsHostileValue is the neutralization test for issue
// 3's validation requirement: width flows into a CSS custom-property value
// position inside the page's <style> element, so a value that could close
// the declaration early (';' or '}') is rejected outright rather than
// emitted defanged — the option no-ops and the page stays at its otherwise
// configured width (fluid, here).
func TestWithMaxWidthRejectsHostileValue(t *testing.T) {
	hostile := "800px}body{display:none"
	out, err := Render([]byte("# Hi\n"), WithMaxWidth(hostile))
	if err != nil {
		t.Fatal(err)
	}
	s := string(out)
	if strings.Contains(s, "display:none") || strings.Contains(s, "800px}body") {
		t.Fatalf("hostile width value was not neutralized, got %q", s)
	}
	if strings.Contains(s, "--md-max-width:") {
		t.Fatalf("hostile width value should have been rejected entirely, got %q", s)
	}
}

func TestWithMaxWidthRejectsSemicolon(t *testing.T) {
	out, err := Render([]byte("# Hi\n"), WithMaxWidth("860px;--md-bg:red"))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(out), "--md-max-width:") {
		t.Fatalf("semicolon-bearing width value should have been rejected, got %q", out)
	}
}

// TestBOMPrefixedInput verifies a leading UTF-8 BOM doesn't defeat heading
// recognition through the facade (the CLI and any host that reads a file
// verbatim go through this path).
func TestBOMPrefixedInput(t *testing.T) {
	out, err := Render([]byte("\xEF\xBB\xBF# Hi\n"), Fragment())
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(out), `<h1 id="hi">Hi</h1>`) {
		t.Fatalf("BOM not stripped before parsing, got %q", out)
	}
}

func TestRenderDocParity(t *testing.T) {
	src := []byte("# Hi\n\n*em* [[W]]\n")
	direct, err := Render(src, Fragment())
	if err != nil {
		t.Fatal(err)
	}
	doc, err := Parse(src)
	if err != nil {
		t.Fatal(err)
	}
	viaDoc, err := RenderDoc(doc, Fragment())
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(direct, viaDoc) {
		t.Fatalf("parity broken:\n%s\nvs\n%s", direct, viaDoc)
	}
}

func TestWithParserConfig(t *testing.T) {
	cfg := parser.Default()
	cfg.WikiLinks = false
	out, err := Render([]byte("[[NotALink]]\n"), Fragment(), WithParserConfig(cfg))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(out), "wikilink") {
		t.Fatalf("wikilinks should be disabled: %q", out)
	}
}

func TestFacadeParseWith(t *testing.T) {
	doc, err := ParseWith([]byte("~~x~~\n"), parser.CommonMarkOnly())
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(document.Dump(doc), "Strikethrough") {
		t.Fatal("CommonMarkOnly must not parse strikethrough")
	}
}

func TestParseContextCancelled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := ParseContext(ctx, []byte("# Hi\n")); err != context.Canceled {
		t.Fatalf("want context.Canceled, got %v", err)
	}
}

func TestRenderContextNormal(t *testing.T) {
	want, _ := Render([]byte("# Hi\n"), Fragment())
	got, err := RenderContext(context.Background(), []byte("# Hi\n"), Fragment())
	if err != nil || !bytes.Equal(want, got) {
		t.Fatalf("err=%v equal=%t", err, bytes.Equal(want, got))
	}
}

func TestRenderDocContextNoLateWrites(t *testing.T) {
	doc, _ := Parse([]byte("# Hi\n"))
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	var buf bytes.Buffer
	if err := RenderDocContext(ctx, &buf, doc, Fragment()); err != context.Canceled {
		t.Fatalf("want Canceled, got %v", err)
	}
	time.Sleep(50 * time.Millisecond) // abandoned goroutine must not touch buf
	if buf.Len() != 0 {
		t.Fatalf("late write after cancellation: %q", buf.String())
	}
}
