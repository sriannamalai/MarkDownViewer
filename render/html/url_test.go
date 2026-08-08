package htmlrender

import (
	"strings"
	"testing"
)

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
			if got := safeURL(c.u); got != c.want {
				t.Errorf("safeURL(%q) = %v, want %v", c.u, got, c.want)
			}
		})
	}
}

func TestTabBypassLinkRendersWithoutHref(t *testing.T) {
	got := render(t, "[x](jav\tascript:alert(1))\n", nil)
	if strings.Contains(got, "href=") {
		t.Fatalf("tab-bypass link should not produce an href: %q", got)
	}
}

func TestTabBypassWikiLinkRendersWithoutHref(t *testing.T) {
	got := render(t, "[[jav\tascript:alert(1)]]\n", nil)
	if strings.Contains(got, "href=") {
		t.Fatalf("tab-bypass wikilink should not produce an href: %q", got)
	}
	if !strings.Contains(got, `class="wikilink"`) {
		t.Fatalf("blocked wikilink should keep its class: %q", got)
	}
}

func TestUnknownSchemeLinkRendersWithoutHref(t *testing.T) {
	got := render(t, "[x](steam://run)\n", nil)
	if strings.Contains(got, "href=") {
		t.Fatalf("unknown-scheme link should not produce an href: %q", got)
	}
}

func TestRelativeFragmentAndHTTPSKeepWorking(t *testing.T) {
	got := render(t, "[a](./rel.md) [b](#frag) [c](https://example.com)\n", nil)
	if !strings.Contains(got, `href="./rel.md"`) {
		t.Fatalf("relative link stripped: %q", got)
	}
	if !strings.Contains(got, `href="#frag"`) {
		t.Fatalf("fragment link stripped: %q", got)
	}
	if !strings.Contains(got, `href="https://example.com"`) {
		t.Fatalf("https link stripped: %q", got)
	}
}

func TestUnsafeModeBypassesURLPolicy(t *testing.T) {
	got := render(t, "[x](steam://run)\n", func(o *Options) { o.Unsafe = true })
	if !strings.Contains(got, `href="steam://run"`) {
		t.Fatalf("Unsafe should keep unknown scheme: %q", got)
	}

	// The control-char bypass is real via wikilink target parsing (raw
	// links stop scanning the destination at whitespace, so a literal tab
	// there never even forms a link) — confirm Unsafe still keeps it.
	gotWiki := render(t, "[[jav\tascript:alert(1)]]\n", func(o *Options) { o.Unsafe = true })
	if !strings.Contains(gotWiki, "href=\"jav\tascript:alert(1).md\"") {
		t.Fatalf("Unsafe should keep control-char scheme in wikilink: %q", gotWiki)
	}
}
