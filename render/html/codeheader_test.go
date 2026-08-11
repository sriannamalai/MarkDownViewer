package htmlrender

import (
	"strings"
	"testing"
)

// header returns the exact prologue codeBlock emits for lang when the
// CodeHeader option is on (lang already escaped by the caller).
func header(lang string) string {
	return `<div class="md-code"><div class="md-code-header"><span class="md-code-lang">` +
		lang + `</span><button type="button" class="md-code-copy">Copy</button></div>`
}

func TestCodeHeaderWrapsPlainPath(t *testing.T) {
	got := render(t, "```shell\necho hi\n```\n", func(o *Options) { o.CodeHeader = true })
	h := strings.Index(got, header("shell"))
	pre := strings.Index(got, "<pre")
	preEnd := strings.Index(got, "</pre>")
	end := strings.Index(got, "</div>\n")
	if h == -1 {
		t.Fatalf("header prologue missing: %q", got)
	}
	if pre == -1 || preEnd == -1 || end == -1 || !(h < pre && pre < preEnd && preEnd < end) {
		t.Fatalf("wrapper order wrong (h=%d pre=%d /pre=%d end=%d): %q", h, pre, preEnd, end, got)
	}
	if !strings.Contains(got, `class="language-shell"`) {
		t.Fatalf("plain pre/code body missing inside wrapper: %q", got)
	}
}

func TestCodeHeaderWrapsChromaPath(t *testing.T) {
	got := render(t, "```go\nfmt.Println(\"hi\")\n```\n", func(o *Options) {
		o.Highlight = true
		o.CodeHeader = true
	})
	h := strings.Index(got, header("go"))
	chroma := strings.Index(got, `class="chroma"`)
	end := strings.LastIndex(got, "</div>")
	if h == -1 {
		t.Fatalf("header prologue missing: %q", got)
	}
	if chroma == -1 || !(h < chroma && chroma < end) {
		t.Fatalf("wrapper must surround chroma markup (h=%d chroma=%d end=%d): %q", h, chroma, end, got)
	}
}

func TestCodeHeaderUnlabeledUsesCode(t *testing.T) {
	got := render(t, "```\nplain\n```\n", func(o *Options) { o.CodeHeader = true })
	if !strings.Contains(got, `<span class="md-code-lang">code</span>`) {
		t.Fatalf("unlabeled fence must label as %q: %q", "code", got)
	}
}

func TestCodeHeaderOffByDefault(t *testing.T) {
	got := render(t, "```shell\necho hi\n```\n", nil)
	if strings.Contains(got, "md-code") {
		t.Fatalf("md-code must not appear when CodeHeader is off: %q", got)
	}
}

func TestCodeHeaderPreservesDataMdLine(t *testing.T) {
	got := render(t, "```shell\necho hi\n```\n", func(o *Options) {
		o.CodeHeader = true
		o.SourceMap = true
	})
	if !strings.Contains(got, `<pre data-md-line="`) {
		t.Fatalf("data-md-line must stay on the pre element: %q", got)
	}
	if !strings.Contains(got, `<div class="md-code"><div class="md-code-header">`) {
		t.Fatalf("wrapper div must carry no attributes: %q", got)
	}
}

func TestCodeHeaderSkipsLiveMermaidAndMath(t *testing.T) {
	const md = "```mermaid\ngraph TD;\n```\n\n$$\nx^2\n$$\n"
	live := render(t, md, func(o *Options) { o.CodeHeader = true })
	if strings.Contains(live, "md-code") {
		t.Fatalf("live mermaid/math blocks must not be wrapped: %q", live)
	}
	if !strings.Contains(live, `class="mermaid"`) || !strings.Contains(live, "math-display") {
		t.Fatalf("expected live mermaid+math renderings: %q", live)
	}
	fallback := render(t, md, func(o *Options) {
		o.CodeHeader = true
		o.Mermaid = false
		o.Math = false
	})
	for _, want := range []string{header("mermaid"), header("math")} {
		if !strings.Contains(fallback, want) {
			t.Errorf("engine-disabled fallback missing %q: %q", want, fallback)
		}
	}
}

func TestCodeHeaderLanguageEscaped(t *testing.T) {
	got := render(t, "```sh\"><img\nhi\n```\n", func(o *Options) { o.CodeHeader = true })
	if !strings.Contains(got, `<span class="md-code-lang">sh&quot;&gt;&lt;img</span>`) {
		t.Fatalf("language must be escaped in the label: %q", got)
	}
	if strings.Contains(got, `md-code-lang">sh"><img`) {
		t.Fatalf("raw language leaked unescaped: %q", got)
	}
}
