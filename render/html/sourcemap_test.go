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

func TestSourceMapAnnotatesThematicBreak(t *testing.T) {
	// ThematicBreak gained a real span in v0.9 (span-recording parser in
	// parser/tbreak.go), closing the documented v0.2 gap where <hr> was
	// the one top-level block without a data-md-line.
	got := render(t, "para\n\n***\n\nafter\n", func(o *Options) { o.SourceMap = true })
	if !strings.Contains(got, `<hr data-md-line="3"`) {
		t.Fatalf("missing hr annotation in %q", got)
	}
}

func TestSourceMapStaysBlockLevel(t *testing.T) {
	// Inline nodes carry spans since v0.9, but data-md-line must remain a
	// block-level annotation only.
	got := render(t, "a *b* `c` [d](u)\n", func(o *Options) { o.SourceMap = true })
	for _, tag := range []string{"<em", "<code", "<a"} {
		if strings.Contains(got, tag+" data-md-line") {
			t.Fatalf("inline %s element annotated: %q", tag, got)
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
