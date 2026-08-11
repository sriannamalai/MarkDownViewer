package boundary

import (
	"bytes"
	"encoding/json"
	"reflect"
	"regexp"
	"strings"
	"testing"

	"github.com/alecthomas/chroma/v2"
	"github.com/alecthomas/chroma/v2/styles"
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

// TestParseImplParserToggles drives every nested parser toggle through
// the boundary and asserts it changes parse output as documented: the
// default (all extensions on) document JSON carries the marker, the
// toggled-off parse does not. This pins the JSON-name → parser.Config
// field wiring, not just decoding.
func TestParseImplParserToggles(t *testing.T) {
	cases := []struct {
		key    string // nested JSON key set to false
		md     string
		marker string // present by default, absent with the toggle off
	}{
		{"tables", "| a |\n| --- |\n| b |\n", `"kind":"table"`},
		{"strikethrough", "~~x~~\n", `"kind":"strikethrough"`},
		{"taskLists", "- [x] done\n", `"task":true`},
		{"linkify", "visit https://example.com now\n", `"kind":"link"`},
		{"footnotes", "a[^1]\n\n[^1]: b\n", `"kind":"footnoteRef"`},
		{"definitionLists", "Term\n: def\n", `"kind":"definitionList"`},
		{"frontMatter", "---\ntitle: x\n---\n\nbody\n", `"meta"`},
		{"emoji", "hi :smile:\n", "\U0001F604"},
		{"wikiLinks", "[[Page]]\n", `"kind":"wikiLink"`},
		{"math", "$x^2$\n", `"kind":"mathInline"`},
		{"admonitions", "> [!NOTE]\n> hi\n", `"kind":"admonition"`},
	}
	for _, tc := range cases {
		t.Run(tc.key, func(t *testing.T) {
			def, err := Parse([]byte(tc.md), nil)
			if err != nil {
				t.Fatal(err)
			}
			if !bytes.Contains(def, []byte(tc.marker)) {
				t.Fatalf("default parse missing marker %q:\n%s", tc.marker, def)
			}
			off, err := Parse([]byte(tc.md), []byte(`{"parser": {"`+tc.key+`": false}}`))
			if err != nil {
				t.Fatal(err)
			}
			if bytes.Contains(off, []byte(tc.marker)) {
				t.Fatalf("%s=false still parses marker %q:\n%s", tc.key, tc.marker, off)
			}
		})
	}
}

func TestParseImplCommonmarkOnly(t *testing.T) {
	md := []byte("~~x~~ [[W]]\n\n| a |\n| --- |\n| b |\n")
	doc, err := Parse(md, []byte(`{"parser": {"commonmarkOnly": true}}`))
	if err != nil {
		t.Fatal(err)
	}
	for _, gone := range []string{`"kind":"strikethrough"`, `"kind":"wikiLink"`, `"kind":"table"`} {
		if bytes.Contains(doc, []byte(gone)) {
			t.Errorf("commonmarkOnly parse still contains %s:\n%s", gone, doc)
		}
	}
	// Tristate re-enable on the commonmarkOnly base.
	doc, err = Parse(md, []byte(`{"parser": {"commonmarkOnly": true, "tables": true}}`))
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(doc, []byte(`"kind":"table"`)) {
		t.Errorf("commonmarkOnly+tables should parse tables:\n%s", doc)
	}
	if bytes.Contains(doc, []byte(`"kind":"strikethrough"`)) {
		t.Errorf("commonmarkOnly+tables must not parse strikethrough:\n%s", doc)
	}
}

// TestParserOptionsAbsentIsByteIdentical pins the additive-change
// guarantee: payloads without the new fields (and their no-op spellings)
// produce byte-identical output to the pre-v0.9 defaults.
func TestParserOptionsAbsentIsByteIdentical(t *testing.T) {
	md := []byte("# Hi\n\n~~x~~ [[W]]\n")
	base, err := Render(md, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	for _, opts := range []string{`{}`, `{"parser": {}}`, `{"parser": null}`, `{"headingAnchors": true}`} {
		got, err := Render(md, []byte(opts), nil)
		if err != nil {
			t.Fatalf("Render(%s): %v", opts, err)
		}
		if !bytes.Equal(base, got) {
			t.Errorf("Render(%s) differs from default output", opts)
		}
	}
	pbase, err := Parse(md, nil)
	if err != nil {
		t.Fatal(err)
	}
	pgot, err := Parse(md, []byte(`{"parser": {}}`))
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(pbase, pgot) {
		t.Error(`Parse with {"parser": {}} differs from default parse`)
	}
}

func TestRenderImplParserConfig(t *testing.T) {
	html, err := Render([]byte("[[Wiki Page]]\n"), []byte(`{"fragment": true, "parser": {"wikiLinks": false}}`), nil)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(html, []byte("[[Wiki Page]]")) || bytes.Contains(html, []byte("<a ")) {
		t.Errorf("wikiLinks=false should render wiki syntax literally: %s", html)
	}
}

func TestRenderImplHeadingAnchors(t *testing.T) {
	md := []byte("# Hello\n")
	def, err := Render(md, []byte(`{"fragment": true}`), nil)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(def, []byte(`<h1 id="hello">`)) {
		t.Errorf("default should emit heading anchors: %s", def)
	}
	off, err := Render(md, []byte(`{"fragment": true, "headingAnchors": false}`), nil)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(off, []byte("<h1>")) || bytes.Contains(off, []byte("id=")) {
		t.Errorf("headingAnchors=false should omit heading ids: %s", off)
	}
	// Render-time toggle: it applies through RenderDoc too.
	doc, err := Parse(md, nil)
	if err != nil {
		t.Fatal(err)
	}
	viaDoc, err := RenderDoc(doc, []byte(`{"fragment": true, "headingAnchors": false}`), nil)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(off, viaDoc) {
		t.Error("headingAnchors=false differs between Render and RenderDoc")
	}
}

// RenderDoc decodes-and-ignores the parser object (the document is
// already parsed), per the options struct's documented precedent.
func TestRenderDocIgnoresParserConfig(t *testing.T) {
	doc, err := Parse([]byte("[[Wiki Page]]\n"), nil)
	if err != nil {
		t.Fatal(err)
	}
	plain, err := RenderDoc(doc, []byte(`{"fragment": true}`), nil)
	if err != nil {
		t.Fatal(err)
	}
	withParser, err := RenderDoc(doc, []byte(`{"fragment": true, "parser": {"wikiLinks": false}}`), nil)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(plain, withParser) {
		t.Error("parser options must not affect RenderDoc output")
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
		"theme-light.json":     {"--md-bg", `"version"`},
		"theme-dark.json":      {"--md-bg", `"version"`},
		"highlight-light.json": {`"colors"`, `"Keyword"`, `"version"`},
		"highlight-dark.json":  {`"colors"`, `"Keyword"`, `"version"`},
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

func TestAssetImplHighlightJSON(t *testing.T) {
	hexColor := regexp.MustCompile(`^#[0-9a-f]{6}$`)
	cases := []struct {
		name string
		want theme.Theme
	}{
		{"highlight-light.json", theme.Light()},
		{"highlight-dark.json", theme.Dark()},
	}
	for _, tc := range cases {
		got, err := Asset(tc.name)
		if err != nil {
			t.Fatalf("Asset(%q): %v", tc.name, err)
		}
		var decoded struct {
			Version int               `json:"version"`
			Style   string            `json:"style"`
			Colors  map[string]string `json:"colors"`
		}
		if err := json.Unmarshal(got, &decoded); err != nil {
			t.Fatalf("Asset(%q): not valid JSON: %v", tc.name, err)
		}
		if decoded.Version != 1 {
			t.Errorf("Asset(%q): version = %d, want 1", tc.name, decoded.Version)
		}
		if decoded.Style != tc.want.ChromaStyle {
			t.Errorf("Asset(%q): style = %q, want the theme's chroma style %q",
				tc.name, decoded.Style, tc.want.ChromaStyle)
		}
		if len(decoded.Colors) == 0 {
			t.Fatalf("Asset(%q): colors map is empty", tc.name)
		}
		for tt, color := range decoded.Colors {
			if !hexColor.MatchString(color) {
				t.Errorf("Asset(%q): colors[%q] = %q is not #rrggbb", tc.name, tt, color)
			}
		}
		// Cross-check against the style object — the single source of
		// truth both this asset and the chroma HTML CSS derive from: for
		// a handful of token types, the JSON color must equal the
		// foreground the HTML formatter resolves for the same style
		// (inherited entry minus the Background entry's fields).
		style := styles.Get(tc.want.ChromaStyle)
		bg := style.Get(chroma.Background)
		for _, tt := range []chroma.TokenType{chroma.Keyword, chroma.LiteralString, chroma.Comment, chroma.NameFunction} {
			entry := style.Get(tt).Sub(bg)
			if !entry.Colour.IsSet() {
				t.Fatalf("style %q: token %s has no foreground — pick a different cross-check token", tc.want.ChromaStyle, tt)
			}
			if got, want := decoded.Colors[tt.String()], entry.Colour.String(); got != want {
				t.Errorf("Asset(%q): colors[%q] = %q, want the style's %q", tc.name, tt.String(), got, want)
			}
		}
		// Every standard token type must be present exactly when the
		// style resolves a foreground for it (no stray keys) — the same
		// universe the chroma HTML formatter's WriteCSS walks, so the
		// asset covers exactly what the CSS covers.
		wantKeys := 0
		for tt := range chroma.StandardTypes {
			entry := style.Get(tt)
			if tt != chroma.Background {
				entry = entry.Sub(bg)
			}
			_, present := decoded.Colors[tt.String()]
			if entry.Colour.IsSet() != present {
				t.Errorf("Asset(%q): token %s present=%v, want %v", tc.name, tt.String(), present, entry.Colour.IsSet())
			}
			if entry.Colour.IsSet() {
				wantKeys++
			}
		}
		if len(decoded.Colors) != wantKeys {
			t.Errorf("Asset(%q): %d colors, want %d (stray keys?)", tc.name, len(decoded.Colors), wantKeys)
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
		for _, valid := range []string{"mermaid.js", "theme-dark.css", "theme-light.json", "theme-dark.json", "highlight-light.json", "highlight-dark.json"} {
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
