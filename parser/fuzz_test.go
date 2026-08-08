package parser_test

import (
	"bytes"
	"testing"

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
