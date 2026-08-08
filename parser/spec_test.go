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

// commonMarkSkips documents CommonMark spec examples our pipeline
// intentionally renders differently. Every entry needs a reason. Adding
// entries requires justification in the commit message; the count is
// asserted so it cannot silently grow.
//
// This map is scoped to the CommonMark suite only — NOT shared with
// TestGFMExtras. The two suites number their examples independently
// (CommonMark 1-652, gfm-extras 1-12), so a single shared map keyed on
// example number would let a low-numbered CommonMark skip silently skip
// an unrelated GFM-extras case with the same number.
var commonMarkSkips = map[int]string{}

const maxSkips = 15

func TestCommonMarkSpec(t *testing.T) {
	runSpec(t, "testdata/commonmark-0.31.2.json", parser.CommonMarkOnly(), commonMarkSkips, 652)
}

func TestGFMExtras(t *testing.T) {
	runSpec(t, "testdata/gfm-extras.json", parser.Default(), nil, 12)
}

// runSpec loads the spec cases at path and checks each one's rendered HTML
// against its expected HTML, skipping any example number present in skips
// (with a logged reason) and applying cfg's config to the parser.
//
// wantCount pins the number of cases the file must contain: without it, a
// testdata file truncated or corrupted down to zero (or partial) cases
// would make the loop below run 0 (or too few) iterations and the test
// would pass vacuously, silently losing conformance coverage.
func runSpec(t *testing.T, path string, cfg parser.Config, skips map[int]string, wantCount int) {
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var cases []specCase
	if err := json.Unmarshal(data, &cases); err != nil {
		t.Fatal(err)
	}
	if len(cases) != wantCount {
		t.Fatalf("%s: loaded %d cases, want %d", path, len(cases), wantCount)
	}
	if len(skips) > maxSkips {
		t.Fatalf("skip list has %d entries; cap is %d", len(skips), maxSkips)
	}
	opts := specRenderOptions()
	failed := 0
	for _, c := range cases {
		if reason, ok := skips[c.Example]; ok {
			t.Logf("example %d skipped: %s", c.Example, reason)
			continue
		}
		if !checkSpecCase(t, cfg, opts, c) {
			failed++
		}
	}
	if failed > 0 {
		t.Errorf("%d spec examples failed", failed)
	}
}

// specRenderOptions is the rendering configuration shared by every spec
// case check: fragment output, raw HTML passed through verbatim (the spec
// fixtures embed literal HTML in their expected output), and every
// extension this pipeline adds beyond bare CommonMark/GFM turned off so it
// can't interfere with byte-for-byte comparison.
func specRenderOptions() htmlrender.Options {
	opts := htmlrender.DefaultOptions()
	opts.Fragment = true
	opts.Unsafe = true
	opts.HeadingAnchors = false
	opts.Highlight = false
	opts.Mermaid = false
	opts.Math = false
	return opts
}

// checkSpecCase parses and renders one case under cfg/opts and reports
// (via t.Errorf) any parse/render error or output mismatch. It returns
// whether the case passed, so callers doing bulk runs (runSpec) can tally
// failures separately from one-off assertions (e.g.
// TestOverLongHexEntityStaysLiteral) that just want a plain pass/fail.
func checkSpecCase(t *testing.T, cfg parser.Config, opts htmlrender.Options, c specCase) bool {
	t.Helper()
	doc, err := parser.ParseWith([]byte(c.Markdown), cfg)
	if err != nil {
		t.Errorf("example %d: parse error %v", c.Example, err)
		return false
	}
	var buf bytes.Buffer
	if err := htmlrender.Render(&buf, doc, opts); err != nil {
		t.Errorf("example %d: render error %v", c.Example, err)
		return false
	}
	if buf.String() != c.HTML {
		t.Errorf("example %d:\ninput  %q\ngot    %q\nwant   %q", c.Example, c.Markdown, buf.String(), c.HTML)
		return false
	}
	return true
}

// TestOverLongHexEntityStaysLiteral pins the CommonMark numeric character
// reference grammar's cap of 6 hex digits for "&#xHHHHHH;" references (see
// transform.go's resolveCharRef): a 7-digit hex reference like
// "&#x0000041;" is not a valid character reference at all, so the whole
// "&...;" run must stay literal (HTML-escaped as text) rather than
// decoding to "A". This is a hand-written regression case, not part of the
// commonmark-0.31.2.json fixture — the 652 official examples don't happen
// to exercise this specific boundary.
func TestOverLongHexEntityStaysLiteral(t *testing.T) {
	checkSpecCase(t, parser.CommonMarkOnly(), specRenderOptions(), specCase{
		Markdown: "&#x0000041;\n",
		HTML:     "<p>&amp;#x0000041;</p>\n",
		Example:  -1,
	})
}
