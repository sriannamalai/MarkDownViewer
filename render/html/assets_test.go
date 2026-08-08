package htmlrender

import (
	"strings"
	"testing"
)

func TestMermaidEmbeddedOnlyWhenUsed(t *testing.T) {
	with := render(t, "```mermaid\ngraph TD; A-->B\n```\n", func(o *Options) { o.Fragment = false })
	if !strings.Contains(with, "mermaid.initialize") {
		t.Error("mermaid page missing init")
	}
	without := render(t, "plain text\n", func(o *Options) { o.Fragment = false })
	if strings.Contains(without, "mermaid.initialize") {
		t.Error("mermaid embedded without diagrams")
	}
}

func TestKatexEmbeddedOnlyWhenUsed(t *testing.T) {
	with := render(t, "$x^2$\n", func(o *Options) { o.Fragment = false })
	if !strings.Contains(with, "katex.render") {
		t.Error("math page missing katex init")
	}
	without := render(t, "plain\n", func(o *Options) { o.Fragment = false })
	if strings.Contains(without, "katex") {
		t.Error("katex embedded without math")
	}
}
