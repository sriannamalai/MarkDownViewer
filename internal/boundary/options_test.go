package boundary

import (
	"strings"
	"testing"
)

func TestDecodeOptionsDefaults(t *testing.T) {
	for _, src := range [][]byte{nil, []byte(""), []byte("  \n")} {
		o, err := decodeOptions(src)
		if err != nil {
			t.Fatalf("decodeOptions(%q): %v", src, err)
		}
		if o.Theme != "auto" || !o.Mermaid || !o.Math || !o.Highlighting {
			t.Errorf("defaults wrong: %+v", o)
		}
		if o.Fragment || o.AllowRawHTML || o.SourceMap || o.MaxWidth != "" || o.Stylesheet != "" || o.ExtraCSS != "" {
			t.Errorf("zero-value fields wrong: %+v", o)
		}
	}
}

func TestDecodeOptionsRoundTrip(t *testing.T) {
	o, err := decodeOptions([]byte(`{
		"version": 1, "theme": "dark", "fragment": true, "allowRawHTML": true,
		"mermaid": false, "math": false, "highlighting": false,
		"maxWidth": "70ch", "sourceMap": true,
		"themeOverrides": {"--md-bg": "#123"}, "stylesheet": "body{}",
		"extraCss": ".x{}"
	}`))
	if err != nil {
		t.Fatal(err)
	}
	if o.Theme != "dark" || !o.Fragment || !o.AllowRawHTML || o.Mermaid || o.Math ||
		o.Highlighting || o.MaxWidth != "70ch" || !o.SourceMap ||
		o.ThemeOverrides["--md-bg"] != "#123" || o.Stylesheet != "body{}" ||
		o.ExtraCSS != ".x{}" {
		t.Errorf("decoded wrong: %+v", o)
	}
}

func TestDecodeOptionsUnknownField(t *testing.T) {
	_, err := decodeOptions([]byte(`{"them": "dark"}`))
	if err == nil || !strings.Contains(err.Error(), "them") {
		t.Fatalf("want error naming unknown field, got %v", err)
	}
}

func TestDecodeOptionsBadVersion(t *testing.T) {
	_, err := decodeOptions([]byte(`{"version": 2}`))
	if err == nil || !strings.Contains(err.Error(), "version") {
		t.Fatalf("want version error, got %v", err)
	}
}

func TestDecodeOptionsTypeMismatch(t *testing.T) {
	_, err := decodeOptions([]byte(`{"fragment": "yes"}`))
	if err == nil {
		t.Fatal("want type error, got nil")
	}
}

func TestDecodeOptionsBadJSON(t *testing.T) {
	_, err := decodeOptions([]byte(`{`))
	if err == nil {
		t.Fatal("want syntax error, got nil")
	}
}

func TestToFacadeOptionsCount(t *testing.T) {
	// Defaults map to just WithTheme("auto").
	if n := len(defaultOptions().toFacadeOptions(nil)); n != 1 {
		t.Errorf("defaults: want 1 option, got %d", n)
	}
	// Everything toggled maps to all 11 constructors: WithTheme, Fragment,
	// AllowRawHTML, DisableMermaid, DisableMath, DisableHighlighting,
	// WithSourceMap, WithThemeOverrides, WithMaxWidth, WithStylesheet,
	// WithExtraCSS (Mermaid/Math/Highlighting are zero-value false here).
	o := options{Theme: "dark", Fragment: true, AllowRawHTML: true,
		MaxWidth: "70ch", SourceMap: true,
		ThemeOverrides: map[string]string{"--md-bg": "#123"}, Stylesheet: "body{}",
		ExtraCSS: ".x{}"}
	if n := len(o.toFacadeOptions(nil)); n != 11 {
		t.Errorf("full: want 11 options, got %d", n)
	}
	// Empty theme is skipped.
	if n := len(options{Mermaid: true, Math: true, Highlighting: true}.toFacadeOptions(nil)); n != 0 {
		t.Errorf("empty theme: want 0 options, got %d", n)
	}
}

func TestDecodeOptionsTrailingData(t *testing.T) {
	_, err := decodeOptions([]byte(`{"theme": "dark"} {"bogus": 1}`))
	if err == nil || !strings.Contains(err.Error(), "trailing") {
		t.Fatalf("want trailing-data error, got %v", err)
	}
	if _, err := decodeOptions([]byte("{\"theme\": \"dark\"}\n  ")); err != nil {
		t.Fatalf("trailing whitespace should be fine: %v", err)
	}
}
