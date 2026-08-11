package derive_test

import (
	"strings"
	"testing"

	"github.com/sriannamalai/markdownviewer/document"
	"github.com/sriannamalai/markdownviewer/render/internal/derive"
	"github.com/sriannamalai/markdownviewer/resolve"
)

func TestAdmonitionTitle(t *testing.T) {
	for _, c := range []struct{ in, variant, title string }{
		{"", "note", "Note"},
		{"note", "note", "Note"},
		{"tip", "tip", "Tip"},
		{"important", "important", "Important"},
		{"warning", "warning", "Warning"},
		{"caution", "caution", "Caution"},
		{"custom", "custom", "Custom"}, // hand-built docs can carry anything
	} {
		v, title := derive.AdmonitionTitle(c.in)
		if v != c.variant || title != c.title {
			t.Errorf("AdmonitionTitle(%q) = %q, %q; want %q, %q", c.in, v, title, c.variant, c.title)
		}
	}
}

func TestCodeLabel(t *testing.T) {
	if got := derive.CodeLabel(""); got != "code" {
		t.Errorf("CodeLabel(\"\") = %q", got)
	}
	if got := derive.CodeLabel("Go"); got != "Go" {
		t.Errorf("CodeLabel(Go) = %q", got)
	}
}

func TestHref(t *testing.T) {
	// Default path: SafeURL filter + percent-encoding.
	if u, ok := derive.Href(nil, false, resolve.ResolveLink, "https://e.com/a b"); !ok || u != "https://e.com/a%20b" {
		t.Errorf("default link: %q %v", u, ok)
	}
	if u, ok := derive.Href(nil, false, resolve.ResolveLink, "javascript:x"); ok || u != "" {
		t.Errorf("blocked scheme: %q %v", u, ok)
	}
	// Unsafe bypasses the filter but still percent-encodes.
	if u, ok := derive.Href(nil, true, resolve.ResolveLink, "javascript:alert(1)"); !ok || !strings.HasPrefix(u, "javascript:") {
		t.Errorf("unsafe: %q %v", u, ok)
	}
	// Wiki default: ".md" append, no percent-encoding (filesystem path).
	if u, ok := derive.Href(nil, false, resolve.ResolveWikiLink, "Page Name"); !ok || u != "Page Name.md" {
		t.Errorf("wiki default: %q %v", u, ok)
	}
	// Resolver ok=true is trusted verbatim: no filter, no encoding.
	res := func(kind resolve.ResolveKind, target string) (string, bool) {
		return "custom:with space/" + target, true
	}
	if u, ok := derive.Href(res, false, resolve.ResolveImage, "x"); !ok || u != "custom:with space/x" {
		t.Errorf("resolver trust: %q %v", u, ok)
	}
	// Resolver decline falls back to the default path.
	decline := func(resolve.ResolveKind, string) (string, bool) { return "", false }
	if u, ok := derive.Href(decline, false, resolve.ResolveWikiLink, "W"); !ok || u != "W.md" {
		t.Errorf("resolver decline: %q %v", u, ok)
	}
}

func TestFootnotes(t *testing.T) {
	if got := derive.Footnotes(&document.Document{}); got != nil {
		t.Fatalf("no footnotes: %v", got)
	}
	doc := &document.Document{}
	p := &document.Paragraph{}
	p.AppendChild(&document.FootnoteRef{Index: 1})
	p.AppendChild(&document.FootnoteRef{Index: 1})
	doc.AppendChild(p)
	def1 := &document.FootnoteDef{Index: 1}
	// A footnote body referencing another footnote counts too.
	body := &document.Paragraph{}
	body.AppendChild(&document.FootnoteRef{Index: 2})
	def1.AppendChild(body)
	def2 := &document.FootnoteDef{Index: 2}
	doc.Footnotes = []*document.FootnoteDef{def1, def2}
	fns := derive.Footnotes(doc)
	if len(fns) != 2 || fns[0].Def != def1 || fns[1].Def != def2 {
		t.Fatalf("pairing order: %+v", fns)
	}
	if fns[0].RefCount != 2 || fns[1].RefCount != 1 {
		t.Fatalf("ref counts: %d, %d", fns[0].RefCount, fns[1].RefCount)
	}
}

func TestSanitizeHTML(t *testing.T) {
	out := derive.SanitizeHTML(`<em>ok</em><script>alert(1)</script><b onclick="x()">b</b>`)
	if strings.Contains(out, "script") || strings.Contains(out, "onclick") {
		t.Fatalf("sanitize leak: %q", out)
	}
	if !strings.Contains(out, "<em>ok</em>") || !strings.Contains(out, "<b>b</b>") {
		t.Fatalf("over-stripped: %q", out)
	}
}
