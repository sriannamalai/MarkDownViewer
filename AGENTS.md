# AGENTS.md — MarkDownViewer (core library)

This file is persistent project memory for agents. It also cross-references
the two downstream apps built on this library so work across all three repos
can be coordinated consistently. Update this file whenever architecture,
status, or roadmap materially changes.

## What this project is
A Go library (`github.com/sriannamalai/markdownviewer`) that turns Markdown
into self-contained, sanitized HTML — and, since v0.10, into a native
"render tree" (layout-free semantic JSON) for widget-native hosts. One
rendering core, five consumption surfaces: Go package, CLI (`cmd/mdview`),
C ABI (`libmdviewer`), WASM, and a Flutter plugin (`flutter/mdviewer`).

Current version: **v0.11.0** (pre-1.0, API not frozen). 158 commits.
Latest tags: v0.10.0, v0.11.0 (+ `flutter-v0.10.0`, `flutter-v0.10.1` for the
plugin; `flutter-v0.11.0` pending — see the Engine version sync checklist
below). See `CHANGELOG.md` for full release notes and `docs/Design.md` for
the authoritative architecture doc — **read that file first** for anything
non-trivial; this AGENTS.md summarizes it plus adds cross-repo context.

## The sibling repos (the bigger picture)
This library is the shared rendering engine for two downstream apps that
the same author (Sri) is building. When working on any one of the three,
it's useful to know what the others are doing:

- **`~/Developer/OpenSource/MDViewer.Desktop`** — a Tauri 2 (Rust +
  TypeScript) desktop app for macOS/Windows/Linux. Consumes this library
  via the **C ABI** (`libmdviewer`, vendored per-platform, currently pinned
  at **v0.8.1**). Renders documents in a sandboxed webview
  (`<iframe sandbox>`) fed HTML from the FFI.
- **`~/Developer/OpenSource/MDViewer.Mobile`** — a Flutter app for
  iOS/Android. Consumes this library as a **git submodule**
  (`vendor/markdownviewer`, pinned to tag `flutter-v0.10.1`) via the
  `flutter/mdviewer` plugin in this repo. It has a **dual-engine reader**:
  native (`MdvDocumentView`/`MdvDocumentAdapter`, the render tree) as
  default, with a Webview fallback for Mermaid diagrams and a few
  native-engine gaps.

**Finalized architectural specialization** (Cross-Repo Rendering Engine
Synchronization Plan, 2026-08): the two apps are deliberately kept
specialized, not converged onto one rendering model — this library does
not push either app toward adopting the other's approach.
- **Desktop is the HTML/webview flagship.** It does not adopt the v0.10
  native render tree, on purpose: Tauri's entire UI already runs inside a
  system webview, so there are no native widgets to gain by switching, and
  Desktop's HTML pipeline is already a complete showcase (live
  interactive `mermaid.js` + KaTeX). "Desktop adopts the render tree" is
  a closed question, not a backlog item.
- **Mobile is the native render-tree + dual-engine flagship.** Its
  Flutter widgets paint directly via Skia/Impeller with virtualized
  scrolling, genuinely lighter than embedding a full platform WebView per
  document; the Webview fallback stays as Mobile's deliberate
  demonstration of the library's third capability (engine switching).

Both apps share one design language, defined once and duplicated into each
app's `design/TOKENS.md` (byte-identical color/typography/spacing tokens —
if you change one, check whether the other needs the same edit). The Mobile
app's native reader is the closest thing to a live testbed for this
library's `render/tree` package — bugs found there often mean fixing this
repo, then re-pinning Mobile to a new `flutter-v<ver>` tag.

### Engine version sync checklist
Run this on every core release so all three repos' pinned versions and
rendering paths stay genuinely aligned (not just superficially matching
version numbers):
1. Bump `CHANGELOG.md` for the new version and land it on `main`.
2. Tag `v<ver>` (and `flutter-v<ver>` if `flutter/mdviewer` changed) per
   `CONTRIBUTING.md`'s Releasing section.
3. Desktop: re-run `scripts/fetch-libmdviewer.sh`, verify
   `vendor/checksums.txt`, bump the pinned version noted in its README.
4. Mobile: bump the `vendor/markdownviewer` submodule to the new
   `flutter-v<ver>` tag, re-test dual-engine parity (native vs. Webview
   render the same document consistently).
5. Update all three repos' `AGENTS.md` "Finished so far" sections with
   what actually shipped.
Current pinned versions/paths (see each repo's own `AGENTS.md` for detail):
Desktop → C ABI v0.8.1, HTML/webview only. Mobile → submodule
`flutter-v0.10.1` (native `v0.10.0` binaries), native render tree default
with Webview fallback.

**Practical implication:** a change here (especially to the C ABI, the
options JSON, or `render/tree`) is not "done" from the ecosystem's
perspective until the relevant downstream repo re-vendors it:
- Desktop: bump `vendor/checksums.txt` / re-run
  `scripts/fetch-libmdviewer.sh`, update the pinned version in its README.
- Mobile: bump the `vendor/markdownviewer` submodule to the new
  `flutter-v<ver>` tag (see this repo's `CONTRIBUTING.md` Releasing section
  for how that tag gets cut).

## Architecture (see `docs/Design.md` for full detail)
Two-stage pipeline: `parser` (wraps goldmark) → `document.Document` (typed,
renderer-agnostic AST) → either `render/html` (HTML) or `render/tree`
(native render tree). **Layering rule:** `document` has zero non-stdlib
deps; nothing outside `parser` may import goldmark types. This is what lets
two renderers (and someday more) share one AST without duplicating parsing.

Key packages: `document` (AST + versioned JSON codec + pinned `Kind` enum,
0–31, append-only), `parser` (Markdown → AST, `parser.Config` toggles per
extension), `render/html`, `render/tree` (v0.10, shares derivation logic
with `render/html` via `render/internal/derive` + `resolve`), `resolve`
(URL policy / `Resolver` contract, ABI-frozen `ResolveKind` ints), `theme`,
`assets` (embedded offline KaTeX/mermaid), `ffi/` (C ABI, 13 symbols),
`wasm/`, `flutter/mdviewer/` (the plugin consumed by MDViewer.Mobile).

Security model: sanitize-by-default (bluemonday), URL scheme allowlist
(http/https/mailto/tel), offline/self-contained output, documented
`Resolver` trust boundary. See `SECURITY.md`.

## Finished so far (through v0.10.1)
- v0.1–v0.3: CommonMark+GFM parser, HTML renderer, CLI, theming, source
  maps, max-width option.
- v0.4–v0.6: C-shared FFI (`libmdviewer`), WASM build, `Resolver` callback
  crossing both boundaries.
- v0.7: Flutter FFI plugin + iOS/Android release artifacts (mobile as a
  third embedding surface).
- v0.8: host-integration polish (`extraCss`, `codeHeader`, Flutter
  pre-resolve helpers, `flutter-v<ver>` tag process).
- v0.9: the "native-render enabling train" — `resolve` package extracted,
  chroma tokenise/format split + cache, theme palettes as JSON, inline
  source spans, parser config across every surface, plugin↔library version
  handshake.
- v0.10 / v0.10.1: **the native render tree** (`render/tree`) — the second
  renderer, differential-tested for text parity against HTML; token runs +
  highlight color JSON assets; 4 new C symbols (13 total); WASM
  `renderTree`/`renderTreeDoc`; Flutter typed model + `MdvDocumentView`
  native widget renderer with native math; `MdvDocumentAdapter` (v0.10.1)
  exposing list assembly for host-owned scrollables + line-mapping helpers.
- v0.11.0: the sync-plan Phase 1 gap-train — CRLF code-fence highlighting
  fixed (was fail-closed); footnote jump-to-definition primitive
  (`FootnoteRef.DefID` / `Tree.FootnoteByIndex`); `mermaid-bridge.js` asset
  (`mdvRenderMermaid`) for offscreen-webview hosts; `onFootnoteRefTap` on
  the Flutter plugin's render scope/adapter/view; windows-arm64 added to
  the release build matrix (six desktop targets). Finalized and recorded
  the cross-repo architectural specialization + Engine version sync
  checklist (below).

## Known limitations / open items (roadmap toward v1.0)
From `docs/Design.md`'s Roadmap section, in rough order:
1. **Mobile native-reader validation** — feed real-world gaps back from
   MDViewer.Mobile's native reader. The library-side primitive (footnote
   ref→definition linkage: `FootnoteRef.DefID` / `Tree.FootnoteByIndex`,
   see `render/tree`) has shipped; Mobile's own scroll-controller design
   to actually jump on tap is still tracked there (Phase 2).
2. **Mermaid offscreen-SVG fast-follow** — decided to stay a host-side
   (webview) responsibility, not a Go-side rendering pipeline; see
   `.superpowers/specs/2026-08-28 Mermaid Offscreen-SVG Direction.md`.
   The library now ships a first-cut primitive, the `mermaid-bridge.js`
   asset (`mdvRenderMermaid(id, source, theme)`), for a host's offscreen
   webview to call; wiring it into Mobile's `diagram` builder is Phase 2.
3. **The 1.0-freeze discussion** — decide what in `document`, the options
   JSON, the C ABI, and the tree schema freezes vs. stays explicitly open.
4. **Incremental rendering** — only if profiling on real host workloads
   demands it; not committed to.

Also documented, not roadmap items but known caveats: resource exhaustion
on deeply-nested lists is a documented, only partially mitigated gap
(`Context` variants bound latency, not CPU). Previously listed here and
now fixed: CRLF code fences used to decline native token runs
(fail-closed) — `render/html`'s `TokenRuns` now re-expands chroma's
LF-normalized tokenization back to the source's exact CRLF/CR bytes, so
`render/tree` code blocks highlight normally.

## Build & test
```bash
go build ./...
go test ./...              # full suite
go test -race ./...        # required before PR
go vet ./...
gofmt -l .                 # must print nothing
go test -coverprofile=/tmp/cover.out ./... && ./scripts/check-coverage.sh /tmp/cover.out 75
```
Golden-file tests live under `testdata/`; regenerate with `go test ./... -update`
and hand-review every diff — never regenerate to paper over an unintended
change. FFI: `./scripts/build-ffi.sh` then the C harness (see
`CONTRIBUTING.md`). Flutter plugin: `cd flutter/mdviewer && flutter test`.
Requires Go 1.26+. Conventional Commits style; DCO sign-off (`git commit -s`)
required on all commits (see `CONTRIBUTING.md`).

## Conventions worth knowing before editing
- `Kind` enum values and C ABI symbols are **append-only** — never renumber
  or remove.
- The options JSON (Go functional options mirrored as strict JSON across
  C/WASM/Flutter) rejects unknown fields and wrong-case keys by design —
  don't relax that.
- `render/tree` and `render/html` must stay differentially text-parity
  tested; don't let them drift by hand-patching one without the other's
  shared `render/internal/derive`/`resolve` helpers.
- Two independently-maintained shadow implementations of upstream goldmark
  internals exist (`parser/parser.go`'s front-matter swallow detection,
  `parser/tbreak.go`'s thematic-break predicate) — any goldmark version
  bump requires re-diffing both against upstream (see `CONTRIBUTING.md`).
