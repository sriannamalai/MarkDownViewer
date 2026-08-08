# Changelog

All notable changes to this project are documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/).
This project is pre-1.0 (see `docs/Design.md`'s Status section); until
v1.0.0, minor version bumps may include breaking changes to the `document`
model and renderer options.

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
