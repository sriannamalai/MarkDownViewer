# MarkDownViewer

A modern, embeddable Markdown viewer library for Go. Feed it Markdown, get
back self-contained, themed, sanitized HTML — no CDN calls, no external
assets, safe to render untrusted input by default. Built around a document
model that sits between parsing and rendering, so the HTML renderer shipped
today and native renderers planned for later both consume the same typed
AST.

## Status

**v0.x — the API is not yet frozen.** The `document` model is designed to
become the stable contract for future renderers and bindings; it now
carries block-level source spans, pinned `Kind` values, and a versioned
JSON codec, but hasn't earned frozen-API status yet — expect further
additive evolution before a v1.0 that commits to compatibility.
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
| Source map | Opt-in `data-md-line` attributes (`WithSourceMap()`) for editor↔preview scroll sync |
| JSON document tree | `document.MarshalJSON`/`UnmarshalJSON` — versioned wire format with pinned `Kind` names |

## Install

```bash
go get github.com/sriannamalai/markdownviewer
```

Requires Go 1.26+.

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

### Editor integration

Live-preview hosts (an editor pane rendering Markdown as you type) tend to
need two things: a way to map rendered DOM back to source lines, and a way
to avoid re-parsing on every keystroke or theme flip.

**Scroll sync via `WithSourceMap()`.** Top-level block elements (and
footnote `<li>`s) get a `data-md-line="<n>"` attribute pointing at the
1-based source line the block started on:

```go
out, err := markdownviewer.Render(src, markdownviewer.WithSourceMap())
```

```html
<h1 data-md-line="1">Title</h1>
<p data-md-line="3">Some text.</p>
```

An editor can scroll the preview to the block whose `data-md-line` is
closest to the cursor line, or do the reverse on click. Two kinds of output
are deliberately left unannotated because the renderer doesn't own the
markup it emits for them: raw HTML blocks, and chroma-highlighted code
blocks (chroma emits its own `<pre>`/`<span>` structure).

**Parse once, render many with `RenderDoc`.** Re-parsing on every theme
switch is wasted work when the source hasn't changed — parse once with
`Parse`, then render the same tree repeatedly with `RenderDoc`/`RenderDocTo`:

```go
doc, err := markdownviewer.Parse(src)
if err != nil {
	panic(err)
}

light, err := markdownviewer.RenderDoc(doc, markdownviewer.WithTheme("light"))
dark, err := markdownviewer.RenderDoc(doc, markdownviewer.WithTheme("dark"))
```

`doc` must not be mutated while a `RenderDoc`/`RenderDocTo` call is reading
it — see the Concurrency section below.

## CLI usage

The `mdview` command renders a file or stdin to HTML. Flags come before the
positional file argument:

```bash
mdview -o out.html README.md          # file to file
mdview -theme dark README.md          # force dark theme
cat notes.md | mdview -fragment       # stdin to stdout, body-only fragment
mdview -unsafe notes.md               # trust the input: raw HTML, all schemes
mdview -width 860px README.md         # constrain content width (default is fluid)
```

Full flag list: `-o FILE` (default stdout), `-theme light|dark|auto`
(default `auto`), `-fragment`, `-unsafe`, `-no-mermaid`, `-no-math`,
`-no-highlight`, `-width STRING` (any CSS length, e.g. `860px` or `70ch`;
default is fluid — no max-width).

Running `mdview` with no file argument and no piped input (stdin is an
interactive terminal) prints name/version, flag defaults, and a few example
invocations instead of blocking on a read that would never complete. Piped
or redirected stdin (`cat notes.md | mdview`) is unaffected — that still
renders normally.

## Concurrency

All top-level functions (`Parse`, `ParseWith`, `Render`, `RenderTo`,
`RenderDoc`, `RenderDocTo`, `ParseContext`, `RenderContext`,
`RenderDocContext`) are safe for concurrent use — they share no mutable
state beyond two package-level values, constructed once and never mutated
afterward: bluemonday's sanitizer `Policy` (documented safe to `Sanitize`
concurrently once constructed) and chroma's HTML formatter (internally
mutex-protected for concurrent `Format` calls). The one thing that isn't
synchronized: a `*document.Document` returned by `Parse`/`ParseWith` must
not be mutated while it's being read by a concurrent
`Render`/`RenderTo`/`RenderDoc`/`RenderDocTo` call, or by another mutation
— `document.Document` has no internal locking of its own.

The `Context` variants extend this contract rather than relaxing it: when
`ctx` ends before the underlying work finishes, the function returns
`ctx.Err()` immediately, but the abandoned goroutine may still be reading
`src` (`ParseContext`/`RenderContext`) or `doc` (`RenderDocContext`) for an
unbounded window afterward — there's no signal for when it actually stops.
Getting `ctx.Err()` back is not a guarantee those inputs are safe to mutate
or reuse; treat `src`/`doc` passed to a `Context` variant as immutable for
the lifetime of the call, and don't assume that lifetime has ended just
because the call returned. `ParseContext`/`RenderContext`/`RenderDocContext`
bound caller-observed latency, not CPU spend — the Markdown engine has no
cancellation hooks, so an abandoned parse/render keeps running to
completion. See SECURITY.md for the resource-exhaustion background.

## Theming

Fragment output (`Fragment()` / `mdview -fragment`) contains **no CSS or
JS** — it's body-only markup. The host owns the page and must supply its
own styling; `theme.BaseCSS()` and the theme CSS-variable sets below are
there to reuse if useful. Diagram and math nodes still emit their markup in
fragment mode (`<pre class="mermaid">…</pre>`, `<span class="math …">…`),
but as **inert placeholders**: without mermaid.js/KaTeX loaded, a viewer
sees the raw diagram/math source until the host supplies those libraries
itself.

Fragment mode emits no CSS/JS. To activate mermaid/math in your own page:

```go
import "github.com/sriannamalai/markdownviewer/assets"
// once per page:
fmt.Fprintf(w, "<style>%s</style><script>%s</script><script>%s</script>",
    assets.KatexCSS(), assets.KatexJS(), assets.MermaidJS())
```

Fragment hosts also need the syntax-highlighting stylesheet for code blocks — `htmlrender.HighlightCSS(theme.Light())` / `(theme.Dark())` in Go, or the composed `theme-light.css` / `theme-dark.css` assets over the C ABI, which bundle the theme tokens and highlight CSS together.

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

Or append CSS after whatever base is in effect — the built-in base+theme
styling by default, or a `WithStylesheet` replacement — without replacing
it (contrast with `WithStylesheet`, which swaps the base out entirely):

```go
out, err := markdownviewer.Render(src, markdownviewer.WithExtraCSS(
	".md-code { border-radius: 8px; }"))
```

All three are emitted into the page's `<style>` element; `</style`
sequences in supplied content are stripped defensively.

Code blocks can opt into a header row carrying the fence language and a
Copy button:

```go
out, err := markdownviewer.Render(src, markdownviewer.WithCodeHeader())
```

Full pages also get a small inline clipboard script wiring the buttons;
fragment hosts receive the markup only and wire their own click handler.

The default layout is fluid — no `max-width` constraint, so the page fills
its container. Opt in to a constrained width with `WithMaxWidth` (any CSS
length, e.g. `"860px"` or `"70ch"`), implemented as a `--md-max-width`
theme override:

```go
out, err := markdownviewer.Render(src, markdownviewer.WithMaxWidth("860px"))
```

An empty string (the default) stays fluid. Since the value flows into a CSS
custom-property declaration, one containing `;` or `}` is rejected
defensively and the option no-ops rather than emitting a defanged value.

## Embedding from other languages

Every release ships prebuilt C-shared libraries (`libmdviewer`) for macOS
(arm64/x86_64), Linux (amd64/arm64), and Windows (amd64) — see the
release's `libmdviewer-*.zip` assets. Nine thread-safe symbols — this is
unchanged by the resolver callback, which runs synchronously on the
calling thread: `mdv_render` (Markdown → HTML), `mdv_render_r` (same,
plus a host `Resolver` callback), `mdv_parse` (Markdown → versioned
document-AST JSON), `mdv_render_doc` (AST JSON → HTML, for
parse-once/render-many), `mdv_render_doc_r` (same, plus a `Resolver`
callback), `mdv_asset` (embedded assets: mermaid/KaTeX bundles,
theme+highlight CSS), `mdv_alloc` (library-heap allocator, for the
resolver's returned URL), `mdv_free`, and `mdv_version`. Options cross
the boundary as a small JSON object mirroring this package's functional
options. Fragment-mode hosts can pull the embedded mermaid/KaTeX bundles
and per-theme highlight CSS over the boundary via `mdv_asset` (v0.5); Go
fragment hosts get the same via the `assets` package and
`htmlrender.HighlightCSS`.

```c
char *html = NULL, *err = NULL; size_t n = 0;
if (mdv_render(md, strlen(md), "{\"theme\": \"dark\"}", &html, &n, &err) == 0) {
    fwrite(html, 1, n, stdout);
    mdv_free(html);
} else {
    fprintf(stderr, "%s\n", err);
    mdv_free(err);
}
```

Each release zip bundles `ffi/README.md` as `README.md` alongside the
library and header — that's the API reference (ownership: every returned
buffer is freed with `mdv_free`). Working consumers: `examples/c/`
(the CI-gated harness) and `examples/dart/` (`dart:ffi`, the pattern a
Flutter host uses). To build locally: `./scripts/build-ffi.sh`.

Browser and Node hosts get the same rendering over WebAssembly instead of
a C ABI: every release also ships `libmdviewer-<version>-wasm.zip`, an
npm-ready ESM package (`import { loadMdviewer } from 'libmdviewer'`) with
no bundler or native dependency required. See
[`wasm/npm/README.md`](wasm/npm/README.md) for the JS API, and
`./scripts/build-wasm.sh` to build locally.

Flutter/mobile hosts get a plugin, `flutter/mdviewer`, over the same
nine-symbol C ABI instead of a new binding layer — static on iOS,
`c-shared` on Android, consumed via `dart:ffi` with `NativeCallable` for
the `Resolver` callback. Every release also ships the two mobile
artifacts the plugin's `tool/fetch_binaries.sh` pulls down:
`libmdviewer-<version>-ios.xcframework.zip` and
`libmdviewer-<version>-android.zip`. See
[`flutter/mdviewer/README.md`](flutter/mdviewer/README.md) for the Dart
API, and `./scripts/build-mobile.sh` to build locally.

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

v0.2 adds block-level source spans, a versioned JSON codec for the
`document` tree, `RenderDoc`/`ParseWith`/`WithParserConfig`, an exported
`assets` package, and context-aware parse/render variants on top of v0.1's
Go package API and `mdview` CLI. v0.4 adds the C-shared `libmdviewer` FFI
described above, v0.6 adds the WASM build and npm package described
above plus the `Resolver` callback over both the C ABI and WASM, and
v0.7 adds the Flutter/mobile plugin described above, ahead of an
eventual v1.0 that commits to API stability. See
[`docs/Design.md`](docs/Design.md#roadmap) for what remains — a native
render-tree renderer and incremental rendering if profiling demands it.

## License

Apache-2.0 — see [LICENSE](LICENSE).

This project bundles third-party software (goldmark, chroma, bluemonday,
mermaid, KaTeX). See [NOTICE](NOTICE) and
[third_party/README.md](third_party/README.md) for attributions, versions,
and license details.
