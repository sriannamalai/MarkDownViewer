package parser_test

import (
	"bytes"
	"testing"

	"github.com/sriannamalai/markdownviewer/document"
	"github.com/sriannamalai/markdownviewer/parser"
	htmlrender "github.com/sriannamalai/markdownviewer/render/html"
)

func FuzzParseRender(f *testing.F) {
	seeds := []string{
		"# h\n\npara\n", "- [x] t\n", "| a |\n|---|\n| b |\n",
		"```go\nx\n```\n", "$a$ $$b$$\n", "> [!NOTE]\n> x\n",
		"[[w]] :smile: ~~s~~ [l](u) ![i](u)\n", "Term\n: def\n",
		"---\nk: v\n---\nx[^1]\n\n[^1]: n\n", "$$\nx\n$$\n",
		"<div><script>x</script></div>\n",
	}
	for _, s := range seeds {
		f.Add([]byte(s))
	}
	f.Fuzz(func(t *testing.T, data []byte) {
		doc, err := parser.Parse(data)
		if err != nil {
			return
		}
		var buf bytes.Buffer
		opts := htmlrender.DefaultOptions()
		opts.Fragment = true
		_ = htmlrender.Render(&buf, doc, opts)
	})
}

func FuzzJSONRoundTrip(f *testing.F) {
	f.Add([]byte("# h\n\npara *em*\n\n- [x] t\n\n| a |\n|---|\n| b |\n"))
	f.Add([]byte("> [!NOTE]\n> x\n\n$m$\n\n```mermaid\ng\n```\n\ntext[^1]\n\n[^1]: n\n"))
	f.Fuzz(func(t *testing.T, data []byte) {
		doc, err := parser.Parse(data)
		if err != nil {
			return
		}
		out, err := document.MarshalJSON(doc)
		if err != nil {
			t.Fatalf("marshal: %v", err)
		}
		back, err := document.UnmarshalJSON(out)
		if err != nil {
			t.Fatalf("unmarshal: %v\njson: %s", err, out)
		}
		if document.Dump(doc) != document.Dump(back) {
			t.Fatalf("round trip changed tree")
		}
	})
}
