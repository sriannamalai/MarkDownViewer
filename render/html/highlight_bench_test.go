package htmlrender

import (
	"bytes"
	"fmt"
	"strings"
	"testing"

	"github.com/sriannamalai/markdownviewer/parser"
)

// codeHeavySource builds a markdown document with n fenced code blocks
// (distinct contents, cycling over a few real languages) — the re-render
// workload the highlight cache targets (theme flips, font-size steps, live
// preview of a mostly-unchanged doc).
func codeHeavySource(n int) []byte {
	langs := []string{"go", "python", "javascript", "json", "c"}
	bodies := []string{
		"func f%d(x int) int {\n\tif x > %d {\n\t\treturn x * 2 // branch %d\n\t}\n\ts := fmt.Sprintf(\"v=%%d\", x)\n\treturn len(s) + %d\n}",
		"def f%d(x):\n    if x > %d:\n        return x * 2  # branch %d\n    s = f\"v={x}\"\n    return len(s) + %d",
		"function f%d(x) {\n  if (x > %d) {\n    return x * 2; // branch %d\n  }\n  const s = `v=${x}`;\n  return s.length + %d;\n}",
		"{\n  \"id\": %d,\n  \"limit\": %d,\n  \"branch\": %d,\n  \"offset\": %d\n}",
		"static int f%d(int x) {\n    if (x > %d) {\n        return x * 2; /* branch %d */\n    }\n    return x + %d;\n}",
	}
	var b strings.Builder
	b.WriteString("# Code-heavy document\n\n")
	for i := 0; i < n; i++ {
		li := i % len(langs)
		fmt.Fprintf(&b, "Paragraph %d introducing a snippet.\n\n", i)
		b.WriteString("```" + langs[li] + "\n")
		fmt.Fprintf(&b, bodies[li], i, i+10, i, i)
		b.WriteString("\n```\n\n")
	}
	return []byte(b.String())
}

func benchRenderCodeHeavy(b *testing.B, cold bool) {
	b.Helper()
	doc, err := parser.Parse(codeHeavySource(50))
	if err != nil {
		b.Fatal(err)
	}
	opts := DefaultOptions()
	opts.Fragment = true
	opts.Highlight = true
	var buf bytes.Buffer
	// Prime once so the warm variant measures pure cache hits and so both
	// variants exclude one-time chroma setup (lexer registry, formatter).
	if err := Render(&buf, doc, opts); err != nil {
		b.Fatal(err)
	}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if cold {
			b.StopTimer()
			resetHighlightCacheForTest()
			b.StartTimer()
		}
		buf.Reset()
		if err := Render(&buf, doc, opts); err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkRenderCodeHeavyCold measures a first render (empty highlight
// cache each iteration): the chroma tokenise+format cost is paid for every
// fence.
func BenchmarkRenderCodeHeavyCold(b *testing.B) { benchRenderCodeHeavy(b, true) }

// BenchmarkRenderCodeHeavyWarm measures a re-render of the same document
// with the highlight cache primed: every fence is a cache hit.
func BenchmarkRenderCodeHeavyWarm(b *testing.B) { benchRenderCodeHeavy(b, false) }
