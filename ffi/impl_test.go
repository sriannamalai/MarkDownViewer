package main

import (
	"bytes"
	"errors"
	"strings"
	"testing"
)

const sampleMD = "# Hello *world*\n\nSome `code` here.\n"

func TestRenderImpl(t *testing.T) {
	html, err := renderImpl([]byte(sampleMD), nil)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"<h1", "Hello", "<em>world</em>", "<code>code</code>"} {
		if !bytes.Contains(html, []byte(want)) {
			t.Errorf("output missing %q", want)
		}
	}
}

func TestRenderImplFragment(t *testing.T) {
	html, err := renderImpl([]byte(sampleMD), []byte(`{"fragment": true}`))
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Contains(html, []byte("<html")) {
		t.Error("fragment output contains <html")
	}
}

func TestRenderImplBadOptions(t *testing.T) {
	if _, err := renderImpl([]byte(sampleMD), []byte(`{"bogus": 1}`)); err == nil ||
		!strings.Contains(err.Error(), "bogus") {
		t.Fatalf("want unknown-field error, got %v", err)
	}
}

func TestParseImpl(t *testing.T) {
	doc, err := parseImpl([]byte(sampleMD), nil)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{`"version":1`, `"heading"`} {
		if !bytes.Contains(doc, []byte(want)) {
			t.Errorf("document JSON missing %q in %s", want, doc)
		}
	}
}

func TestParseImplValidatesOptions(t *testing.T) {
	if _, err := parseImpl([]byte(sampleMD), []byte(`{"version": 2}`)); err == nil {
		t.Fatal("want version error, got nil")
	}
}

func TestRenderDocRoundTrip(t *testing.T) {
	opts := []byte(`{"theme": "dark"}`)
	direct, err := renderImpl([]byte(sampleMD), opts)
	if err != nil {
		t.Fatal(err)
	}
	docJSON, err := parseImpl([]byte(sampleMD), nil)
	if err != nil {
		t.Fatal(err)
	}
	viaDoc, err := renderDocImpl(docJSON, opts)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(direct, viaDoc) {
		t.Error("render and parse->renderDoc outputs differ")
	}
}

func TestRenderDocImplBadJSON(t *testing.T) {
	if _, err := renderDocImpl([]byte(`{`), nil); err == nil {
		t.Fatal("want document JSON error, got nil")
	}
}

func TestEmptyInputs(t *testing.T) {
	if _, err := renderImpl(nil, nil); err != nil {
		t.Errorf("renderImpl(nil): %v", err)
	}
	if _, err := parseImpl(nil, nil); err != nil {
		t.Errorf("parseImpl(nil): %v", err)
	}
}

func TestPanicError(t *testing.T) {
	if got := panicError("boom").Error(); got != "panic: boom" {
		t.Errorf("panicError(string) = %q", got)
	}
	wrapped := errors.New("boom")
	err := panicError(wrapped)
	if !errors.Is(err, wrapped) {
		t.Errorf("panicError(error) = %v, does not wrap the panic value", err)
	}
}

func TestAssetImpl(t *testing.T) {
	markers := map[string][]string{
		"mermaid.js":      {"mermaid"},
		"katex.js":        {"katex"},
		"katex.css":       {"@font-face", "data:font"},
		"base.css":        {"--md-max-width"},
		"theme-light.css": {"--md-bg", ".chroma"},
		"theme-dark.css":  {"--md-bg", ".chroma"},
	}
	for name, wants := range markers {
		got, err := assetImpl(name)
		if err != nil {
			t.Fatalf("assetImpl(%q): %v", name, err)
		}
		if len(got) == 0 {
			t.Fatalf("assetImpl(%q): empty", name)
		}
		for _, w := range wants {
			if !bytes.Contains(bytes.ToLower(got), []byte(strings.ToLower(w))) {
				t.Errorf("assetImpl(%q): missing marker %q", name, w)
			}
		}
	}
}

func TestAssetImplThemesDiffer(t *testing.T) {
	light, err := assetImpl("theme-light.css")
	if err != nil {
		t.Fatal(err)
	}
	dark, err := assetImpl("theme-dark.css")
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Equal(light, dark) {
		t.Error("theme-light.css and theme-dark.css are identical")
	}
}

func TestAssetImplUnknown(t *testing.T) {
	for _, name := range []string{"", "bogus.js", "Mermaid.js"} {
		_, err := assetImpl(name)
		if err == nil {
			t.Fatalf("assetImpl(%q): want error", name)
		}
		if !strings.Contains(err.Error(), "mermaid.js") || !strings.Contains(err.Error(), "theme-dark.css") {
			t.Errorf("assetImpl(%q): error does not list valid names: %v", name, err)
		}
	}
}
