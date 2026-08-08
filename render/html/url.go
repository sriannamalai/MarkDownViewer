package htmlrender

import "strings"

// allowedSchemes is the set of URL schemes permitted under the default
// policy. Everything else — including unknown schemes and legacy
// script-ish ones like javascript:/vbscript:/data: — is blocked.
var allowedSchemes = map[string]bool{
	"http": true, "https": true, "mailto": true, "tel": true,
}

// safeURL reports whether a URL is allowed under the default policy.
//
// Control characters (runes <= 0x20) are stripped before scheme inspection.
// This closes a bypass where e.g. "jav\tascript:" evades a naive prefix
// blocklist: browsers strip such characters when parsing a URL's scheme,
// so the destination is still treated as javascript: even though it
// doesn't literally start with that prefix.
//
// After stripping, if a scheme is present (a ':' before any '/', '?', or
// '#'), it must be on the allowlist. URLs with no scheme — relative paths,
// #fragments, protocol-relative // URLs — are allowed.
func safeURL(u string) bool {
	s := stripControl(u)
	scheme, ok := urlScheme(s)
	if !ok {
		return true
	}
	return allowedSchemes[strings.ToLower(scheme)]
}

// stripControl removes every rune <= 0x20 (space and below, including tab,
// newline, and other ASCII control characters) from s.
func stripControl(s string) string {
	if strings.IndexFunc(s, func(r rune) bool { return r <= 0x20 }) < 0 {
		return s
	}
	var b strings.Builder
	b.Grow(len(s))
	for _, r := range s {
		if r <= 0x20 {
			continue
		}
		b.WriteRune(r)
	}
	return b.String()
}

// urlScheme extracts a leading "scheme" if a ':' appears before any of
// '/', '?', or '#'. A leading ':' (empty scheme) does not count.
func urlScheme(s string) (string, bool) {
	for i, r := range s {
		switch r {
		case ':':
			if i == 0 {
				return "", false
			}
			return s[:i], true
		case '/', '?', '#':
			return "", false
		}
	}
	return "", false
}
