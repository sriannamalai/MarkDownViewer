package parser_test

import (
	"strings"
	"testing"
	"time"

	"github.com/sriannamalai/markdownviewer/parser"
)

// nestedListSource generates depth levels of a "  "-indented "- x" list,
// each level nested one deeper than the previous: line i is prefixed with
// i pairs of spaces, producing a list item nested i levels deep.
func nestedListSource(depth int) []byte {
	var b strings.Builder
	for i := 0; i < depth; i++ {
		b.WriteString(strings.Repeat("  ", i))
		b.WriteString("- x\n")
	}
	return []byte(b.String())
}

// TestDeeplyNestedListDoesNotHang is a regression-visible guard for the
// deeply-nested-list resource-exhaustion vector documented in SECURITY.md
// and in this package's doc comment: goldmark's list-nesting parse cost is
// super-quadratic in nesting depth (empirically: ~2000 levels take ~6s,
// ~4000 levels take >30s, on an input of only tens of kilobytes).
//
// This test is a hang detector, not a benchmark: it uses a moderate depth
// (300 levels) that parses in tens of milliseconds on typical hardware, and
// a deliberately huge 30s wall-clock watchdog. Shared CI runners (notably
// macOS under -race) have been observed 10-50x slower than local hardware —
// a 500-level/5s version of this test flaked there. Only a pathological
// regression (minutes-scale hang) should ever trip the watchdog. It is not
// a mitigation — Parse has no built-in timeout, by design (see the package
// doc comment).
func TestDeeplyNestedListDoesNotHang(t *testing.T) {
	const depth = 300
	const watchdog = 30 * time.Second

	src := nestedListSource(depth)

	done := make(chan struct{})
	var doc any
	var err error
	go func() {
		defer close(done)
		doc, err = parser.Parse(src)
	}()

	select {
	case <-done:
		if err != nil {
			t.Fatalf("Parse: %v", err)
		}
		if doc == nil {
			t.Fatal("Parse returned a nil document")
		}
	case <-time.After(watchdog):
		t.Fatalf("Parse did not return within %v for a %d-level nested list (%d bytes); "+
			"see the resource-exhaustion note in SECURITY.md and the parser package doc comment",
			watchdog, depth, len(src))
	}
}
