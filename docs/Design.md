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
                                             ┌──────────────┴───────────────┐
                                             ▼                              ▼
                                    ┌────────────────┐            ┌────────────────┐
                                    │  render/html   │──▶ HTML    │  render/tree   │──▶ render tree
                                    │ (+ theme, +    │            │ (shared derive │      (JSON)
                                    │  assets)       │            │  + resolve)    │
                                    └────────────────┘            └────────────────┘
```

Two stages, one boundary in between (shown here with both renderers —
the HTML renderer described next, and the v0.10 native render tree
covered in its own section below):

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
| `render/tree` | `document.Document` → the version-1 native render tree (`tree.Build`): a layout-free, fully resolved semantic tree as strict JSON, for hosts that render platform widgets instead of HTML. |
| `resolve` | Renderer-agnostic resolution policy: `Resolver`, the ABI-frozen `ResolveKind` ints, the safe-URL scheme allowlist, and the wiki-link default resolution — consumed by both renderers. |
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
Leaf blocks (headings, paragraphs, code blocks, …) get an exact span from
goldmark's own line info, inclusive of block markers (`#`, ` ``` `,
blockquote `>`); container blocks (block quotes, lists, list items,
tables, definition lists) get the union of their direct children's
non-zero spans instead of their own markers. `ThematicBreak` — which had
no span in v0.2 because goldmark's stock parser leaves its `Lines()`
empty — gets a real, marker-inclusive line span since v0.9 via a
span-recording replacement block parser (`parser/tbreak.go`) registered
between goldmark's setext-heading and stock thematic-break parsers with
an identical match predicate, so parse precedence is unchanged (pinned by
the CommonMark suite). With `SourceMap` on, `<hr>` therefore now carries
`data-md-line` like every other top-level block.

Inline nodes carry spans since v0.9, with an honesty caveat on delimited
containers. goldmark's inline AST keeps source segments only for content
(text runs, raw-HTML segments) and drops the delimiter tokens, so:

- **Exact spans:** `Text` (its merged segment), `HTMLInline` (raw
  segments cover the markup itself), `MathInline` (our own inline math
  parser records the full `$…$`/`$$…$$` range, delimiters included).
- **Content-only spans:** `Emphasis`, `Strong`, `Strikethrough`,
  `CodeSpan`, `Link`, `Image`, `WikiLink` span the union of their
  content — the visible inner text — NOT the `*`/`**`/`~~`/backtick/
  `[](dest)`/`[[]]` delimiters, whose positions are not recoverable from
  goldmark and are deliberately not guessed. A `Link`'s span covers its
  label text only, an `Image`'s its alt text, a `WikiLink`'s the label
  half after any `|`.
- **No span (zero value):** autolinks (`<https://…>` and linkified bare
  URLs — goldmark's `AutoLink` node hides its segment in an unexported
  field), synthesized `Text` (emoji shortcode expansions, autolink
  labels), `SoftBreak`/`HardBreak`, and `FootnoteRef`.

Front matter needs no special offset handling: the fence lines stay in
the source buffer (the front-matter parser consumes them but offsets are
absolute), so block and inline spans below the fence point at the true
source lines. Inline spans do not render — `data-md-line` stays a
block-level annotation — and the JSON codec simply carries the extra
`span` fields (`omitempty`, wire-compatible).
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

v0.5 adds `mdv_asset`, exposing the embedded asset bundles and composed
per-theme highlight CSS — closing the fragment-host gap where
diagrams/math/highlighting were reachable only through full-page output.
The registry is append-only, mirroring the Kind-value policy.

v0.6 extracts the JSON boundary — request/option decoding, operation
dispatch, response encoding — out of `ffi/` and into `internal/boundary`,
so it's a shared, cgo-free package consumed by both the C ABI main
(`ffi/`) and the `js/wasm` main (`wasm/`) instead of living only in the
cgo layer. It also lets the `Resolver` callback cross both boundaries,
each in the idiom native to its host: over the C ABI as a function
pointer (`mdv_render_r`/`mdv_render_doc_r`) plus the `mdv_alloc`
library-heap ownership contract for the resolver's returned URL, so
freeing never crosses a CRT boundary on Windows; over the WASM boundary
as a plain JS function, since `js/wasm` already marshals JS values
without a C calling convention in between. Both crossings share the
same `kind` encoding — 0 (link), 1 (image), 2 (wiki-link) — ABI-frozen
like the `Kind` enum. See the "Resolver trust contract" safety bullet
below for what a resolver is and isn't responsible for; that contract is
unchanged by which boundary it crosses.

v0.7's mobile artifacts (`flutter/mdviewer`, `scripts/build-mobile.sh`)
consume this same ABI unchanged (nine symbols at the time) — static
(`c-archive`) on iOS, `c-shared` on Android, exactly the shape `ffi/`
already produces for desktop hosts. The Flutter plugin talks to it over
`dart:ffi`, with a per-call `NativeCallable.isolateLocal` bridging the
`Resolver` callback the same way the C ABI defines it; no new binding
layer or boundary crossing was needed on the Go side.

v0.10 grows the ABI to thirteen symbols with the four render-tree entry
points (`mdv_render_tree`, `mdv_render_tree_r`, `mdv_render_tree_doc`,
`mdv_render_tree_doc_r`), riding the same `internal/boundary` dispatch,
options decoding, resolver bridging, and panic containment as the
render family — see "The native render tree" below.

## The native render tree (`render/tree`, v0.10)

`render/tree` is the second renderer over the `document` model — the
layering rule's payoff made concrete — and the third form of rendered
output after full-page and fragment HTML. `tree.Build(doc, opts)` walks
the same `document.Document` the HTML renderer consumes and produces a
`*tree.Tree` whose `MarshalJSON` emits the strict version-1 wire schema
(`{"version":1, "blocks":[...], "footnotes":[...]}`): a layout-free
semantic tree for toolkit-native hosts (Flutter, SwiftUI, Compose, …)
that render platform widgets instead of loading HTML into a webview.

The design center is **resolved semantics**: everything policy-heavy
happens library-side, once, before the tree crosses any boundary — URL
resolution and scheme filtering (a blocked destination is `url:""` +
`blocked:true`), raw-HTML sanitizing through the same bluemonday policy
the HTML renderer uses, admonition title derivation, footnote
reference/definition pairing, wiki-link fallback, and the math/mermaid
off-fallbacks to code shapes. A host walks the tree and styles nodes;
it never re-implements policy.

Three properties keep the two renderers honest with each other:

- **Shared derivations, not copied ones.** The admonition-title,
  footnote-pairing, wiki-fallback, and URL-policy logic lives in shared
  helpers (`render/internal/derive`, the `resolve` package) that *both*
  renderers call — extracted, byte-identity-proven for HTML, rather than
  duplicated. Differential tests then pin text parity: the tree's plain
  text equals the HTML renderer's text content across the fixture corpus
  and the CommonMark suite.
- **One naming universe.** Tree kind names are exactly the document
  codec's wire names (`document.Kind.String()`), so a kind in the doc
  JSON and the same kind in the render tree spell identically.
- **Highlighting as data, from the same source.** Code blocks carry
  chroma token runs (the v0.9 tokenise seam, now exported as
  `htmlrender.TokenRuns`, with its own bounded cache) whose
  concatenated text equals the code exactly; the
  `highlight-light.json`/`highlight-dark.json` assets map the token
  types to colors, generated from the same chroma style objects the CSS
  uses — the CSS and the data cannot drift.

Block identity is a content hash — `hex(sha256(block source bytes))[:16]`
— so ids are stable across edits elsewhere in the document (the basis
for host-side diffing and itemized rebuilds). The corollary is
documented on every surface: byte-identical blocks *share* an id by
design, so hosts key by `(id, occurrenceIndex)`; and blocks with no
source at hand (spanless nodes, the whole document-JSON path) take a
deterministic positional fallback id.

The tree rides the existing boundary machinery unchanged:
`internal/boundary` grows RenderTree/RenderTreeDoc operations, the C ABI
four symbols (`mdv_render_tree*`, thirteen total), WASM
`renderTree`/`renderTreeDoc` (typed in `index.d.ts`), and the Flutter
plugin a strict typed model (`MdvTree`, sealed `MdvBlock`/`MdvInline`)
plus `MdvDocumentView` — the in-repo proof that the tree is actually
renderable as native widgets, with native math (`flutter_math_fork`),
token-run code with a native copy button, and a never-live default for
HTML nodes. Only the semantic options apply to tree operations; the
HTML-only fields are decoded and ignored (documented per surface), and
spans are always included. C and WASM tree output is verified
byte-identical for the same input.

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

v0.6 shipped the `js`/`wasm` build (`wasm/`, `scripts/build-wasm.sh`) over
the same shared JSON boundary the C-shared FFI established, now factored
out into `internal/boundary` and consumed by both cgo and wasm mains, with
an npm-ready ESM wrapper (`wasm/npm/`) so browser and Node hosts load it
like any other package. It also extends the `Resolver` callback across
both non-Go boundaries — the C ABI (`mdv_render_r`/`mdv_render_doc_r`) and
WASM (a plain JS function) — closing the gap the v0.4 FFI left open. See
the "FFI boundary" subsection above.

v0.7 shipped mobile — a Flutter FFI plugin (`flutter/mdviewer`) and the
release artifacts it consumes (`libmdviewer-<version>-ios.xcframework.zip`,
`libmdviewer-<version>-android.zip`, built by `scripts/build-mobile.sh`).
No new boundary was needed: the plugin binds the same ABI the
C-shared FFI already exposes (nine symbols at the time; thirteen since
v0.10), static on iOS and `c-shared` on Android, over
`dart:ffi` with `NativeCallable` bridging the `Resolver` callback. See the
"FFI boundary" subsection above and `flutter/mdviewer/README.md`.

v0.8 closed the host-integration gaps that surfaced while embedding the
library in real desktop and mobile hosts, ahead of any new rendering
featureset: two opt-in options riding the existing strict options JSON
across every surface — `extraCss` (host CSS appended after the page's
base styling, or after a `stylesheet` replacement) and `codeHeader`
(a language-label + copy-button header on code blocks, with inline copy
JS on full pages) — plus Flutter-side pre-resolve helpers
(`collectResolvables`/`resolverFromMap`) codifying the parse → prefetch →
sync-resolver pattern for async vaults, and the `flutter-v<ver>` tag
process for submodule consumers (CONTRIBUTING.md's Releasing section).

v0.9 shipped the native-render enabling train — the collisions a native
render-tree renderer would otherwise hit, cleared ahead of its design,
with no behavior change to existing output: the renderer-agnostic
`resolve` package (URL policy and wiki-link resolution out of
`render/html`, so a non-HTML renderer never imports the HTML one), the
chroma tokenise/format split in `render/html/highlight.go` (the seam a
native renderer needs for (text, style) runs) with a bounded per-block
highlight cache behind it, the theme palettes as version-1 JSON assets
(`theme-light.json`/`theme-dark.json` — native hosts stop parsing CSS to
recover colors), inline-level source spans on the document model
(emphasis, links, code spans, math, ... — what selection mapping needs),
`parser.Config` and heading-anchor control across every surface's
options JSON, and a Flutter plugin↔library version handshake — clearing
the way for the render tree itself.

v0.10 shipped the native render tree — the second renderer the layering
rule always promised: the `render/tree` package building the version-1
resolved semantic tree from the same `document.Document`, shared
derivations with the HTML renderer (differential-tested for text
parity), token runs + `highlight-*.json` color assets, four new C
symbols (thirteen total), WASM `renderTree`/`renderTreeDoc` with typed
d.ts, and the Flutter typed model + `MdvDocumentView` native widget
renderer with native math (`flutter_math_fork`) — validated by the
example app's Native page on both mobile platforms. See "The native
render tree" section above. Known v0.10 limits: footnote refs are
superscript-only (no jump-to-definition — needs scroll-controller work),
and mermaid renders as a pluggable placeholder.

v0.11.0 closed the first two of those v0.10 limits from the library
side (a cross-repo rendering-engine synchronization effort's Phase 1):
a footnote ref→definition linkage (`FootnoteRef.DefID` /
`Tree.FootnoteByIndex`) so a host can jump to a definition without a
scroll-controller-free heuristic, and `mermaid-bridge.js`, a first-cut
asset primitive (`mdvRenderMermaid`) for a host's own offscreen webview
to turn Mermaid source into real SVG — offscreen-SVG generation itself
stays a host-side responsibility, not a Go-side pipeline (see the
train's design note). Also fixed: CRLF/CR-terminated code fences no
longer decline native token runs. No wire schema change (tree stays
version 1).

Toward v1.0, what remains, roughly in order:

1. **Mobile native reader validation** — the MDViewer.Mobile app's
   native reader consumes the v0.11.0 primitives above; expected to keep
   feeding back smaller tree/widget gaps as it does (e.g. Mobile's own
   scroll-controller work to jump to a footnote definition on tap, not
   just its containing footnotes section).
2. **Mermaid rendering polish** — fold `mermaid-bridge.js` into the
   Flutter plugin's default `diagram` builder if the Mobile prototype
   earns it; still no headless-rendering dependency inside the Go
   library itself (see "The native render tree" and the v0.11.0 note
   above for why).
3. **The 1.0-freeze discussion** — the `document` model, options JSON,
   C ABI, and now the tree schema have all been shaped by real hosts;
   deciding what freezes (and what stays explicitly unfrozen) is its own
   train.
4. **Incremental rendering**, if profiling on real host workloads shows
   full re-render (even with parse-once/`RenderDoc`) is the bottleneck —
   not committed to; only worth doing if the numbers demand it.

Every step above consumes (and, where relevant, extends) the same
`document` model both renderers use today; none requires rearchitecting
the parsing layer.
