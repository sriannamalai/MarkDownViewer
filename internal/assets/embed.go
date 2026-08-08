// Package assets embeds the vendored third-party viewer assets. See
// third_party/README.md for versions and licenses.
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
