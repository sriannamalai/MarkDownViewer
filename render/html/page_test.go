package htmlrender

import (
	"errors"
	"strings"
	"testing"

	"github.com/sriannamalai/markdownviewer/document"
	"github.com/sriannamalai/markdownviewer/parser"
)

type failAfterWriter struct {
	bytesWritten int
	failAfter    int
}

func (w *failAfterWriter) Write(p []byte) (int, error) {
	if w.bytesWritten >= w.failAfter {
		return 0, errors.New("write failed")
	}
	avail := w.failAfter - w.bytesWritten
	if len(p) > avail {
		p = p[:avail]
	}
	w.bytesWritten += len(p)
	return len(p), nil
}

func renderDoc(t *testing.T, md string) *document.Document {
	t.Helper()
	parsed, err := parser.Parse([]byte(md))
	if err != nil {
		t.Fatal(err)
	}
	return parsed
}

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

func TestThemeOverridesInLightMode(t *testing.T) {
	got := render(t, "# Hi\n", func(o *Options) {
		o.Fragment = false
		o.ThemeName = "light"
		o.ThemeOverrides = map[string]string{"--md-bg": "#f0f0f0"}
	})
	if !strings.Contains(got, ":root{--md-bg:#f0f0f0;") {
		t.Error("light mode should emit theme override block")
	}
}

func TestThemeOverridesAppearsAfterThemeAndInDarkBlock(t *testing.T) {
	got := render(t, "# Hi\n", func(o *Options) {
		o.Fragment = false
		o.ThemeName = "auto"
		o.ThemeOverrides = map[string]string{"--md-fg": "#000000"}
	})
	// Check that override appears after light theme :root block
	rootIdx := strings.Index(got, ":root{--md-")
	if rootIdx == -1 {
		t.Fatalf("light theme :root block not found")
	}
	overrideIdx := strings.Index(got, ":root{--md-fg:#000000;}")
	if overrideIdx == -1 || overrideIdx <= rootIdx {
		t.Error("override should appear after light theme block")
	}
	// Check that override appears inside dark media query
	darkMediaIdx := strings.Index(got, "@media (prefers-color-scheme: dark){")
	darkMediaEndIdx := strings.LastIndex(got[darkMediaIdx:], "}")
	if darkMediaIdx == -1 || darkMediaEndIdx == -1 {
		t.Fatalf("dark media block not found")
	}
	darkMediaContent := got[darkMediaIdx : darkMediaIdx+darkMediaEndIdx]
	if !strings.Contains(darkMediaContent, ":root{--md-fg:#000000;}") {
		t.Error("override should appear inside dark media block")
	}
}

func TestCustomStylesheetReplacesBaseCSS(t *testing.T) {
	customCSS := ".custom-class { color: red; }"
	got := render(t, "# Hi\n", func(o *Options) {
		o.Fragment = false
		o.ThemeName = "light"
		o.Stylesheet = customCSS
	})
	if !strings.Contains(got, customCSS) {
		t.Error("custom stylesheet should be present")
	}
	if strings.Contains(got, "max-width: 860px") {
		t.Error("base.css should not be present when custom stylesheet is used")
	}
}

func TestFragmentModeIgnoresCSS(t *testing.T) {
	got := render(t, "# Hi\n", func(o *Options) {
		o.Fragment = true
		o.ThemeOverrides = map[string]string{"--md-bg": "#f0f0f0"}
	})
	if strings.Contains(got, "<style>") {
		t.Error("fragment mode should not include style tags")
	}
	if strings.Contains(got, ":root{") {
		t.Error("fragment mode should not include theme overrides")
	}
}

func TestStylesheetBreakoutPrevention(t *testing.T) {
	injected := "</style><script>alert(1)</script>"
	got := render(t, "# Hi\n", func(o *Options) {
		o.Fragment = false
		o.Stylesheet = injected
	})
	// Verify exactly one </style> (the real one)
	if strings.Count(got, "</style>") != 1 {
		t.Error("page should contain exactly one </style> closing tag")
	}
	// Verify the breakout sequence is removed
	if strings.Contains(got, "</style><script>alert") {
		t.Error("style-element breakout sequence should not be present")
	}
}

func TestThemeOverridesBreakoutPrevention(t *testing.T) {
	injected := "red}</style><script>x"
	got := render(t, "# Hi\n", func(o *Options) {
		o.Fragment = false
		o.ThemeName = "light"
		o.ThemeOverrides = map[string]string{"--md-fg": injected}
	})
	// Verify exactly one </style> (the real one)
	if strings.Count(got, "</style>") != 1 {
		t.Error("page should contain exactly one </style> closing tag")
	}
	// Verify the breakout sequence is removed
	if strings.Contains(got, "</style><script>") {
		t.Error("style-element breakout sequence should not be present")
	}
	// Verify override value has breakout sequence removed but CSS value preserved
	if !strings.Contains(got, "--md-fg:red") {
		t.Error("override value base should be preserved")
	}
}

func TestThemeOverrideKeyValidation(t *testing.T) {
	got := render(t, "# Hi\n", func(o *Options) {
		o.Fragment = false
		o.ThemeName = "light"
		o.ThemeOverrides = map[string]string{
			"--md</style><script>x": "red",  // malicious key
			"--md-valid-key":        "blue", // valid key
		}
	})
	// Verify exactly one </style> (the real one)
	if strings.Count(got, "</style>") != 1 {
		t.Error("page should contain exactly one </style> closing tag")
	}
	// Verify no script tags
	if strings.Contains(got, "<script>") {
		t.Error("no script tags should be present")
	}
	// Verify malicious key text does not appear
	if strings.Contains(got, "--md</style><script>") {
		t.Error("malicious key should not appear in output")
	}
	// Verify valid key is emitted
	if !strings.Contains(got, "--md-valid-key:blue") {
		t.Error("valid key should be emitted")
	}
}

func TestRenderPageWriteError(t *testing.T) {
	doc := renderDoc(t, "# Hi\n")
	opts := Options{Fragment: false, ThemeName: "light"}
	// Fail very early, during the doctype write
	fw := &failAfterWriter{failAfter: 1}
	err := renderPage(fw, doc, opts)
	if err == nil {
		t.Fatal("expected error when write fails")
	}
	if err.Error() != "write failed" {
		t.Errorf("expected 'write failed' error, got %q", err)
	}
}
