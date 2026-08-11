package boundary

import (
	"bytes"
	"encoding/json"
	"reflect"
	"strings"
	"testing"

	htmlrender "github.com/sriannamalai/markdownviewer/render/html"
	"github.com/sriannamalai/markdownviewer/theme"
)

const sampleMD = "# Hello *world*\n\nSome `code` here.\n"

func TestRenderImpl(t *testing.T) {
	html, err := Render([]byte(sampleMD), nil, nil)
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
	html, err := Render([]byte(sampleMD), []byte(`{"fragment": true}`), nil)
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Contains(html, []byte("<html")) {
		t.Error("fragment output contains <html")
	}
}

func TestRenderImplBadOptions(t *testing.T) {
	if _, err := Render([]byte(sampleMD), []byte(`{"bogus": 1}`), nil); err == nil ||
		!strings.Contains(err.Error(), "bogus") {
		t.Fatalf("want unknown-field error, got %v", err)
	}
}

func TestRenderImplExtraCSS(t *testing.T) {
	html, err := Render([]byte(sampleMD), []byte(`{"extraCss": "body{font-size:117%}"}`), nil)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(html, []byte("body{font-size:117%}")) {
		t.Error("extraCss not present in rendered output")
	}
}

func TestRenderImplExtraCSSWrongCaseRejected(t *testing.T) {
	if _, err := Render([]byte(sampleMD), []byte(`{"extraCSS": "body{}"}`), nil); err == nil ||
		!strings.Contains(err.Error(), "extraCSS") {
		t.Fatalf("want unknown-field error for wrong-case extraCSS, got %v", err)
	}
}

func TestRenderImplCodeHeader(t *testing.T) {
	src := []byte("```shell\necho hi\n```\n")
	html, err := Render(src, []byte(`{"codeHeader": true}`), nil)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{`<div class="md-code">`, `<span class="md-code-lang">shell</span>`, "md-code-copy"} {
		if !bytes.Contains(html, []byte(want)) {
			t.Errorf("codeHeader output missing %q", want)
		}
	}
	plain, err := Render(src, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	// The base stylesheet always carries .md-code selectors; absence is
	// judged on the emitted markup, not the CSS.
	if bytes.Contains(plain, []byte(`<div class="md-code">`)) {
		t.Error("md-code wrapper must not appear without codeHeader")
	}
}

func TestParseImpl(t *testing.T) {
	doc, err := Parse([]byte(sampleMD), nil)
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
	if _, err := Parse([]byte(sampleMD), []byte(`{"version": 2}`)); err == nil {
		t.Fatal("want version error, got nil")
	}
}

func TestRenderDocRoundTrip(t *testing.T) {
	opts := []byte(`{"theme": "dark"}`)
	direct, err := Render([]byte(sampleMD), opts, nil)
	if err != nil {
		t.Fatal(err)
	}
	docJSON, err := Parse([]byte(sampleMD), nil)
	if err != nil {
		t.Fatal(err)
	}
	viaDoc, err := RenderDoc(docJSON, opts, nil)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(direct, viaDoc) {
		t.Error("render and parse->renderDoc outputs differ")
	}
}

// hostileAdmonitionDocJSON is the v0.8.1 XSS reproducer: valid v1 document
// JSON whose admonition variant carries an attribute-breakout script tag.
// RenderDoc accepts arbitrary doc JSON, so the renderer must escape the
// variant in both the class attribute and the derived title.
const hostileAdmonitionDocJSON = `{"version":1,"kind":"document","children":[{"kind":"admonition","variant":"x\"><script>alert(1)</script>","children":[{"kind":"paragraph","children":[{"kind":"text","value":"hi"}]}]}]}`

func TestRenderDocHostileVariantCannotInjectScript(t *testing.T) {
	html, err := RenderDoc([]byte(hostileAdmonitionDocJSON), []byte(`{"fragment": true}`), nil)
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Contains(bytes.ToLower(html), []byte("<script")) {
		t.Fatalf("hostile doc JSON produced a live <script tag:\n%s", html)
	}
	if !bytes.Contains(html, []byte("&lt;script&gt;alert(1)&lt;/script&gt;")) {
		t.Errorf("escaped variant text missing from output:\n%s", html)
	}
}

func TestRenderDocImplBadJSON(t *testing.T) {
	if _, err := RenderDoc([]byte(`{`), nil, nil); err == nil {
		t.Fatal("want document JSON error, got nil")
	}
}

func TestEmptyInputs(t *testing.T) {
	if _, err := Render(nil, nil, nil); err != nil {
		t.Errorf("Render(nil): %v", err)
	}
	if _, err := Parse(nil, nil); err != nil {
		t.Errorf("Parse(nil): %v", err)
	}
}

func TestAssetImpl(t *testing.T) {
	markers := map[string][]string{
		"mermaid.js":       {"mermaid"},
		"katex.js":         {"katex"},
		"katex.css":        {"@font-face", "data:font"},
		"base.css":         {"--md-max-width"},
		"theme-light.css":  {"--md-bg", ".chroma"},
		"theme-dark.css":   {"--md-bg", ".chroma"},
		"theme-light.json": {"--md-bg", `"version"`},
		"theme-dark.json":  {"--md-bg", `"version"`},
	}
	for name, wants := range markers {
		got, err := Asset(name)
		if err != nil {
			t.Fatalf("Asset(%q): %v", name, err)
		}
		if len(got) == 0 {
			t.Fatalf("Asset(%q): empty", name)
		}
		for _, w := range wants {
			if !bytes.Contains(bytes.ToLower(got), []byte(strings.ToLower(w))) {
				t.Errorf("Asset(%q): missing marker %q", name, w)
			}
		}
	}
}

func TestAssetImplThemesDiffer(t *testing.T) {
	light, err := Asset("theme-light.css")
	if err != nil {
		t.Fatal(err)
	}
	dark, err := Asset("theme-dark.css")
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Equal(light, dark) {
		t.Error("theme-light.css and theme-dark.css are identical")
	}
}

func TestAssetImplThemeJSON(t *testing.T) {
	cases := []struct {
		name string
		want theme.Theme
	}{
		{"theme-light.json", theme.Light()},
		{"theme-dark.json", theme.Dark()},
	}
	for _, tc := range cases {
		got, err := Asset(tc.name)
		if err != nil {
			t.Fatalf("Asset(%q): %v", tc.name, err)
		}
		var decoded struct {
			Version     int               `json:"version"`
			Mode        string            `json:"mode"`
			ChromaStyle string            `json:"chromaStyle"`
			Vars        map[string]string `json:"vars"`
		}
		if err := json.Unmarshal(got, &decoded); err != nil {
			t.Fatalf("Asset(%q): not valid JSON: %v", tc.name, err)
		}
		if decoded.Version != 1 {
			t.Errorf("Asset(%q): version = %d, want 1", tc.name, decoded.Version)
		}
		if decoded.Mode != tc.want.Name {
			t.Errorf("Asset(%q): mode = %q, want %q", tc.name, decoded.Mode, tc.want.Name)
		}
		if decoded.ChromaStyle != tc.want.ChromaStyle {
			t.Errorf("Asset(%q): chromaStyle = %q, want %q", tc.name, decoded.ChromaStyle, tc.want.ChromaStyle)
		}
		if !reflect.DeepEqual(decoded.Vars, tc.want.Vars) {
			t.Errorf("Asset(%q): vars = %v, want theme Vars %v", tc.name, decoded.Vars, tc.want.Vars)
		}
		// Deterministic: a second fetch is byte-identical.
		again, err := Asset(tc.name)
		if err != nil {
			t.Fatalf("Asset(%q) second call: %v", tc.name, err)
		}
		if !bytes.Equal(got, again) {
			t.Errorf("Asset(%q): two calls differ:\n%s\n%s", tc.name, got, again)
		}
	}
}

func TestAssetImplUnknown(t *testing.T) {
	for _, name := range []string{"", "bogus.js", "Mermaid.js"} {
		_, err := Asset(name)
		if err == nil {
			t.Fatalf("Asset(%q): want error", name)
		}
		for _, valid := range []string{"mermaid.js", "theme-dark.css", "theme-light.json", "theme-dark.json"} {
			if !strings.Contains(err.Error(), valid) {
				t.Errorf("Asset(%q): error does not list valid name %q: %v", name, valid, err)
			}
		}
	}
}

func TestRenderThreadsResolver(t *testing.T) {
	md := []byte("![a](img/x.png)\n\n[b](docs/y.md)\n")
	var kinds []htmlrender.ResolveKind
	var targets []string
	r := func(kind htmlrender.ResolveKind, target string) (string, bool) {
		kinds = append(kinds, kind)
		targets = append(targets, target)
		if kind == htmlrender.ResolveImage {
			return "asset://rewritten.png", true
		}
		return "", false
	}
	got, err := Render(md, []byte(`{"fragment": true}`), r)
	if err != nil {
		t.Fatal(err)
	}
	html := string(got)
	if !strings.Contains(html, `src="asset://rewritten.png"`) {
		t.Errorf("resolved image URL not emitted: %s", html)
	}
	if !strings.Contains(html, `href="docs/y.md"`) {
		t.Errorf("declined link did not take default resolution: %s", html)
	}
	if len(kinds) != 2 || kinds[0] != htmlrender.ResolveImage || kinds[1] != htmlrender.ResolveLink {
		t.Errorf("kinds = %v; want [image link]", kinds)
	}
	if targets[0] != "img/x.png" || targets[1] != "docs/y.md" {
		t.Errorf("targets = %v", targets)
	}
}

func TestRenderNilResolverUnchanged(t *testing.T) {
	md := []byte("[b](docs/y.md)\n")
	a, err := Render(md, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	// A resolver that always declines must be byte-identical to nil.
	decline := func(htmlrender.ResolveKind, string) (string, bool) { return "", false }
	b, err := Render(md, nil, decline)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(a, b) {
		t.Error("always-decline resolver changed output vs nil resolver")
	}
}

func TestRenderDocThreadsResolver(t *testing.T) {
	doc, err := Parse([]byte("![a](img/x.png)\n"), nil)
	if err != nil {
		t.Fatal(err)
	}
	r := func(kind htmlrender.ResolveKind, target string) (string, bool) {
		return "asset://rewritten.png", true
	}
	got, err := RenderDoc(doc, []byte(`{"fragment": true}`), r)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(got), `src="asset://rewritten.png"`) {
		t.Errorf("RenderDoc did not thread resolver: %s", got)
	}
}
