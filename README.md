# MarkDownViewer

A modern, embeddable Markdown viewer library for Go. Feed it Markdown, get
back self-contained, themed, sanitized HTML — no CDN calls, no external
assets, safe to render untrusted input by default. Built around a document
model that sits between parsing and rendering, so the HTML renderer shipped
today and native renderers planned for later both consume the same typed
AST.

## Status

**v0.x — the API is not yet frozen.** The `document` model is designed to
become the stable contract for future renderers and bindings, but it hasn't
earned that status yet: expect additive evolution (source positions on
nodes, a JSON serialization) before a v1.0 that commits to compatibility.
See [`docs/Design.md`](docs/Design.md) for the architecture and roadmap, and
[`CHANGELOG.md`](CHANGELOG.md) for release notes.

## Features

| Feature | Notes |
| --- | --- |
| CommonMark + GFM | 652/652 official CommonMark spec examples verified, plus GFM extras (tables, strikethrough, task lists, autolinks) |
| Footnotes | `[^1]`-style references and definitions |
| Heading anchors | Stable `id` attributes for deep linking |
| Front-matter | YAML front-matter parsed as document metadata |
| Emoji | `:shortcode:` support |
| Definition lists | `Term` / `: Description` syntax |
| Admonitions / callouts | `> [!NOTE]`-style blocks |
| Wiki-links | `[[Page]]` / `[[Page\|Text]]` |
| TeX math | Inline and block math via KaTeX, rendered offline |
| Mermaid diagrams | Flowcharts, sequence diagrams, etc., rendered offline |
| Syntax highlighting | chroma, pure Go, 250+ languages |
| Themes | Built-in light / dark / auto, with CSS custom-property overrides or a full stylesheet swap |
| Safe by default | HTML sanitized via bluemonday, URL scheme allowlist, opt-in `AllowRawHTML()` escape hatch |
| Host-controlled resolution | Pluggable `Resolver` for rewriting link/image/wiki-link targets |
| Offline output | Every asset (KaTeX, mermaid, fonts) is embedded — rendered HTML has zero external dependencies |

## Install

```bash
go get github.com/sriannamalai/markdownviewer
```

Requires Go 1.25+.

## Quick start

```go
package main

import (
	"fmt"

	markdownviewer "github.com/sriannamalai/markdownviewer"
)

func main() {
	html, err := markdownviewer.Render([]byte("# Hi\n"))
	if err != nil {
		panic(err)
	}
	fmt.Println(string(html))
}
```

Options compose as functional options on `Render` / `RenderTo`:

```go
out, err := markdownviewer.Render(
	src,
	markdownviewer.WithTheme("dark"),
	markdownviewer.Fragment(),        // body-only HTML, no <html>/<head> wrapper
	markdownviewer.AllowRawHTML(),    // trust the input; disables sanitization
	markdownviewer.DisableMath(),     // skip KaTeX
)
```

If you only need the parsed document model — for example to build your own
renderer — use `Parse`:

```go
doc, err := markdownviewer.Parse(src)
```

`document.Document` and its node types are documented in the `document`
package and are the intended long-term contract: renderers only ever depend
on it, never on parser internals. It is not API-frozen yet — see Status
above and the roadmap in [`docs/Design.md`](docs/Design.md).

### Rewriting links and images

Hosts that need to resolve relative paths, rewrite `wiki-links`, or route
images through an asset pipeline can supply a `Resolver`:

```go
out, err := markdownviewer.Render(src, markdownviewer.WithResolver(
	func(kind markdownviewer.ResolveKind, target string) (url string, ok bool) {
		if kind == markdownviewer.ResolveWikiLink {
			return "/wiki/" + target, true
		}
		return "", false // fall back to default handling
	},
))
```

**Trust contract:** URLs a `Resolver` returns with `ok=true` are emitted
as-is, without scheme filtering. The library assumes hosts fully control
resolution and will not echo untrusted targets back unexamined — see
`SECURITY.md`.

## CLI usage

The `mdview` command renders a file or stdin to HTML. Flags come before the
positional file argument:

```bash
mdview -o out.html README.md          # file to file
mdview -theme dark README.md          # force dark theme
cat notes.md | mdview -fragment       # stdin to stdout, body-only fragment
mdview -unsafe notes.md               # trust the input: raw HTML, all schemes
```

Full flag list: `-o FILE` (default stdout), `-theme light|dark|auto`
(default `auto`), `-fragment`, `-unsafe`, `-no-mermaid`, `-no-math`,
`-no-highlight`.

## Theming

Built-in themes are CSS custom-property sets layered over one base
stylesheet: `light`, `dark`, and `auto` (light by default, with a
`prefers-color-scheme` media query for dark). Override individual variables
without replacing the whole stylesheet:

```go
out, err := markdownviewer.Render(src, markdownviewer.WithThemeOverrides(map[string]string{
	"--md-bg":     "#f8f8f2",
	"--md-accent": "#ff79c6",
}))
```

Or replace the base stylesheet entirely:

```go
out, err := markdownviewer.Render(src, markdownviewer.WithStylesheet(myCSS))
```

Both are emitted into the page's `<style>` element; `</style` sequences in
supplied content are stripped defensively.

## Security model

- **Sanitized by default.** Raw HTML in Markdown input is passed through
  bluemonday's UGC policy unless `AllowRawHTML()` / `-unsafe` is set.
- **URL scheme allowlist.** Only `http`, `https`, `mailto`, and `tel` are
  permitted in links/images by default; everything else (including
  `javascript:`, `data:`, and unknown schemes) is blocked. `AllowRawHTML()`
  lifts this restriction.
- **Resolver trust boundary.** A host-supplied `Resolver` fully controls
  its own output — see the trust contract above.
- **Offline, self-contained output.** Rendered HTML never fetches from a
  CDN, so there's no third-party request surface at render or view time.

See `SECURITY.md` for the vulnerability reporting process and what's in
scope (sanitizer bypasses and URL scheme allowlist bypasses are explicitly
in scope).

## Roadmap

v0.1 ships the Go package API and the `mdview` CLI on today's `document`
model, ahead of an eventual v1.0 that commits to API stability. See
[`docs/Design.md`](docs/Design.md#roadmap) for the full roadmap — source
positions on nodes, JSON serialization, an exported asset bundle for
fragment hosts, context-aware parse/render, C-shared FFI + WASM builds,
mobile bindings, and a native render-tree renderer, roughly in that order.

## License

Apache-2.0 — see [LICENSE](LICENSE).

This project bundles third-party software (goldmark, chroma, bluemonday,
mermaid, KaTeX). See [NOTICE](NOTICE) and
[third_party/README.md](third_party/README.md) for attributions, versions,
and license details.
