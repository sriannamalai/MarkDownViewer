package resolve

import "testing"

func TestSafeURLSchemeAllowlist(t *testing.T) {
	cases := []struct {
		name string
		u    string
		want bool
	}{
		{"http", "http://example.com", true},
		{"https", "https://example.com", true},
		{"mailto", "mailto:a@b.com", true},
		{"tel", "tel:+15551234567", true},
		{"relative", "picture.png", true},
		{"relative-path", "docs/page.md", true},
		{"fragment", "#section", true},
		{"protocol-relative", "//example.com/x", true},
		{"javascript", "javascript:alert(1)", false},
		{"javascript-mixed-case", "JaVaScRiPt:alert(1)", false},
		{"javascript-tab-bypass", "jav\tascript:alert(1)", false},
		{"javascript-newline-bypass", "java\nscript:alert(1)", false},
		{"data", "data:text/html;base64,xx", false},
		{"vbscript", "vbscript:msgbox(1)", false},
		{"unknown-scheme", "steam://run", false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := SafeURL(c.u); got != c.want {
				t.Errorf("SafeURL(%q) = %v, want %v", c.u, got, c.want)
			}
		})
	}
}

func TestDefaultResolution(t *testing.T) {
	cases := []struct {
		name   string
		kind   ResolveKind
		target string
		want   string
	}{
		{"link-passes-through", ResolveLink, "docs/page.md", "docs/page.md"},
		{"image-passes-through", ResolveImage, "image.png", "image.png"},
		{"wikilink-gets-md-appended", ResolveWikiLink, "WikiPage", "WikiPage.md"},
		{"wikilink-space-preserved", ResolveWikiLink, "Wiki Page", "Wiki Page.md"},
		{"wikilink-empty-target", ResolveWikiLink, "", ".md"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := DefaultResolution(c.kind, c.target); got != c.want {
				t.Errorf("DefaultResolution(%d, %q) = %q, want %q", c.kind, c.target, got, c.want)
			}
		})
	}
}
