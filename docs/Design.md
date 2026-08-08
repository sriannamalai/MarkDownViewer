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
                                                     │  internal/     │
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
| `markdownviewer` (root) | Public facade: `Parse`, `Render`, `RenderTo`, functional `Option`s. What most callers import. |
| `document` | The AST: `Node` interface, concrete node types (`Heading`, `List`, `Link`, …), `Walk`, `PlainText`, `Dump` (debug printer). Zero non-stdlib imports. |
| `parser` | Markdown → `document.Document`. Wraps goldmark; owns the goldmark→document tree transform, heading-slug generation, and the math/admonition extensions. `Config` toggles individual syntax extensions. |
| `render/html` | `document.Document` → HTML. Sanitization (bluemonday), URL policy, page assembly, chroma syntax highlighting, resolver hook. |
| `theme` | CSS custom-property theme definitions (`light`, `dark`) layered over a shared base stylesheet. |
| `internal/assets` | Embedded, offline copies of KaTeX and mermaid (JS/CSS), inlined at build time. Not part of the public API. |
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

## Feature set

CommonMark (via goldmark) plus: GFM tables, strikethrough, task lists,
autolinks (`Linkify`); footnotes; definition lists; YAML/TOML/JSON front
matter; `:emoji:` shortcodes; `[[wiki-links]]`; inline/display TeX math
(`$...$`, `$$...$$`, or ` ```math ` fences); GitHub-style admonitions
(`> [!NOTE]` etc.); mermaid diagrams (` ```mermaid ` fences); syntax
highlighting for 250+ languages via chroma; deterministic, deduplicated
heading anchor IDs. Every extension beyond plain CommonMark is individually
toggleable via `parser.Config` (not yet exposed through the facade — see
Roadmap).

## Output contract

`Render`/`RenderTo` produce either:

- **A full page** (default): `<!doctype html>` through `</html>`, an
  embedded `<style>` with the resolved theme + base CSS + (if the document
  uses them) KaTeX CSS, and inline `<script>` blocks for mermaid/KaTeX
  *only if* the document actually contains diagram/math nodes. Nothing is
  fetched from a CDN — every asset is embedded at build time
  (`internal/assets`, sourced by `scripts/fetch-assets.sh` and inlined by
  `scripts/inlinefonts`).
- **A fragment** (`Fragment()` / `-fragment`): body-only markup, no
  `<html>`/`<head>`/`<style>`/`<script>` at all. The host owns the page.
  Diagram/math nodes still emit their markup (`<pre class="mermaid">`,
  `<span class="math ...">`) but render as inert placeholders — raw source
  visible — until the host loads its own mermaid.js/KaTeX. See the
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
- **Resource exhaustion is a known, documented gap, not a mitigated one.**
  See SECURITY.md's "Resource exhaustion" section: deeply nested list input
  can take goldmark's parser tens of seconds on a small input, and there is
  currently no built-in timeout. Hosts handling untrusted input need to
  apply their own wall-clock bound.

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

Toward v1.0, roughly in order:

1. **Source positions on nodes.** `document` nodes currently carry no
   source-location information. Adding it (offsets or line/column,
   optionally behind a parse option to avoid paying for it when unused) is
   the highest-priority addition — hosts doing live-preview scroll-sync or
   inline diagnostics need it.
2. **JSON serialization + pinned `Kind` values.** A `document.Document` →
   JSON (and back) path, with `Kind` constants pinned to stable numeric or
   string values suitable for a wire format, so non-Go consumers (a future
   WASM/FFI host, a separate renderer process) can work from the same tree
   without linking Go.
3. **`parser.Config` exposed through the facade, plus a `RenderDoc`-style
   entry point.** Today `parser.Config`'s per-extension toggles aren't
   reachable from `markdownviewer`'s functional-option API, and there's no
   facade function that takes an already-parsed `*document.Document`
   straight to `Render` (skipping re-parsing). Both are straightforward;
   neither has shipped yet.
4. **Exported asset bundle for fragment hosts.** `internal/assets`
   (mermaid.js, KaTeX JS/CSS) is currently unexported — a fragment host
   that wants offline mermaid/KaTeX support has to source its own copies.
   Exporting a stable accessor removes that duplication.
5. **Context-aware `Parse`/`Render` variants.** Given the resource-exhaustion
   gap documented in `SECURITY.md`, `context.Context`-accepting variants
   that can be cancelled mid-parse/render would let hosts enforce a
   deadline without the goroutine-plus-timer workaround described there.
6. **C-shared FFI (`.so`/`.dylib`/`.dll`) + WASM builds**, once the model
   above is stable enough to commit to a C ABI / JS boundary.
7. **Mobile bindings** (`gomobile` / Flutter FFI) on top of the FFI layer.
8. **A native render-tree renderer** — for toolkit-native (non-webview)
   hosts, rendering directly from `document.Document` instead of through
   HTML.

Every step above consumes (and, where relevant, extends) the same
`document` model the HTML renderer uses today; none requires rearchitecting
the parsing layer.
