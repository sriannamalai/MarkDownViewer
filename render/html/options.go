// Package htmlrender renders the document model to themed HTML.
package htmlrender

import "github.com/sriannamalai/markdownviewer/resolve"

// ResolveKind identifies which kind of destination is being resolved by a
// Resolver call. It is an alias for [resolve.ResolveKind]; the resolve
// package is the renderer-agnostic home of the resolution policy.
type ResolveKind = resolve.ResolveKind

// The kind values are ABI-frozen (0=link, 1=image, 2=wiki-link);
// see resolve.ResolveKind and internal/boundary/kinds_test.go.
const (
	ResolveLink     = resolve.ResolveLink     // a Link's Destination
	ResolveImage    = resolve.ResolveImage    // an Image's Destination
	ResolveWikiLink = resolve.ResolveWikiLink // a WikiLink's Target
)

// Resolver lets hosts rewrite link/image/wiki-link targets. It is an
// alias for [resolve.Resolver], which documents the trust contract:
// URLs returned with ok=true are emitted as-is, without scheme
// filtering; ok=false falls back to default handling
// (resolve.DefaultResolution filtered by resolve.SafeURL).
type Resolver = resolve.Resolver

// Options configures Render's output: page assembly, safety policy, and
// which optional feature renderings are enabled.
type Options struct {
	Fragment       bool              // emit body-only fragment instead of a full page
	ThemeName      string            // "light", "dark", "auto"
	Unsafe         bool              // allow raw HTML and all URL schemes
	Highlight      bool              // chroma syntax highlighting
	Mermaid        bool              // mermaid diagram support
	Math           bool              // KaTeX math support
	HeadingAnchors bool              // id attributes on headings
	Resolver       Resolver          // optional hook to rewrite link/image/wiki-link targets; nil uses default resolution
	ThemeOverrides map[string]string // CSS custom-property overrides (name → value); keys must match --[a-zA-Z0-9_-]+, non-conforming keys silently dropped. VALUES are host-trusted CSS emitted into the page's <style> element: only </style sequences are stripped, so a value can close the :root{} block and inject arbitrary CSS rules (including url()) — never echo untrusted data into values
	Stylesheet     string            // non-empty replaces theme.BaseCSS() entirely; emitted into the page's <style> element with </style sequences stripped defensively
	ExtraCSS       string            // appended after whatever base styling applied (base+theme, or Stylesheet when set); sanitized like Stylesheet
	SourceMap      bool              // annotate top-level blocks with data-md-line attributes
	CodeHeader     bool              // wrap code blocks with a header row (language label + copy button); full pages also get an inline clipboard script
}

// DefaultOptions returns the recommended Options: full-page output, light
// theme, all optional renderings (highlighting, mermaid, math, heading
// anchors) enabled, and raw HTML/unsafe URL schemes disallowed.
func DefaultOptions() Options {
	return Options{
		ThemeName: "auto", Highlight: true, Mermaid: true,
		Math: true, HeadingAnchors: true,
	}
}
