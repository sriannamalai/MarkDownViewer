// Package htmlrender renders the document model to themed HTML.
package htmlrender

// ResolveKind identifies which kind of destination is being resolved by a
// Resolver call.
type ResolveKind int

// These values are ABI-frozen: the FFI and WASM boundaries expose them
// as bare ints (0=link, 1=image, 2=wiki-link). Append-only — never
// reorder or renumber (see internal/boundary/kinds_test.go).
const (
	ResolveLink     ResolveKind = iota // a Link's Destination
	ResolveImage                       // an Image's Destination
	ResolveWikiLink                    // a WikiLink's Target
)

// Resolver lets hosts rewrite link/image/wiki-link targets. Returning
// ok=false falls back to default handling.
//
// Trust contract: URLs returned with ok=true are emitted as-is, without
// scheme filtering — hosts fully control resolution, so they must not
// echo untrusted targets back unexamined.
type Resolver func(kind ResolveKind, target string) (url string, ok bool)

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
	ThemeOverrides map[string]string // CSS custom-property overrides (name → value); keys must match --[a-zA-Z0-9_-]+, non-conforming keys silently dropped; values emitted into the page's <style> element with </style sequences stripped defensively
	Stylesheet     string            // non-empty replaces theme.BaseCSS() entirely; emitted into the page's <style> element with </style sequences stripped defensively
	SourceMap      bool              // annotate top-level blocks with data-md-line attributes
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
