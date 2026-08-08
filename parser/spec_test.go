package parser_test

import (
	"bytes"
	"encoding/json"
	"os"
	"testing"

	"github.com/sriannamalai/markdownviewer/parser"
	htmlrender "github.com/sriannamalai/markdownviewer/render/html"
)

type specCase struct {
	Markdown string `json:"markdown"`
	HTML     string `json:"html"`
	Example  int    `json:"example"`
}

// skips documents examples our pipeline intentionally renders differently.
// Every entry needs a reason. Adding entries requires justification in the
// commit message; the count is asserted so it cannot silently grow.
var skips = map[int]string{}

const maxSkips = 15

func TestCommonMarkSpec(t *testing.T) {
	runSpec(t, "testdata/commonmark-0.31.2.json", parser.CommonMarkOnly())
}

func TestGFMExtras(t *testing.T) {
	runSpec(t, "testdata/gfm-extras.json", parser.Default())
}

func runSpec(t *testing.T, path string, cfg parser.Config) {
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var cases []specCase
	if err := json.Unmarshal(data, &cases); err != nil {
		t.Fatal(err)
	}
	if len(skips) > maxSkips {
		t.Fatalf("skip list has %d entries; cap is %d", len(skips), maxSkips)
	}
	opts := htmlrender.DefaultOptions()
	opts.Fragment = true
	opts.Unsafe = true // spec output includes raw HTML verbatim
	opts.HeadingAnchors = false
	opts.Highlight = false
	opts.Mermaid = false
	opts.Math = false
	failed := 0
	for _, c := range cases {
		if reason, ok := skips[c.Example]; ok {
			t.Logf("example %d skipped: %s", c.Example, reason)
			continue
		}
		doc, err := parser.ParseWith([]byte(c.Markdown), cfg)
		if err != nil {
			t.Errorf("example %d: parse error %v", c.Example, err)
			continue
		}
		var buf bytes.Buffer
		if err := htmlrender.Render(&buf, doc, opts); err != nil {
			t.Errorf("example %d: render error %v", c.Example, err)
			continue
		}
		if buf.String() != c.HTML {
			failed++
			if failed <= 20 {
				t.Errorf("example %d:\ninput  %q\ngot    %q\nwant   %q", c.Example, c.Markdown, buf.String(), c.HTML)
			}
		}
	}
	if failed > 0 {
		t.Errorf("%d spec examples failed", failed)
	}
}
