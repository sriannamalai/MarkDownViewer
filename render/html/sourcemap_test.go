package htmlrender

import (
	"strings"
	"testing"
)

func TestSourceMapAnnotatesTopLevelBlocks(t *testing.T) {
	got := render(t, "# Title\n\npara\n\n- a\n", func(o *Options) { o.SourceMap = true })
	for _, want := range []string{
		`<h1 data-md-line="1"`, `<p data-md-line="3">`, `<ul data-md-line="5">`,
	} {
		if !strings.Contains(got, want) {
			t.Errorf("missing %q in %q", want, got)
		}
	}
}

func TestSourceMapOffByDefault(t *testing.T) {
	got := render(t, "# Title\n", nil)
	if strings.Contains(got, "data-md-line") {
		t.Fatalf("source map leaked: %q", got)
	}
}

func TestSourceMapSkipsUnownedMarkup(t *testing.T) {
	got := render(t, "```go\nx := 1\n```\n", func(o *Options) { o.SourceMap = true; o.Highlight = true })
	if strings.Contains(got, "data-md-line") {
		t.Fatalf("chroma markup must not be annotated: %q", got)
	}
}
