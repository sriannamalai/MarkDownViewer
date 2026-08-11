// Package resolve holds the renderer-agnostic destination-resolution
// policy: the Resolver hook and its ResolveKind discriminants, the
// safeURL scheme allowlist ([SafeURL]), and the built-in default
// resolution rule ([DefaultResolution], the wiki-link ".md" fallback).
//
// This is the single home of the project's URL security policy. Every
// renderer — the HTML renderer today, any future native render-tree
// renderer — must consume this package rather than fork its own copy:
// a change here changes the security posture of every renderer at once,
// which is exactly the point. render/html re-exports the types as
// aliases for compatibility, and the top-level markdownviewer facade
// re-exports them for hosts.
package resolve

// ResolveKind identifies which kind of destination is being resolved by a
// Resolver call.
type ResolveKind int

// These values are ABI-frozen: the FFI and WASM boundaries expose them
// as bare ints (0=link, 1=image, 2=wiki-link). Append-only — never
// reorder or renumber (see kinds_test.go here and
// internal/boundary/kinds_test.go).
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

// DefaultResolution is the built-in resolution rule applied when no
// Resolver is installed, or when the installed Resolver declined
// (ok=false): a wiki-link target gets ".md" appended — a bare page name
// resolves to its Markdown file, mirroring how wikis link pages — and
// every other destination passes through unchanged. The result is still
// subject to [SafeURL] filtering unless the renderer runs in unsafe
// mode.
func DefaultResolution(kind ResolveKind, target string) string {
	if kind == ResolveWikiLink {
		return target + ".md"
	}
	return target
}
