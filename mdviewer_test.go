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
