package htmlrender

import (
	"strings"
	"testing"

	"github.com/sriannamalai/markdownviewer/theme"
)

func TestHighlightCSS(t *testing.T) {
	light, err := HighlightCSS(theme.Light())
	if err != nil {
		t.Fatal(err)
	}
	dark, err := HighlightCSS(theme.Dark())
	if err != nil {
		t.Fatal(err)
	}
	for name, css := range map[string]string{"light": light, "dark": dark} {
		if !strings.Contains(css, ".chroma") {
			t.Errorf("%s: missing .chroma class prefix", name)
		}
		if !strings.Contains(css, "{") || !strings.Contains(css, "}") {
			t.Errorf("%s: does not look like CSS", name)
		}
	}
	if light == dark {
		t.Error("light and dark chroma CSS are identical")
	}
}

func TestHighlightCSSUnknownStyle(t *testing.T) {
	// chroma falls back to a default style for unknown names rather than
	// erroring; HighlightCSS must still return usable CSS, not fail.
	css, err := HighlightCSS(theme.Theme{Name: "custom", ChromaStyle: "no-such-style"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(css, ".chroma") {
		t.Error("fallback CSS missing .chroma")
	}
}
