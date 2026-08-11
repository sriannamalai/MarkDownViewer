package boundary

import (
	"strings"
	"testing"

	"github.com/sriannamalai/markdownviewer/parser"
)

func TestDecodeOptionsDefaults(t *testing.T) {
	for _, src := range [][]byte{nil, []byte(""), []byte("  \n")} {
		o, err := decodeOptions(src)
		if err != nil {
			t.Fatalf("decodeOptions(%q): %v", src, err)
		}
		if o.Theme != "auto" || !o.Mermaid || !o.Math || !o.Highlighting || !o.HeadingAnchors {
			t.Errorf("defaults wrong: %+v", o)
		}
		if o.Fragment || o.AllowRawHTML || o.SourceMap || o.MaxWidth != "" || o.Stylesheet != "" || o.ExtraCSS != "" || o.CodeHeader || o.Parser != nil {
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
		"extraCss": ".x{}", "codeHeader": true, "headingAnchors": false,
		"parser": {"commonmarkOnly": true, "tables": true, "wikiLinks": false}
	}`))
	if err != nil {
		t.Fatal(err)
	}
	if o.Theme != "dark" || !o.Fragment || !o.AllowRawHTML || o.Mermaid || o.Math ||
		o.Highlighting || o.MaxWidth != "70ch" || !o.SourceMap ||
		o.ThemeOverrides["--md-bg"] != "#123" || o.Stylesheet != "body{}" ||
		o.ExtraCSS != ".x{}" || !o.CodeHeader || o.HeadingAnchors {
		t.Errorf("decoded wrong: %+v", o)
	}
	p := o.Parser
	if p == nil || !p.CommonmarkOnly || p.Tables == nil || !*p.Tables ||
		p.WikiLinks == nil || *p.WikiLinks || p.Strikethrough != nil {
		t.Errorf("nested parser decoded wrong: %+v", p)
	}
	// The tristate folds as documented: commonmarkOnly base, tables
	// re-enabled, wikiLinks off (already off on that base), the rest at
	// the base's (all-off) setting.
	cfg := p.toConfig()
	if !cfg.Tables || cfg.WikiLinks || cfg.Strikethrough || cfg.Math || cfg.FrontMatter {
		t.Errorf("toConfig folded wrong: %+v", cfg)
	}
}

func TestDecodeOptionsUnknownField(t *testing.T) {
	_, err := decodeOptions([]byte(`{"them": "dark"}`))
	if err == nil || !strings.Contains(err.Error(), "them") {
		t.Fatalf("want error naming unknown field, got %v", err)
	}
}

func TestDecodeOptionsNestedParserStrict(t *testing.T) {
	// Unknown nested key errors, naming the nested path.
	_, err := decodeOptions([]byte(`{"parser": {"bogus": true}}`))
	if err == nil || !strings.Contains(err.Error(), `"parser.bogus"`) {
		t.Errorf("unknown nested key: want error naming parser.bogus, got %v", err)
	}
	// Wrong-case nested key is rejected exact-case, naming the key.
	_, err = decodeOptions([]byte(`{"parser": {"wikilinks": false}}`))
	if err == nil || !strings.Contains(err.Error(), `"parser.wikilinks"`) {
		t.Errorf("wrong-case nested key: want error naming parser.wikilinks, got %v", err)
	}
	// Non-object "parser" values are rejected with a usable message.
	for _, src := range []string{`{"parser": 5}`, `{"parser": "x"}`, `{"parser": [1]}`, `{"parser": true}`} {
		if _, err := decodeOptions([]byte(src)); err == nil || !strings.Contains(err.Error(), "parser") {
			t.Errorf("decodeOptions(%s): want must-be-object error, got %v", src, err)
		}
	}
	// JSON null is the same as absent: defaults, no error.
	o, err := decodeOptions([]byte(`{"parser": null}`))
	if err != nil || o.Parser != nil {
		t.Errorf(`{"parser": null}: want nil Parser and no error, got %+v, %v`, o.Parser, err)
	}
	// Nested type mismatch is still an error (strict decode).
	if _, err := decodeOptions([]byte(`{"parser": {"tables": "yes"}}`)); err == nil {
		t.Error(`{"parser": {"tables": "yes"}}: want type error, got nil`)
	}
}

func TestDecodeOptionsEmptyParserObjectIsDefault(t *testing.T) {
	o, err := decodeOptions([]byte(`{"parser": {}}`))
	if err != nil {
		t.Fatal(err)
	}
	if o.Parser == nil {
		t.Fatal("empty parser object should decode to a non-nil parserOptions")
	}
	if got := o.Parser.toConfig(); got != parser.Default() {
		t.Errorf("empty parser object config = %+v, want parser.Default()", got)
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
	// Everything toggled maps to all 14 constructors: WithTheme, Fragment,
	// AllowRawHTML, DisableMermaid, DisableMath, DisableHighlighting,
	// DisableHeadingAnchors, WithSourceMap, WithThemeOverrides,
	// WithMaxWidth, WithStylesheet, WithExtraCSS, WithCodeHeader,
	// WithParserConfig (Mermaid/Math/Highlighting/HeadingAnchors are
	// zero-value false here).
	o := options{Theme: "dark", Fragment: true, AllowRawHTML: true,
		MaxWidth: "70ch", SourceMap: true,
		ThemeOverrides: map[string]string{"--md-bg": "#123"}, Stylesheet: "body{}",
		ExtraCSS: ".x{}", CodeHeader: true, Parser: &parserOptions{}}
	if n := len(o.toFacadeOptions(nil)); n != 14 {
		t.Errorf("full: want 14 options, got %d", n)
	}
	// Empty theme is skipped.
	if n := len(options{Mermaid: true, Math: true, Highlighting: true, HeadingAnchors: true}.toFacadeOptions(nil)); n != 0 {
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
