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
