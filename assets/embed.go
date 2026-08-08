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

//go:embed katex.min.js
var katexJS string

//go:embed katex.inline.css
var katexCSS string

func MermaidJS() string { return mermaidJS }
func KatexJS() string   { return katexJS }
func KatexCSS() string  { return katexCSS }
