// Package derive holds the tiny output-shaping derivations shared by
// every renderer: the admonition title rule, the code-block display
// label, the footnote def/ref pairing walk, the raw-HTML sanitize
// policy, and the destination-resolution pipeline (resolver → default
// resolution → SafeURL filter → percent-encoding).
//
// These are the pieces of renderer behavior that must be THE SAME CODE
// in the HTML renderer and the native render tree (and any future
// renderer): duplicating any of them would let the two outputs drift.
// A change here changes every renderer at once — which is the point.
// Renderer-specific concerns (HTML escaping, tag shapes, wire schema)
// stay in the renderers.
package derive

import (
	"strings"

	"github.com/microcosm-cc/bluemonday"
	"github.com/yuin/goldmark/util"

	"github.com/sriannamalai/markdownviewer/document"
	"github.com/sriannamalai/markdownviewer/resolve"
)

// AdmonitionTitle normalizes an Admonition's variant and derives its
// display title. An empty variant defaults to "note" (hand-built
// documents and doc JSON can carry one); the title is the normalized
// variant with its first byte upper-cased ("note" → "Note"). The
// returned variant is what renderers should use anywhere the variant
// itself is emitted (CSS class, tree field) so the default applies
// uniformly.
//
// The parser constrains variants to lowercase ASCII words, so the
// single-byte upper-case is exact for parser output; arbitrary bytes in
// hand-built documents get the same byte-wise treatment both renderers
// always applied.
func AdmonitionTitle(variant string) (normalized, title string) {
	if variant == "" {
		variant = "note"
	}
	return variant, strings.ToUpper(variant[:1]) + variant[1:]
}

// CodeLabel derives a code block's display label from its language tag:
// the tag itself, or "code" when the block has none. Display casing is
// presentation (CSS text-transform in the HTML renderer, host styling
// in native renderers) — the label is the raw tag.
func CodeLabel(language string) string {
	if language == "" {
		return "code"
	}
	return language
}

// Href runs the full destination-resolution pipeline shared by all
// renderers. The boolean reports whether the destination is allowed:
// false means it was blocked by policy, which is a different outcome
// from an empty-but-allowed destination (e.g. "[foo]: <>", CommonMark
// spec example 200) — callers must not conflate the two by testing the
// string alone.
//
// A Resolver's ok=true result is trusted per its documented contract
// (resolve.Resolver): the host controls resolution, so its URL is
// returned verbatim — no scheme filtering, no percent-encoding.
// Everything else — no Resolver installed, or the Resolver declined
// with ok=false — takes resolve.DefaultResolution (wikilink targets get
// ".md" appended; other destinations pass through unchanged) and is
// filtered by resolve.SafeURL unless unsafe is set. Allowed
// default-path destinations are percent-encoded to match the CommonMark
// reference output, except wikilink targets: they're a filesystem path
// resolved by the host, not a URI, and encoding would corrupt e.g.
// spaces in page names. Backslash-escapes and entity references in
// ordinary link/image destinations are already decoded by the parser
// (transform.go); for autolinks — where CommonMark forbids that
// decoding — the raw text flows through untouched, so the escape only
// ever adds percent-encoding, never a second decode pass.
func Href(res resolve.Resolver, unsafe bool, kind resolve.ResolveKind, dest string) (string, bool) {
	if res != nil {
		if u, ok := res(kind, dest); ok {
			return u, true
		}
	}
	u := resolve.DefaultResolution(kind, dest)
	if !unsafe && !resolve.SafeURL(u) {
		return "", false
	}
	if kind != resolve.ResolveWikiLink {
		u = string(util.URLEscape([]byte(u), false))
	}
	return u, true
}

// Footnote pairs one footnote definition with its reference sites.
type Footnote struct {
	Def      *document.FootnoteDef
	RefCount int // number of [^label] reference sites carrying Def's index
}

// Footnotes derives a document's footnote section: the definitions in
// doc.Footnotes order (first-reference order, established by the
// parser), each paired with the count of its reference sites. The
// def/ref pairing itself is index-based — a FootnoteRef's Index matches
// its definition's — so renderers derive anchors/keys from the shared
// Index, never from position. Reference sites are counted across the
// document body and the footnote bodies themselves (a footnote may
// reference another footnote). Returns nil when the document has no
// footnotes.
func Footnotes(doc *document.Document) []Footnote {
	if len(doc.Footnotes) == 0 {
		return nil
	}
	counts := make(map[int]int)
	count := func(n document.Node) {
		document.Walk(n, func(n document.Node, entering bool) document.WalkStatus {
			if entering {
				if ref, ok := n.(*document.FootnoteRef); ok {
					counts[ref.Index]++
				}
			}
			return document.Continue
		})
	}
	count(doc)
	for _, def := range doc.Footnotes {
		count(def)
	}
	out := make([]Footnote, 0, len(doc.Footnotes))
	for _, def := range doc.Footnotes {
		out = append(out, Footnote{Def: def, RefCount: counts[def.Index]})
	}
	return out
}

// sanitizePolicy is the single policy for raw HTML embedded in
// markdown: bluemonday's UGC policy (basic formatting, links, images;
// no scripts, event handlers, or iframes). It lives here so every
// renderer sanitizes with the same policy.
var sanitizePolicy = bluemonday.UGCPolicy()

// SanitizeHTML sanitizes raw markdown HTML with the project's shared
// bluemonday UGC policy, stripping scripts, event handlers, and other
// XSS vectors.
func SanitizeHTML(s string) string { return sanitizePolicy.Sanitize(s) }
