# Changelog

All notable changes to this project are documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/).
This project is pre-1.0 (see `docs/Design.md`'s Status section); until
v1.0.0, minor version bumps may include breaking changes to the `document`
model and renderer options.

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
