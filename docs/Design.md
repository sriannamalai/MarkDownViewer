# Design

An overview of what MarkDownViewer actually ships, for readers who want the
architecture rather than the API reference. See the package godoc for
exact signatures; this document explains how the pieces fit together and
why.

## Status

v0.x. The API works and is tested, but is not yet frozen — see the Roadmap
below for what's expected to change before v1.0.

## Pipeline

```
                    ┌─────────────┐
  Markdown bytes ──▶│   parser    │──▶ document.Document ──┐
                    │  (goldmark) │      (typed AST)        │
                    └─────────────┘                         │
                                                              ▼
                                                     ┌────────────────┐
                                                     │  render/html   │──▶ HTML
                                                     │ (+ theme, +    │
                                                     │  assets)       │
                                                     └────────────────┘
```

Two stages, one boundary in between:

1. **`parser`** turns Markdown source into a `document.Document`. It wraps
   [goldmark](https://github.com/yuin/goldmark) (CommonMark) plus goldmark
   extensions (GFM tables/strikethrough/task-lists/autolinks, footnotes,
   definition lists, front matter, emoji, wiki-links) and a couple of
   extensions written in this repo (math, admonitions). goldmark's own AST
   is walked once and translated into `document` types; nothing downstream
   ever sees a goldmark type.
2. **`render/html`** walks a `document.Document` and writes HTML, either a
   full self-contained page or a body-only fragment. It owns sanitization,
   the URL scheme allowlist, theme CSS assembly, and conditional inlining
   of the mermaid/KaTeX asset bundles.

`markdownviewer` (the repo-root package) is a thin facade over both:
`Parse` calls `parser.Parse`; `Render`/`RenderTo` chain `parser.Parse` into
`htmlrender.Render` with functional options translated into
`htmlrender.Options`.

## Package layout

| Package | Role |
| --- | --- |
| `markdownviewer` (root) | Public facade: `Parse`/`ParseWith`, `Render`/`RenderTo`, `RenderDoc`/`RenderDocTo`, `ParseContext`/`RenderContext`/`RenderDocContext`, functional `Option`s. What most callers import. |
| `document` | The AST: `Node` interface, concrete node types (`Heading`, `List`, `Link`, …), `Walk`, `PlainText`, `Dump` (debug printer). Zero non-stdlib imports. |
| `parser` | Markdown → `document.Document`. Wraps goldmark; owns the goldmark→document tree transform, heading-slug generation, and the math/admonition extensions. `Config` toggles individual syntax extensions. |
| `render/html` | `document.Document` → HTML. Sanitization (bluemonday), URL policy, page assembly, chroma syntax highlighting, resolver hook. |
| `theme` | CSS custom-property theme definitions (`light`, `dark`) layered over a shared base stylesheet. |
| `assets` | Embedded, offline copies of KaTeX and mermaid (JS/CSS), inlined at build time. Exported for fragment hosts to inject. |
| `cmd/mdview` | CLI: reads a file or stdin, calls the facade, writes HTML. |

## The layering rule

**`document` depends on nothing outside the standard library, and nothing
outside `parser` may import goldmark types.** The document model is the
only channel between parsing and rendering. This is enforced by convention
today (see the package doc comments on `document` and `parser`), not by a
lint rule — a future CI check that fails on a goldmark import outside
`parser` is a reasonable hardening, currently just discipline. The payoff:
`render/html` (and any future renderer — native, JSON, whatever) only ever
needs to understand `document` types, and swapping the underlying Markdown
engine would in principle only touch `parser`.

## Document model

Every `document.Node` carries a `Kind`, a fixed-width enum (`KindDocument`
through `KindFootnoteRef`, currently 0-31). Values are pinned and
append-only by convention — a serialization/FFI compatibility contract,
documented on the `const` block itself: existing values are never
renumbered, new node types only ever append. `Kind.String()` and
`document.KindFromString` map each value to a stable lowerCamel wire name
(`"blockQuote"`, `"codeBlock"`, …) independent of the Go identifier.

**Source spans.** `document.Span` (`StartLine`/`EndLine`, 1-based, plus
`StartOffset`/`EndOffset`, 0-based half-open byte offsets) locates a node in
the source it was parsed from; the zero `Span` means "position unknown".
Coverage in v0.2 is block-level only — inline nodes carry no span. Leaf
blocks (headings, paragraphs, code blocks, thematic breaks, …) get an exact
span from goldmark's own line info, inclusive of block markers (`#`,
` ``` `, blockquote `>`); container blocks (block quotes, lists, list
items, tables, definition lists) get the union of their direct children's
non-zero spans instead of their own markers. One node currently has no span
at all: `ThematicBreak` is a single-token leaf with no `Lines()` data from
goldmark to derive one from, so its `Span()` is always the zero value —
hosts that need `<hr>` positions must fall back to surrounding-node spans.
Offsets are relative to the source *after* parsing's up-front
normalization: a leading UTF-8 BOM is stripped and any invalid UTF-8 byte
sequences are replaced with U+FFFD before goldmark ever sees the bytes, so
offset 0 is always the first byte of the normalized (not necessarily the
original) input.

**JSON codec.** `document.MarshalJSON`/`UnmarshalJSON` round-trip a
`*Document` through a versioned wire format: a top-level envelope
(`{"version":1,"kind":"document",...}`), each node as
`{"kind":"<wire name>",...}` with lowerCamel field names, zero spans and
empty children arrays omitted to keep output compact. `Kind` travels as its
string name (not the numeric value) so the format stays readable and
resilient to Go-side renumbering that can never happen anyway. `Document.Meta`
(decoded front matter) must be JSON-marshalable as-is — it round-trips
JSON-native, so a YAML timestamp becomes an RFC 3339 string rather than a
Go `time.Time`. Invalid UTF-8 in the source is sanitized (replaced with
U+FFFD) before the parser ever builds node text, which is what keeps every
`Text`/`CodeSpan`/`HTMLBlock`/… string field already valid UTF-8 by the
time it reaches `encoding/json` — the codec never has to silently mangle a
string the way `json.Marshal` does for invalid UTF-8 on its own.

## Facade

The root `markdownviewer` package exposes two independent axes: parse vs.
render, and source-driven vs. already-parsed-tree-driven.

- **`Parse`/`ParseWith`** — Markdown bytes to `*document.Document`.
  `ParseWith` takes an explicit `parser.Config`; `Parse` uses
  `parser.Default()` (every extension on).
- **`Render`/`RenderTo`** — parse-then-render in one call, the common case.
  `WithParserConfig(cfg)` selects the `parser.Config` these two use
  internally (no effect on the `RenderDoc*` family below, which never
  parses).
- **`RenderDoc`/`RenderDocTo`** — render an already-parsed
  `*document.Document`, skipping re-parsing entirely. This is the
  parse-once/render-many path: parse a document once, then call `RenderDoc`
  repeatedly with different options (e.g. `WithTheme("dark")` vs.
  `WithTheme("light")`) without re-running the parser each time.
- **`WithSourceMap`** — opts `Render`/`RenderTo`/`RenderDoc`/`RenderDocTo`
  into `data-md-line="<n>"` attributes on top-level block elements (and
  footnote `<li>`s), sourced from each node's `Span.StartLine`. Two kinds
  of output are deliberately left unannotated because the renderer doesn't
  own the markup it emits for them: raw HTML blocks (the source markup
  passes through as-is) and chroma-highlighted code blocks (chroma emits
  its own `<pre>`/`<span>` structure). Off by default; the attribute exists
  to let a live-preview host scroll-sync its editor cursor to the rendered
  DOM node it produced.
- **`ParseContext`/`RenderContext`/`RenderDocContext`** — context-aware
  variants of the three families above. They honor `ctx`'s deadline: if
  `ctx` ends first, the call returns `ctx.Err()` promptly. What they do
  *not* do is reclaim the abandoned work — the underlying goroutine keeps
  running to completion (the Markdown engine has no cancellation hooks),
  and it may still be reading `src` (or `doc`, for `RenderDocContext`) for
  an unbounded window after the call returns. This bounds caller-observed
  latency, not CPU spend; callers must treat `src`/`doc` as immutable for
  the lifetime of the call and must not assume the goroutine has stopped
  just because the call returned. See the root package godoc's Concurrency
  section and `SECURITY.md` for the full contract and the resource-
  exhaustion background this is meant to help hosts work around.

### FFI boundary (`ffi/`, v0.4)

`ffi/` builds `libmdviewer` with `-buildmode=c-shared`. It is a leaf
consumer of the public facade: pure-Go operation/option logic in
cgo-free files (unit-tested), conversion-only cgo wrappers on top
(proven by the C harness in CI on all three OSes). The AST crosses the
boundary as the v0.2 versioned JSON — `mdv_parse` + `mdv_render_doc`
give parse-once/render-many without cross-boundary object lifetimes,
which is why no handle-based API exists. Options cross as a strict,
versioned JSON object; unknown fields are errors so binding typos
surface immediately. The `Resolver` callback does not cross the FFI
(function-pointer marshalling is deferred).

Every exported entry point runs its operation under a `recover`, so a Go
panic — from hostile document JSON or a bug in a parsing/rendering
dependency — becomes the documented non-zero return plus a
`panic:`-prefixed error string. A Go caller can isolate a panic itself;
an FFI caller cannot, and an unrecovered one would take down the whole
host process. One failure stays fatal by design: cgo's `malloc` wrapper
aborts the process on true allocation failure (an unrecoverable runtime
throw, not a panic) — the same behavior as the Go runtime itself running
out of memory.

## Feature set

CommonMark (via goldmark) plus: GFM tables, strikethrough, task lists,
autolinks (`Linkify`); footnotes; definition lists; YAML/TOML/JSON front
matter; `:emoji:` shortcodes; `[[wiki-links]]`; inline/display TeX math
(`$...$`, `$$...$$`, or ` ```math ` fences); GitHub-style admonitions
(`> [!NOTE]` etc.); mermaid diagrams (` ```mermaid ` fences); syntax
highlighting for 250+ languages via chroma; deterministic, deduplicated
heading anchor IDs. Every extension beyond plain CommonMark is individually
toggleable via `parser.Config`, reachable from the facade through
`markdownviewer.WithParserConfig` (`Parse`/`ParseWith` also take a `Config`
directly).

## Output contract

`Render`/`RenderTo` produce either:

- **A full page** (default): `<!doctype html>` through `</html>`, an
  embedded `<style>` with the resolved theme + base CSS + (if the document
  uses them) KaTeX CSS, and inline `<script>` blocks for mermaid/KaTeX
  *only if* the document actually contains diagram/math nodes. Nothing is
  fetched from a CDN — every asset is embedded at build time (the public
  `assets` package, sourced by `scripts/fetch-assets.sh` and inlined by
  `scripts/inlinefonts`).
- **A fragment** (`Fragment()` / `-fragment`): body-only markup, no
  `<html>`/`<head>`/`<style>`/`<script>` at all. The host owns the page.
  Diagram/math nodes still emit their markup (`<pre class="mermaid">`,
  `<span class="math ...">`) but render as inert placeholders — raw source
  visible — until the host loads its own mermaid.js/KaTeX, which it can
  source from the `assets` package (`assets.MermaidJS`, `assets.KatexJS`,
  `assets.KatexCSS`) instead of vendoring separate copies. See the
  Concurrency/Fragment notes in the root package godoc and README.

## Theming

Themes are CSS custom-property sets (`theme.Light()`, `theme.Dark()`)
layered over one shared base stylesheet (`theme.BaseCSS()`, `theme/base.css`).
`"auto"` emits the light variables at `:root` plus a
`prefers-color-scheme: dark` media query carrying the dark variables — no
JS theme switch needed. Hosts can override individual `--md-*` custom
properties (`WithThemeOverrides`) or replace the base stylesheet outright
(`WithStylesheet`); both paths strip `</style` sequences defensively before
emission.

## Safety model

- **Sanitize by default.** Raw HTML in Markdown input goes through
  bluemonday's UGC policy (`render/html/sanitize.go`) unless
  `AllowRawHTML()`/`-unsafe` is set. The policy is constructed once at
  package init and never mutated afterward — bluemonday documents that a
  constructed `Policy` is safe to call `Sanitize` on concurrently (only
  construction/editing is not).
- **URL scheme allowlist.** `render/html/url.go`'s `safeURL` allows only
  `http`, `https`, `mailto`, `tel`; everything else — `javascript:`,
  `data:`, `vbscript:`, unknown schemes — is blocked by default. Control
  characters are stripped before scheme inspection specifically to close
  whitespace-splicing bypasses (`jav\tascript:`) that a naive prefix
  blocklist would miss.
- **Resolver trust contract.** A host-supplied `Resolver` fully controls
  its own return value: a URL returned with `ok=true` is emitted as-is,
  bypassing `safeURL`. This is deliberate — a `Resolver` exists precisely
  to let a host redirect targets somewhere the default policy wouldn't
  allow (e.g. rewriting wiki-links to an internal scheme) — but it means a
  `Resolver` that echoes attacker-controlled input back unexamined
  reintroduces exactly the class of bug the default policy prevents. See
  `SECURITY.md`.
- **Host CSS hardening.** `WithThemeOverrides` and `WithStylesheet` content
  is emitted into the page's `<style>` element with `</style` sequences
  stripped case-insensitively, closing the straightforward style-element
  breakout. This is not full CSS sanitization — both are host-supplied
  inputs, not untrusted Markdown content, so the trust boundary is the
  same as for a `Resolver`: the host controls what it passes in.
- **Resource exhaustion is a known, documented gap, only partially
  mitigated.** See SECURITY.md's "Resource exhaustion" section: deeply
  nested list input can take goldmark's parser tens of seconds on a small
  input, and goldmark itself has no built-in timeout. `ParseContext`/
  `RenderContext`/`RenderDocContext` give hosts a bounded-latency escape
  hatch — the call returns on `ctx` expiry instead of blocking until the
  parse finishes — but they do not reclaim the abandoned goroutine's CPU,
  so they bound caller latency, not total resource spend. Hosts handling
  untrusted input still need their own capacity planning around that gap.

## Testing approach

- **CommonMark conformance.** `parser/spec_test.go` runs the full official
  CommonMark 0.31.2 spec suite (652 examples) against `parser.CommonMarkOnly()`
  and asserts 652/652 pass, plus a small GFM-extras suite (tables,
  strikethrough, etc.) against `parser.Default()`. Any intentional
  deviation must be listed in `commonMarkSkips` with a reason and is capped
  (`maxSkips`) so a skip can't silently accumulate.
- **Golden HTML tests.** `render/html/golden_test.go` renders fixed
  Markdown fixtures (`render/html/testdata/*.md`) and diffs against
  checked-in `.golden.html` files (regenerate with `-update`).
- **XSS corpus.** `render/html/testdata/xss.txt` is a list of known-hostile
  Markdown/HTML snippets (script tags, event-handler attributes,
  `javascript:`/`vbscript:` URLs, control-character scheme-splicing
  bypasses, mutation-XSS-style malformed markup); `sanitize_test.go` runs
  every line through the default render path and asserts none of the
  dangerous constructs survive.
- **Fuzzing.** `parser/fuzz_test.go`'s `FuzzParseRender` seeds `go test
  -fuzz` with representative snippets covering every syntax extension and
  round-trips them through `Parse` → fragment `Render`, catching panics and
  crashes across the whole pipeline (not correctness — that's the spec
  suite's job).
- **Resource-exhaustion regression guard.** `parser/dos_test.go` parses a
  moderately (not pathologically) deep nested list under a generous
  wall-clock watchdog, to catch a regression that would make even moderate
  nesting pathologically slow, without making CI flaky or slow itself.

## Roadmap

v0.2 shipped the additive model work the earlier roadmap prioritized:
block-level source spans, the versioned JSON codec with pinned `Kind`
values, `parser.Config` reachable from the facade (`WithParserConfig`) plus
the `RenderDoc`/`RenderDocTo` parse-once/render-many entry points, an
exported `assets` package, and context-aware `Parse`/`Render`/`RenderDoc`
variants. See the Document model and Facade sections above for what
shipped, and `CHANGELOG.md` for the release notes.

v0.4 shipped the C-shared FFI (`.so`/`.dylib`/`.dll`), now that the
`document` model has a stable wire format (JSON, pinned `Kind` values) to
build a C ABI on top of — see the "FFI boundary" subsection above.

Toward v1.0, what remains, roughly in order:

1. **WASM builds**, reusing the same JSON boundary the C-shared FFI
   established.
2. **Mobile bindings** (`gomobile` / Flutter FFI) on top of the FFI layer.
3. **A native render-tree renderer** — for toolkit-native (non-webview)
   hosts, rendering directly from `document.Document` instead of through
   HTML.
4. **Incremental rendering**, if profiling on real host workloads shows
   full re-render (even with parse-once/`RenderDoc`) is the bottleneck —
   not committed to; only worth doing if the numbers demand it.

Every step above consumes (and, where relevant, extends) the same
`document` model the HTML renderer uses today; none requires rearchitecting
the parsing layer.
