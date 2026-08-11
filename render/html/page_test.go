package htmlrender

import (
	"errors"
	"regexp"
	"strings"
	"testing"

	"github.com/sriannamalai/markdownviewer/document"
	"github.com/sriannamalai/markdownviewer/parser"
)

// hexColorRe finds CSS hex colors, used to pull sentinel colors out of a
// chroma style's generated CSS without hardcoding palette values that chroma
// (or the styles it ships) could change out from under us.
var hexColorRe = regexp.MustCompile(`#[0-9a-fA-F]{6}`)

// sentinelColors returns up to n distinct hex colors found in css, in
// first-seen order.
func sentinelColors(t *testing.T, css string, n int) []string {
	t.Helper()
	seen := map[string]bool{}
	var out []string
	for _, m := range hexColorRe.FindAllString(css, -1) {
		if seen[m] {
			continue
		}
		seen[m] = true
		out = append(out, m)
		if len(out) == n {
			return out
		}
	}
	t.Fatalf("fewer than %d distinct hex colors found in css: %q", n, css)
	return nil
}

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

func TestExtraCSSAppendsAfterBase(t *testing.T) {
	got := render(t, "# Hi\n", func(o *Options) {
		o.Fragment = false
		o.ThemeName = "light"
		o.ExtraCSS = "body{font-size:117%}"
	})
	base := strings.Index(got, "body.markdown-body {")
	extra := strings.Index(got, "body{font-size:117%}")
	if base == -1 || extra == -1 || extra < base {
		t.Fatalf("extraCss must be appended after base CSS (base=%d extra=%d)", base, extra)
	}
}

func TestExtraCSSAppendsAfterCustomStylesheet(t *testing.T) {
	got := render(t, "# Hi\n", func(o *Options) {
		o.Fragment = false
		o.ThemeName = "light"
		o.Stylesheet = ".custom{}"
		o.ExtraCSS = ".extra{}"
	})
	if strings.Contains(got, "body.markdown-body {") {
		t.Error("stylesheet must still replace base CSS when extraCss is set")
	}
	custom := strings.Index(got, ".custom{}")
	extra := strings.Index(got, ".extra{}")
	if custom == -1 || extra == -1 || extra < custom {
		t.Fatalf("extraCss must come after the custom stylesheet (custom=%d extra=%d)", custom, extra)
	}
}

func TestExtraCSSBreakoutPrevention(t *testing.T) {
	got := render(t, "# Hi\n", func(o *Options) {
		o.Fragment = false
		o.ExtraCSS = "x{}</style><script>alert(1)</script>"
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

// codeHeavyMD is a small code-heavy fixture: several plain identifiers
// (chroma class "nx", e.g. "fmt", "os") alongside keywords/strings, since the
// dark-mode invisible-text bug was specifically about token classes one
// chroma style leaves unstyled while the other styles explicitly.
const codeHeavyMD = "```go\npackage main\n\nimport \"fmt\"\n\nfunc main() {\n\tfmt.Println(os.Args)\n}\n```\n"

func TestDarkPageContainsGithubDarkPalette(t *testing.T) {
	darkCSS, err := chromaCSS("github-dark")
	if err != nil {
		t.Fatal(err)
	}
	sentinels := sentinelColors(t, darkCSS, 2)

	got := render(t, codeHeavyMD, func(o *Options) { o.Fragment = false; o.ThemeName = "dark"; o.Highlight = true })
	for _, c := range sentinels {
		if !strings.Contains(got, c) {
			t.Errorf("dark page missing github-dark sentinel color %s", c)
		}
	}
}

func TestLightPageContainsGithubPalette(t *testing.T) {
	lightCSS, err := chromaCSS("github")
	if err != nil {
		t.Fatal(err)
	}
	sentinels := sentinelColors(t, lightCSS, 2)

	got := render(t, codeHeavyMD, func(o *Options) { o.Fragment = false; o.ThemeName = "light"; o.Highlight = true })
	for _, c := range sentinels {
		if !strings.Contains(got, c) {
			t.Errorf("light page missing github sentinel color %s", c)
		}
	}
}

func TestAutoPageContainsBothPalettesDarkInsideMediaQuery(t *testing.T) {
	lightCSS, err := chromaCSS("github")
	if err != nil {
		t.Fatal(err)
	}
	darkCSS, err := chromaCSS("github-dark")
	if err != nil {
		t.Fatal(err)
	}
	lightSentinel := sentinelColors(t, lightCSS, 1)[0]
	darkSentinel := sentinelColors(t, darkCSS, 1)[0]

	got := render(t, codeHeavyMD, func(o *Options) { o.Fragment = false; o.ThemeName = "auto"; o.Highlight = true })

	mediaIdx := strings.Index(got, "@media (prefers-color-scheme: dark){")
	if mediaIdx == -1 {
		t.Fatalf("no dark media query found")
	}
	closeIdx := strings.Index(got[mediaIdx:], "</style>")
	if closeIdx == -1 {
		t.Fatalf("no </style> after media query")
	}
	mediaBlock := got[mediaIdx : mediaIdx+closeIdx]

	if !strings.Contains(got[:mediaIdx], lightSentinel) {
		t.Errorf("light sentinel %s should appear before the dark media query", lightSentinel)
	}
	if !strings.Contains(mediaBlock, darkSentinel) {
		t.Errorf("dark sentinel %s should appear inside the dark media query", darkSentinel)
	}
}

// TestAutoModeNeutralizesLightOnlyTokenClasses is the direct regression test
// for the invisible-dark-text bug: "github" (light) styles NameOther (chroma
// class "nx", e.g. a plain identifier like "fmt") but "github-dark" leaves
// it unstyled. In auto mode the light ".chroma .nx" rule is unconditional
// (not inside any media query), so without neutralization it would remain
// the only rule for that class even when the OS prefers dark — leaving
// near-black text (#1f2328) on a dark background. The dark media block must
// carry an explicit override for every class the light block styles.
func TestAutoModeNeutralizesLightOnlyTokenClasses(t *testing.T) {
	lightCSS, err := chromaCSS("github")
	if err != nil {
		t.Fatal(err)
	}
	darkCSS, err := chromaCSS("github-dark")
	if err != nil {
		t.Fatal(err)
	}
	lightOnly := neutralizeMissingClasses(darkCSS, lightCSS)
	if lightOnly == "" {
		t.Fatal("expected at least one class github styles that github-dark leaves unstyled (e.g. NameOther/.nx) — test fixture assumption broke")
	}

	got := render(t, codeHeavyMD, func(o *Options) { o.Fragment = false; o.ThemeName = "auto"; o.Highlight = true })
	mediaIdx := strings.Index(got, "@media (prefers-color-scheme: dark){")
	if mediaIdx == -1 {
		t.Fatalf("no dark media query found")
	}
	closeIdx := strings.Index(got[mediaIdx:], "</style>")
	mediaBlock := got[mediaIdx : mediaIdx+closeIdx]

	if !strings.Contains(mediaBlock, ".nx{color:inherit;background-color:inherit}") {
		t.Errorf("dark media block should neutralize the light-only .nx rule; got media block:\n%s", mediaBlock)
	}
}

const mermaidMD = "```mermaid\ngraph TD\n  A --> B\n```\n"

func TestMermaidDarkThemeInitializesDark(t *testing.T) {
	got := render(t, mermaidMD, func(o *Options) { o.Fragment = false; o.ThemeName = "dark" })
	if !strings.Contains(got, `mermaid.initialize({startOnLoad:true,theme:"dark"});`) {
		t.Fatalf("dark page should initialize mermaid with the static \"dark\" theme, got %q", got)
	}
	if strings.Contains(got, "matchMedia") {
		t.Errorf("explicit dark theme shouldn't need matchMedia, got %q", got)
	}
}

func TestMermaidAutoThemeUsesMatchMedia(t *testing.T) {
	got := render(t, mermaidMD, func(o *Options) { o.Fragment = false; o.ThemeName = "auto" })
	want := `mermaid.initialize({startOnLoad:true,theme:(window.matchMedia('(prefers-color-scheme: dark)').matches ? 'dark' : 'default')});`
	if !strings.Contains(got, want) {
		t.Fatalf("auto page should pick mermaid's theme via matchMedia at load, got %q", got)
	}
}

func TestMermaidLightThemeIsStaticDefault(t *testing.T) {
	got := render(t, mermaidMD, func(o *Options) { o.Fragment = false; o.ThemeName = "light" })
	if !strings.Contains(got, `mermaid.initialize({startOnLoad:true,theme:"default"});`) {
		t.Fatalf("light page should initialize mermaid with the static \"default\" theme, got %q", got)
	}
	if strings.Contains(got, "matchMedia") {
		t.Errorf("explicit light theme shouldn't need matchMedia, got %q", got)
	}
}

func TestMermaidSVGBackgroundTransparent(t *testing.T) {
	got := render(t, mermaidMD, func(o *Options) { o.Fragment = false; o.ThemeName = "dark" })
	if !strings.Contains(got, ".markdown-body .mermaid svg") || !strings.Contains(got, "background: transparent") {
		t.Fatalf("page should force the mermaid SVG canvas background transparent, got %q", got)
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
