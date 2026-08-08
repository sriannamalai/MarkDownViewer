package assets

import (
	"strings"
	"testing"
)

func TestAssetsNonEmpty(t *testing.T) {
	if len(MermaidJS()) < 1_000_000 {
		t.Errorf("mermaid suspiciously small: %d", len(MermaidJS()))
	}
	if len(KatexJS()) < 100_000 {
		t.Errorf("katex js suspiciously small: %d", len(KatexJS()))
	}
	css := KatexCSS()
	if len(css) < 100_000 || strings.Contains(css, "url(fonts/") {
		t.Errorf("katex css wrong: len=%d fontRefs=%t", len(css), strings.Contains(css, "url(fonts/"))
	}
}
