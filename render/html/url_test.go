package htmlrender

import (
	"strings"
	"testing"
)

// The SafeURL scheme-allowlist unit table lives with the policy in
// resolve/url_test.go; the tests here pin that the HTML renderer actually
// enforces that policy on rendered output.

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

// TestEntityEncodedSchemeBypassRendersWithoutHref pins the ordering between
// character-reference decoding (parser/transform.go's unescapeText, applied
// to link destinations before they ever reach the renderer) and the
// resolve.SafeURL scheme check in href() (render/html/renderer.go):
// decoding must happen BEFORE the SafeURL check runs, or an attacker can
// hide a blocked scheme like "javascript:" behind an HTML entity or
// character reference and have it slip through as an "unknown"/unparseable
// scheme.
//
// Nothing else in the suite exercises this ordering: both TestCommonMarkSpec
// and TestGFMExtras render with Options.Unsafe=true, where SafeURL is
// never even consulted, so SafeURL had zero conformance coverage prior to
// this test. All three cases are expected to be blocked (no href
// attribute) under the default (Unsafe=false) policy, exactly like the
// unencoded "javascript:alert(1)" case in TestUnsafeURLsStripped.
func TestEntityEncodedSchemeBypassRendersWithoutHref(t *testing.T) {
	cases := []struct {
		name string
		md   string
	}{
		// "&#106;" is a decimal character reference for 'j'.
		{"decimal-char-ref-in-scheme", "[a](&#106;avascript:alert(1))\n"},
		// A literal tab is a known bypass for naive prefix blocklists
		// (browsers strip control chars before parsing the scheme); this
		// checks the same bypass still works when the tab arrives via
		// "&#9;" (decimal character reference) instead of a raw byte.
		{"decimal-char-ref-tab-in-scheme", "[a](jav&#9;ascript:x)\n"},
		// "&NewLine;" is a named HTML5 entity for a literal newline —
		// same control-character bypass, via a named reference instead of
		// a numeric one.
		{"named-entity-newline-in-scheme", "[a](java&NewLine;script:x)\n"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := render(t, c.md, nil)
			if strings.Contains(got, "href=") {
				t.Fatalf("entity-encoded javascript: scheme should not produce an href: %q", got)
			}
		})
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
