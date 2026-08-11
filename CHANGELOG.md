# Changelog

All notable changes to this project are documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/).
This project is pre-1.0 (see `docs/Design.md`'s Status section); until
v1.0.0, minor version bumps may include breaking changes to the `document`
model and renderer options.

## [0.8.1] - 2026-08-11

Security patch and release-pipeline hardening, from the post-v0.8.0
end-to-end reanalysis.

### Security

- **Admonition titles are now HTML-escaped.** The admonition *title* was
  emitted unescaped (the class attribute was already escaped). Unreachable
  from markdown input — the parser constrains admonition variants to a
  fixed set — but `RenderDoc`, `mdv_render_doc`, and the WASM/Flutter
  `renderDoc` accept caller-supplied document JSON, where a hostile
  `variant` could inject script through the default sanitized render path.
  Hosts that render document JSON from untrusted or externally-stored
  sources should upgrade. Regression tests cover the renderer and the
  boundary attack path, and a new fuzz target (`FuzzRenderDocJSON`)
  continuously exercises hostile document JSON in fragment mode.
- `WithThemeOverrides` documentation now states plainly that override
  *values* are host-trusted CSS: only `</style` is stripped, and a value
  can close the `:root{}` block and inject arbitrary rules. Hosts must not
  echo untrusted data into override values. (Documentation only; behavior
  unchanged.)

### Changed

- **Release artifacts are now smoke-tested before upload**: the C harness
  runs against the built library natively on darwin-arm64, linux-amd64,
  and linux-arm64, and under Rosetta for the cross-compiled darwin-amd64;
  the Node harness runs against the built WASM package. windows-amd64 is
  harness-covered per-commit in CI; iOS/Android remain build-only (no
  runner can execute them).
- **Every release now ships a `SHA256SUMS` asset** covering all eight
  artifact zips (previously only the two mobile zips were pinned, in
  `flutter/mdviewer/tool/checksums.txt`, which continues unchanged).
- CI hardening: workflow token restricted to `contents: read`, the
  third-party Flutter action is pinned by commit SHA, and a new browser
  job loads the WASM playground in headless Chromium (Playwright),
  asserting mermaid, KaTeX, and resolver rendering actually execute —
  previously the browser path had no automated coverage.

## [0.8.0] - 2026-08-11

Closes the host-integration gaps ledgered while embedding the library in
real desktop and mobile hosts: two new opt-in rendering options that ride
the existing strict options JSON across every surface (Go facade, C ABI,
WASM, Flutter plugin), Dart-side pre-resolve helpers for async vaults, and
a documented `flutter-v<ver>` tag process for submodule consumers. Both
new options default off; existing output is unchanged unless opted in
(one byte-level exception noted under Changed).

### Added

- **`extraCss` option** — `markdownviewer.WithExtraCSS(css)`, boundary
  options JSON `"extraCss"` (string), WASM `options.extraCss`, Flutter
  `MdvOptions.extraCss`. Host CSS appended at the end of the full page's
  `<style>` — after the base+theme stylesheets by default, or after the
  `stylesheet` replacement when one is set (`stylesheet`'s replace
  semantics are unchanged). Sanitized through the same CSS sanitizer as
  `stylesheet`. Full-page scope only, like `stylesheet`; no effect on
  fragment output. Lets hosts layer text-scale overrides or `@font-face`
  rules (data: URI fonts) without fetching and re-concatenating
  `base.css`.
- **`codeHeader` option** — `markdownviewer.WithCodeHeader()`, boundary
  options JSON `"codeHeader"` (bool, default false), WASM
  `options.codeHeader`, Flutter `MdvOptions.codeHeader`. Wraps every code
  block (both the chroma-highlighted and plain emit paths) in
  library-authored header markup: `<div class="md-code">` containing a
  `md-code-header` row with a `md-code-lang` language label (the fence
  language, or `code` when unlabeled) and a `md-code-copy` button, styled
  by `theme/base.css` via the existing `--md-*` variables. Live mermaid
  and math blocks are not wrapped; their plain-code fallbacks (engines
  disabled) are. `data-md-line` stays on the element it is on today. Full
  pages additionally embed a small inline clipboard script (copy →
  transient "Copied"); fragment hosts get the markup and classes and wire
  their own click handler.
- **Flutter pre-resolve helpers** (`flutter/mdviewer`):
  `collectResolvables(doc)` walks a parsed document and returns every
  distinct resolvable target (link/image/wiki-link, the ABI-frozen 0/1/2
  kinds), and `resolverFromMap(resolved, {kindFilter})` builds a sync
  `MdvResolver` that answers mapped targets verbatim and declines the
  rest. Together they codify the async-vault pattern (parse once →
  collect targets → prefetch async → render with the map-backed sync
  resolver) now documented as a recipe in the plugin README.
- **`flutter-v<ver>` tag process.** After a release's mobile-artifact
  checksums are appended to `flutter/mdviewer/tool/checksums.txt` and the
  plugin pubspec is bumped, that commit is tagged `flutter-v<ver>` and
  pushed (see CONTRIBUTING.md's Releasing section) — submodule consumers
  pin `flutter-v<ver>`, never a raw SHA. Applied retroactively as
  `flutter-v0.7.0`.

### Changed

- **Boundary options JSON keys are now matched exact-case.**
  `encoding/json` matches field names case-insensitively even with
  `DisallowUnknownFields`, so wrong-case keys (e.g. `"extraCSS"`,
  `"Theme"`) were silently case-folded into the canonical fields since
  the boundary's introduction in v0.4. Decoding now rejects them with an
  unknown-field error, matching the documented contract, which was always
  "unknown fields are errors". Consumers emitting the documented
  lowerCamel keys (all known bindings do) are unaffected.
- **Full-page output always includes the (inert) `.md-code` style block**
  in `base.css`, even when `codeHeader` is off — page bytes differ from
  v0.7.0 while the emitted markup is unchanged. Byte-for-byte page
  comparisons against v0.7.0 output will differ; markup-level comparisons
  will not.
- `flutter/mdviewer/ios/mdviewer.podspec` and the plugin README document
  the known iOS first-build-after-`pod install` ordering quirk around
  `-force_load` ("Build input file cannot be found"; a retry succeeds).
  Docs only — no build-system change.

## [0.7.0] - 2026-08-10

Adds mobile as a third embedding surface alongside the C ABI and WASM: a
Flutter FFI plugin binding the same nine-symbol ABI, plus the release
artifacts and CI it needs. Nothing changed for existing Go/C/WASM
consumers.

### Added

- **Mobile release artifacts** (`scripts/build-mobile.sh`): `libmdviewer`
  built for iOS (`c-archive`, arm64 device + arm64 simulator, merged into
  `libmdviewer.xcframework` via `xcodebuild -create-xcframework`) and
  Android (`c-shared`, arm64-v8a + x86_64 `.so`s under `jniLibs/`). Every
  release now also ships `libmdviewer-<version>-ios.xcframework.zip` and
  `libmdviewer-<version>-android.zip` (`.github/workflows/release-ffi.yml`),
  each bundling `LICENSE` and `ffi/README.md` alongside the binary, same
  as the desktop C-shared zips.
- **Flutter FFI plugin** (`flutter/mdviewer`): a typed `dart:ffi` binding
  over the same nine-symbol C ABI — `render`/`parse`/`renderDoc`/`asset`,
  a strict `MdvOptions` mirroring the boundary's options JSON, and a
  `Resolver` callback bridged via a per-call
  `ffi.NativeCallable.isolateLocal`, matching the C ABI's `mdv_render_r`/
  `mdv_render_doc_r`/`mdv_alloc` contract (trusted-verbatim resolution,
  decline via `null`, a thrown resolver declines the remainder and
  rethrows after the call). Platform loading: the bundled `.so` on
  Android, statically linked via `DynamicLibrary.process()` on iOS (the
  podspec's `-force_load` plus an `EXCLUDED_ARCHS` workaround for the
  arm64-only simulator slice), `MDVIEWER_LIBRARY_PATH` or a `dist/ffi/`
  walk-up for host development on macOS/Linux. `tool/build_binaries.sh`
  (build from source) and `tool/fetch_binaries.sh` (checksum-verified
  release download) populate the platform binaries, which are never
  committed. Not yet on pub.dev — `publish_to: none`, path-dependency
  consumption only; publishing remains a separately gated step. 26 tests
  against the host dylib.
- **Example app** (`flutter/mdviewer/example`): a `webview_flutter` host
  rendering mermaid, inline/block KaTeX math, a syntax-highlighted code
  fence, a resolver-rewritten image, and a wiki-link, with a light/dark
  theme toggle exercising `renderDoc`'s parse-once/render-many path.
  Manually verified on iOS and Android simulators/emulators, light and
  dark.
- **CI:** a `mobile` job (`macos-14`) builds both mobile targets via
  `scripts/build-mobile.sh all`, and a `flutter` job (`macos-14`,
  `subosito/flutter-action@v2`) builds the host FFI library, then runs
  `flutter pub get`/`dart format --set-exit-if-changed`/`flutter analyze`/
  `flutter test` against `flutter/mdviewer`.

## [0.6.1] - 2026-08-10

### Fixed

- **FFI:** `cResolver` now guards the `out_url_len` `size_t → int`
  conversion (contract-violation error instead of a hypothetical 32-bit
  truncation) and frees the host's `*out_url` on every return-`1` path,
  including one with an invalid (oversized) length — ownership transferred,
  so the library owns and frees the buffer even though the render then
  fails. On any other return code, no ownership transfer happened: the
  library no longer touches `*out_url` at all. Freeing an unowned pointer
  a misbehaving host left there could be an invalid free on
  static/stack/foreign memory — strictly worse than the leak, which
  belongs to the host that violated the contract.
- **WASM wrapper:** option values that are `undefined`, functions, or
  symbols now throw a `TypeError` instead of being silently dropped by
  `JSON.stringify` — checked both at the top level and one level into
  object-valued options (e.g. `themeOverrides`); a failed `loadMdviewer()`
  is no longer cached, so a later call can retry after a corrupt-binary or
  network failure.

### Changed

- `ffi/README.md` documents the narrowed ownership rule (freed only on
  return-`1` paths, never touched otherwise), the exact-length requirement
  on `out_url_len`, and the header's two `-Wunused-function` warnings
  under `-Wall`; the packaged wasm README documents load-retry semantics
  and the guarded Node-only dynamic import. Both harnesses now cover the
  wiki-link resolve path, the `out_url_len` MaxInt guard on a real
  allocation, and the no-touch violation path (ASan-verified: the host
  frees its own buffer with no double-free); the wasm harness covers
  corrupt-binary rejection + retry in a subprocess (asserting rejection
  and successful retry rather than exact message wording) plus the new
  symbol/nested-object strictness checks.

## [0.6.0] - 2026-08-10

Adds a second non-Go embedding surface (WebAssembly) alongside the C ABI,
and closes the FFI's biggest remaining gap: the `Resolver` callback now
crosses both boundaries. Nothing changed for pure-Go consumers.

### Added

- **`mdv_render_r`, `mdv_render_doc_r`, `mdv_alloc` C symbols** (three
  new exported symbols, six → nine; append-only ABI growth):
  `mdv_render_r`/`mdv_render_doc_r` are the plain-render/render-doc
  calls plus a host resolver callback, letting a `Resolver` cross the C
  ABI as a function pointer for the first time. `kind` travels as an
  ABI-frozen int (0 = link, 1 = image, 2 = wiki-link), matching the
  `ResolveKind` values. `mdv_alloc(n)` allocates on the library's heap so
  a resolver's returned URL can be freed with `mdv_free` without crossing
  a CRT boundary on Windows. Same memory contract, panic containment, and
  thread-safety as every other symbol — the resolver callback itself runs
  synchronously on the calling thread, one call at a time per render.
- **WASM build** (`GOOS=js GOARCH=wasm`, `wasm/`, built with
  `scripts/build-wasm.sh`): the same rendering surface as the C ABI,
  exposed as a plain ESM module (`wasm/npm/`) with no bundler or native
  dependency required. The `Resolver` crosses this boundary as an
  ordinary JS function rather than a marshalled callback. Every release
  now also ships `libmdviewer-<version>-wasm.zip`, an npm-ready package
  (`.github/workflows/release-ffi.yml`).
- **Node harness** (`examples/node/harness.mjs`), CI-gated against the
  freshly built wasm package — parity checks against the C harness plus
  resolver-specific coverage (all three kinds, declined resolution
  falling back to default handling, a throwing resolver failing the
  render with the host error).
- **Browser playground example** (`examples/web/`) demonstrating the
  ESM module loaded directly in a page, no build step.

### Changed

- **Internal:** the JSON boundary (request/option decoding, operation
  dispatch, response encoding) that used to live entirely inside `ffi/`
  is now `internal/boundary`, a shared cgo-free package consumed by both
  the C ABI main and the wasm main. No public Go API change.
- `ffi/README.md` (packaged into release artifacts) documents the three
  new symbols, the resolver callback signature, and the `mdv_alloc`
  ownership contract.

## [0.5.0] - 2026-08-10

Completes the fragment-host story surfaced by real-world FFI consumption:
diagrams, math, and syntax highlighting are now deliverable to hosts that
render fragments into their own pages, over both the Go API and the C ABI.
Nothing changed for full-page rendering.

### Added

- **`htmlrender.HighlightCSS(t theme.Theme) (string, error)`.** The chroma
  syntax-highlighting stylesheet for that theme's mode — the same CSS a
  full page embeds. Until now this was internal, so fragment output
  emitted class-annotated code spans that no public API could style:
  highlighting silently didn't work for fragment hosts. Go hosts combine
  it with `theme.BaseCSS` and the theme's own CSS.
- **`mdv_asset` C symbol** (sixth exported symbol; append-only ABI
  growth): `int mdv_asset(const char* name, char** out, size_t* out_len,
  char** out_err)`. Returns embedded static assets by registry name so
  FFI fragment hosts can enable diagrams/math/highlighting without
  vendoring anything. Registry (append-only, case-sensitive):
  `mermaid.js`, `katex.js`, `katex.css` (all fonts inlined as data:
  URIs — self-contained), `base.css`, and the composed `theme-light.css`
  / `theme-dark.css` (theme tokens **plus** that mode's highlight CSS,
  exactly what a full page embeds per mode, so applying the one file
  yields working syntax highlighting). Unknown, empty, or NULL names
  error with the message listing the valid names. Same memory contract
  as every other call (`mdv_free`), same panic containment,
  thread-safe.
- C harness: five new checks covering asset retrieval, content markers,
  the composed theme CSS, and the error paths (22 checks total).

### Changed

- `ffi/README.md` (packaged into release artifacts) documents the asset
  registry, the `base.css` pairing rule for the `--md-*` variables, and
  that the generated header is not const-qualified (cast as needed —
  the library never writes through input pointers).

## [0.4.0] - 2026-08-09

First non-Go embedding surface: a C-shared library any language with a C
FFI can load. Nothing changed for pure-Go consumers.

### Added

- **`libmdviewer` C-shared library** (`ffi/`, built with
  `-buildmode=c-shared` via `scripts/build-ffi.sh`). Five thread-safe
  exported symbols: `mdv_render` (Markdown → HTML), `mdv_parse` (Markdown
  → version-1 document-AST JSON), `mdv_render_doc` (AST JSON → HTML,
  enabling parse-once/render-many without cross-boundary object handles),
  `mdv_free`, and `mdv_version`. Out-parameter/int-status calling
  convention; every returned buffer is C-malloc'd UTF-8 with an uncounted
  trailing NUL and must be freed with `mdv_free`. Options cross the
  boundary as a strict, versioned JSON object mirroring the Go facade's
  functional options — unknown fields, type mismatches, trailing content,
  and unsupported versions are errors, so binding typos surface
  immediately. A Go panic anywhere in an operation is contained at the
  boundary and reported as the documented non-zero return plus a
  `panic:`-prefixed message instead of unwinding into (and killing) the
  host process; true C-allocation failure still aborts, like the Go
  runtime itself on OOM. The `Resolver` callback does not cross the FFI.
- **Prebuilt release artifacts.** Publishing a GitHub release now builds
  and attaches `libmdviewer-<version>-<os>-<arch>.zip` (shared library +
  header + LICENSE + an API-reference README) for macOS arm64/x86_64,
  Linux amd64/arm64, and Windows amd64
  (`.github/workflows/release-ffi.yml`).
- **C harness** (`examples/c/`), compiled and run against the freshly
  built library in CI on Linux, macOS, and Windows — the ABI's test
  suite, exercising every symbol including error paths, the NULL-input
  contract, and byte-for-byte render vs parse→render_doc agreement.
- **`dart:ffi` example** (`examples/dart/`) demonstrating the pattern a
  Flutter host uses; verified locally against a locally built library.
- Docs: "Embedding from other languages" in README.md, an FFI-boundary
  section in `docs/Design.md`, and FFI build/test instructions in
  CONTRIBUTING.md.

## [0.3.0] - 2026-08-09

### Added

- **`WithMaxWidth(width string) Option` / `mdview -width STRING`.** Opt in to
  a constrained content width (any CSS length, e.g. `"860px"`, `"70ch"`) via
  the `--md-max-width` CSS custom property. Values are validated
  defensively: one containing `;` or `}` — either of which could break out
  of the CSS declaration it's emitted into — is rejected and the option
  no-ops rather than emitting a defanged value.
- **`mdview` no-args help.** Running `mdview` with no file argument and no
  piped input (stdin is an interactive terminal) now prints name/version,
  a one-line description, the flag defaults, and a few example invocations,
  then exits 0 — instead of blocking forever on a read that would never
  complete.

### Changed

- **Default page layout is now fluid.** `theme/base.css` no longer sets a
  fixed `max-width: 860px`; the default is `max-width: var(--md-max-width,
  none)`, i.e. no width constraint unless `WithMaxWidth`/`-width` is used.
  **Behavior change:** hosts relying on the old fixed-860px default for
  layout should pass `WithMaxWidth("860px")` (or `-width 860px`) to keep it.
- Task-list `<li>`s get `class="task-list-item"` (tight and loose), and
  their `<ul>` gets `class="contains-task-list"` when any child is a task;
  `theme/base.css` uses these to drop the list bullet and indent in favor
  of the checkbox's own alignment, matching GitHub's rendering.

### Fixed

- **Dark-mode syntax highlighting had invisible text.** In `auto` theme
  mode, a chroma token class that the light style ("github") sets a color
  for but the dark style ("github-dark") leaves unstyled (e.g. `.nx`, a
  plain identifier) kept its unconditional light-mode color even when the
  OS preferred dark, since the dark stylesheet — layered in a
  `prefers-color-scheme` media query — never emitted a competing rule for
  that class. The result was near-black text on a dark background. The
  dark chroma block now carries an explicit `color:inherit` for every class
  the light block styles that it doesn't, so it always wins the cascade tie
  and falls back to a readable color instead.
- **Task-list checkboxes rendered as bullets with text wrapping below
  them.** A loose task item rendered its checkbox as a sibling before the
  paragraph (`<li><input.../> <p>text</p>`), which pushed the text onto
  its own line. The checkbox now nests inside the first paragraph
  (`<li><p><input.../> text</p>...`), matching cmark-gfm's shape; see also
  the `task-list-item`/`contains-task-list` classes under Changed.
- **Mermaid diagrams were unreadable on dark backgrounds.** `auto` theme
  mode always initialized mermaid with its light `default` theme
  regardless of the OS's actual color scheme; it now picks mermaid's theme
  in the browser at load time via
  `window.matchMedia('(prefers-color-scheme: dark)')`, matching the
  condition the page's own CSS already uses. Separately, mermaid's
  built-in themes paint an opaque SVG canvas background that doesn't track
  `--md-bg`; `theme/base.css` now forces it transparent so the page
  background shows through.
- **`mdview` with no arguments hung waiting on stdin.** See the no-args
  help entry under Added.

## [0.2.0] - 2026-08-08

### Added

- **Block-level source spans.** `document.Span` locates block nodes in the
  original Markdown source (1-based `StartLine`/`EndLine`, 0-based
  half-open `StartOffset`/`EndOffset`). Leaf blocks get an exact span
  including their markers; container blocks get the union of their
  children's spans; `ThematicBreak` has no `Lines()` data to derive a span
  from and so is always zero-span. Offsets are relative to the source after
  BOM-stripping and invalid-UTF-8 sanitization.
- **Opt-in HTML source map.** `htmlrender.Options.SourceMap` /
  `markdownviewer.WithSourceMap()` annotate top-level block elements (and
  footnote `<li>`s) with `data-md-line="<n>"` for editor↔preview scroll
  sync. Raw HTML blocks and chroma-highlighted code blocks are skipped
  (unowned markup).
- **Versioned JSON codec.** `document.MarshalJSON`/`UnmarshalJSON` round-trip
  a `*document.Document` through a stable, version-1 wire format: string
  `Kind` names, lowerCamel fields, zero spans omitted. `Kind` values are now
  pinned (0-31) and append-only, with `Kind.String()`/`document.KindFromString`
  providing the stable wire-name mapping.
- **`RenderDoc`/`RenderDocTo`, `ParseWith`, `WithParserConfig`.** Parse once
  and render many times (e.g. theme switching) without re-parsing via
  `RenderDoc`/`RenderDocTo`; parse with an explicit `parser.Config` via
  `ParseWith`; and select that `Config` for `Render`/`RenderTo` via the new
  `WithParserConfig` option.
- **Public `assets` package.** The embedded mermaid.js and KaTeX JS/CSS
  bundles used for full-page rendering are now exported
  (`assets.MermaidJS`, `assets.KatexJS`, `assets.KatexCSS`) so fragment
  hosts can activate offline diagram/math support without vendoring their
  own copies.
- **Context-aware parse/render variants.** `ParseContext`, `RenderContext`,
  and `RenderDocContext` honor a `context.Context` deadline, returning
  `ctx.Err()` promptly on expiry. The abandoned goroutine keeps running to
  completion (the underlying Markdown engine has no cancellation hooks), so
  these bound caller-observed latency, not CPU spend — see the root package
  godoc's Concurrency section for the full abandonment contract.

### Fixed

- `parser`: a document whose first line is `---` and that never has a
  closing `---` was silently discarded (`Parse` returned an empty
  document) — go.abhg.dev/goldmark/frontmatter's block parser claimed the
  entire input as an unterminated front-matter block. `ParseWith` now
  detects the swallow (no front-matter data extracted, zero children, from
  non-blank source) and reparses with front matter disabled, so the
  content is recovered as CommonMark (`---` alone is a thematic break;
  `---\nbody` is a thematic break plus a paragraph).

### Changed

- Toolchain: requires Go 1.26 (up from 1.25).
- Dependencies updated to latest: `github.com/BurntSushi/toml` v1.5.0 →
  v1.6.0, `github.com/dlclark/regexp2/v2` v2.2.1 → v2.6.0, `golang.org/x/net`
  v0.26.0 → v0.57.0. `goldmark`, `goldmark-emoji`, `frontmatter`, `wikilink`,
  `chroma/v2`, and `bluemonday` were already pinned at their latest published
  versions, so those did not change.

## [0.1.1] - 2026-08-08

Test-infrastructure follow-up to 0.1.0; no library code changes.

### Fixed

- `.gitattributes` now forces LF checkouts so the byte-exact golden-file
  tests pass on Windows checkouts with `core.autocrlf=true`.
- The deeply-nested-list hang-detector test uses a smaller depth and a much
  wider watchdog so it cannot flake on slow CI runners.

## [0.1.0] - 2026-08-08

Initial release.

### Added

- **Parser** (`parser`): Markdown → `document.Document`, wrapping goldmark
  with CommonMark plus GFM tables, strikethrough, task lists, autolinks,
  footnotes, definition lists, YAML/TOML/JSON front matter, `:emoji:`
  shortcodes, `[[wiki-links]]`, inline/display TeX math, and GitHub-style
  admonitions (`> [!NOTE]`). Every extension beyond plain CommonMark is
  independently toggleable via `parser.Config`.
- **Document model** (`document`): a typed, renderer-agnostic AST —
  `Node`, concrete block/inline node types, `Walk`, `PlainText`, `Dump`.
  Depends on nothing outside the standard library.
- **HTML renderer** (`render/html`): full self-contained page or body-only
  fragment output; sanitized by default (bluemonday UGC policy); URL
  scheme allowlist (`http`, `https`, `mailto`, `tel`); chroma syntax
  highlighting for 250+ languages; offline mermaid diagram and KaTeX math
  rendering with assets embedded at build time (no CDN calls); pluggable
  `Resolver` hook for rewriting link/image/wiki-link targets; theming via
  CSS custom-property overrides or a full stylesheet swap.
- **Public facade** (root package `markdownviewer`): `Parse`, `Render`,
  `RenderTo`, and functional options (`WithTheme`, `Fragment`,
  `AllowRawHTML`, `DisableMermaid`, `DisableMath`, `DisableHighlighting`,
  `WithResolver`, `WithThemeOverrides`, `WithStylesheet`).
- **CLI** (`cmd/mdview`): renders a file or stdin to HTML with flags
  mirroring the facade options (`-o`, `-theme`, `-fragment`, `-unsafe`,
  `-no-mermaid`, `-no-math`, `-no-highlight`).
- **Security model**: sanitize-by-default raw HTML handling, URL scheme
  allowlist with control-character scheme-splicing protection, a
  documented `Resolver` trust contract, and offline/self-contained output
  with no third-party request surface. See `SECURITY.md` for the
  vulnerability reporting process and scope, including the documented
  resource-exhaustion caveat around deeply nested list input.
- **Testing**: full CommonMark 0.31.2 spec suite (652/652 examples) plus a
  GFM-extras suite, golden HTML fixtures, an XSS snippet corpus exercised
  against the default sanitized render path, `go test -fuzz` coverage of
  parse→render across every syntax extension, and a resource-exhaustion
  regression guard.
- **Docs**: `docs/Design.md` (architecture, package layout, safety model,
  testing approach, roadmap), pkg.go.dev-visible `Example` functions, and
  a `Concurrency` note on the root package documenting that all top-level
  functions are safe for concurrent use.

[0.1.0]: https://github.com/sriannamalai/markdownviewer/releases/tag/v0.1.0
