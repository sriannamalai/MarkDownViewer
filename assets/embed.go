// Package assets exposes the vendored mermaid and KaTeX payloads that
// full-page rendering embeds automatically. Fragment-mode hosts own their
// page: to activate diagram and math placeholders, inject MermaidJS and
// KatexJS in <script> tags, KatexCSS in a <style> tag, and initialize as
// render/html's full-page output does (mermaid.initialize; katex.render
// over .math elements). Versions and licenses: third_party/README.md.
package assets

import _ "embed"

//go:embed mermaid.min.js
var mermaidJS string

//go:embed mermaid-bridge.js
var mermaidBridgeJS string

//go:embed katex.min.js
var katexJS string

//go:embed katex.inline.css
var katexCSS string

// MermaidJS returns the vendored mermaid.js bundle contents. Pinned
// version: see the mermaid entry in third_party/README.md.
func MermaidJS() string { return mermaidJS }

// MermaidBridgeJS returns mdvRenderMermaid, a first-cut primitive for
// hosts that only have an offscreen/headless webview (no visible DOM
// to scan the way full-page HTML's startOnLoad does): given the
// vendored mermaid.js bundle already loaded in the same JS context, it
// exposes a single documented function that renders one diagram's
// source to an SVG string on demand. See mermaid-bridge.js's own
// comment for the exact call contract.
func MermaidBridgeJS() string { return mermaidBridgeJS }

// KatexJS returns the vendored KaTeX JS bundle contents. Pinned version:
// see the KaTeX entry in third_party/README.md.
func KatexJS() string { return katexJS }

// KatexCSS returns the vendored KaTeX CSS, with fonts inlined as data:
// URIs. Pinned version: see the KaTeX entry in third_party/README.md.
func KatexCSS() string { return katexCSS }
