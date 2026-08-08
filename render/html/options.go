// Package htmlrender renders the document model to themed HTML.
package htmlrender

type ResolveKind int

const (
	ResolveLink ResolveKind = iota
	ResolveImage
	ResolveWikiLink
)

// Resolver lets hosts rewrite link/image/wiki-link targets. Returning
// ok=false falls back to default handling.
//
// Trust contract: URLs returned with ok=true are emitted as-is, without
// scheme filtering — hosts fully control resolution, so they must not
// echo untrusted targets back unexamined.
type Resolver func(kind ResolveKind, target string) (url string, ok bool)

type Options struct {
	Fragment       bool   // emit body-only fragment instead of a full page
	ThemeName      string // "light", "dark", "auto"
	Unsafe         bool   // allow raw HTML and all URL schemes
	Highlight      bool   // chroma syntax highlighting
	Mermaid        bool   // mermaid diagram support
	Math           bool   // KaTeX math support
	HeadingAnchors bool   // id attributes on headings
	Resolver       Resolver
	ThemeOverrides map[string]string // CSS custom-property overrides (name → value); emitted into the page's <style> element with </style sequences stripped defensively
	Stylesheet     string            // non-empty replaces theme.BaseCSS() entirely; emitted into the page's <style> element with </style sequences stripped defensively
}

func DefaultOptions() Options {
	return Options{
		ThemeName: "auto", Highlight: true, Mermaid: true,
		Math: true, HeadingAnchors: true,
	}
}
