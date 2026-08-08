package htmlrender

import (
	"strings"
	"testing"
)

func TestHighlightGo(t *testing.T) {
	got := render(t, "```go\nfmt.Println(\"hi\")\n```\n", func(o *Options) { o.Highlight = true })
	if !strings.Contains(got, `class="chroma"`) || !strings.Contains(got, "<span") {
		t.Fatalf("not highlighted: %q", got)
	}
	if strings.Contains(got, "style=") {
		t.Fatalf("inline styles leaked (classes mode expected): %q", got)
	}
}

func TestHighlightUnknownLangFallsBack(t *testing.T) {
	got := render(t, "```nosuchlang\nzzz\n```\n", func(o *Options) { o.Highlight = true })
	if !strings.Contains(got, `<pre><code class="language-nosuchlang">zzz`) {
		t.Fatalf("fallback broken: %q", got)
	}
}

func TestChromaCSS(t *testing.T) {
	css, err := chromaCSS("github")
	if err != nil || !strings.Contains(css, ".chroma") {
		t.Fatalf("css: %v %q", err, css)
	}
}
