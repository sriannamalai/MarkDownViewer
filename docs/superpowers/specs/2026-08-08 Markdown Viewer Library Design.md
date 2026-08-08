# Markdown Viewer Library — Design

**Date:** 2026-08-08
**Status:** Approved
**License:** Apache-2.0

## Purpose

A modern, embeddable Markdown viewer library, written in Go, that any IDE or
GUI tool can use to provide an internal Markdown preview — on Windows, macOS,
and Linux today, with a path to mobile (iOS/Android via Flutter and other
hybrid frameworks) and to dedicated desktop/mobile viewer apps built on top of
it later. The repository is strictly a library/component: no application code
beyond a thin CLI that doubles as a living example.

## Decisions (settled during brainstorming)

| Decision | Choice |
| --- | --- |
| Render model | Layered: stable typed document model (AST) at the core; renderer packages on top — HTML now, native renderers later |
| Core language | Go (goldmark parser internally) |
| Public AST | Our own versioned document model; goldmark is an internal, swappable detail |
| v1 feature scope | GFM baseline + footnotes/anchors, pure-Go syntax highlighting (chroma), Mermaid + KaTeX math, front-matter, emoji shortcodes, definition lists, admonitions/callouts, wiki-links |
| v1 surfaces | Go package API + `mdview` CLI. FFI / WASM / mobile bindings deferred to v1.x, after API freeze |
| License | Apache-2.0 (already committed), NOTICE + third-party attributions, DCO for contributions |

## Architecture

A three-stage pipeline behind one small facade:

```
markdown source ──▶ parse ──▶ document model ──▶ renderer ──▶ output
                 (goldmark,     (our stable,      (HTML now,
                  internal)      typed AST)        native later)
```

**Module path:** `github.com/sriannamalai/markdownviewer` (lowercase — GitHub
URLs are case-insensitive, so the existing `MarkDownViewer` repo name is fine).

### Package layout

```
markdownviewer/
├── mdviewer.go            // facade: Render(src, ...Option) → HTML; Parse(src) → *document.Document
├── document/              // THE public contract: typed nodes (Document, Heading, Paragraph,
│                          //   CodeBlock, Table, TaskItem, Admonition, Math, Diagram…),
│                          //   source positions, front-matter metadata. Zero goldmark imports.
├── parser/                // goldmark + extensions, internal transform → document model
├── render/html/           // walks document model → themed, self-contained HTML
├── theme/                 // built-in light/dark/auto themes, CSS-custom-property based
├── internal/assets/       // go:embed'd mermaid.js, KaTeX, chroma CSS — offline, no CDN
└── cmd/mdview/            // CLI: file/stdin → HTML (v1's living example)
```

**Layering rule:** `document` imports nothing; everyone imports `document`.
Renderers and future FFI/mobile bindings only ever see the document model —
goldmark stays an internal dependency that could be swapped without a major
version bump. The document model is designed for rendering and serialization
(it is what future bindings will marshal across FFI), not as a parser artifact.

Hosts re-render whole documents on change. No incremental rendering in v1 —
full re-parse of typical documents is single-digit milliseconds; incremental
is deliberate YAGNI until profiling proves otherwise.

## HTML rendering, theming, and safety

### Output contract

The HTML renderer produces either:

- a **fragment** (body-only) for hosts that own the surrounding page, or
- a **full document** (standalone HTML, CSS/JS inlined) for webviews and the CLI.

Both are fully self-contained and offline: every asset ships embedded in the
Go binary via `go:embed`. No CDN references, ever — IDE previews must work
air-gapped.

### Theming

Themes are CSS-custom-property sets (`--md-bg`, `--md-fg`, `--md-accent`,
`--md-code-bg`, …) applied over one base stylesheet with GitHub-preview-quality
typography. Built-ins: `light`, `dark`, `auto` (follows
`prefers-color-scheme`). Hosts can:

1. pick a built-in theme,
2. override any subset of variables (e.g. to match IDE editor colors), or
3. supply a full replacement stylesheet.

Chroma syntax-highlight themes are paired with each UI theme so code blocks
always match the surrounding colors.

### Feature rendering & graceful degradation

- **Syntax highlighting:** render-time, pure Go (chroma, 250+ languages).
  Zero client-side JS; highlighting is baked into the emitted HTML.
- **Mermaid & KaTeX** are the only JS-requiring features. Blocks are emitted
  as semantic placeholders (`<pre class="mermaid">`, annotated math spans) and
  the embedded JS activates them. With JS disabled in a host webview the user
  still sees raw diagram source / TeX — degraded, never broken. Both features
  are opt-out via options for hosts wanting a lean bundle.

### Safety — untrusted input is the default

IDE previews render files the user did not necessarily author.

- Raw HTML in markdown is **sanitized by default** (strict allowlist via
  bluemonday); `javascript:` and `data:` URLs are stripped.
- `AllowRawHTML()` is an explicit opt-in for trusted contexts.
- Local image paths resolve through a **host-provided resolver callback**;
  the library never touches the filesystem during render, so hosts fully
  control file access.
- Wiki-links (`[[Page]]`) resolve through the same callback mechanism; the
  default (no resolver installed) emits a relative href to `Page.md`.

## Testing

- **Golden-file tests** drive everything: `testdata/*.md` → expected HTML,
  diffed on every run, regenerated with `-update`. Every feature tier gets
  fixture files.
- **CommonMark + GFM spec suites** run through the full
  parse → document → HTML pipeline, proving the transform layer preserves
  goldmark's compliance.
- **Go native fuzzing** on `Parse` — the viewer must never panic on hostile
  input.
- **Sanitizer tests** against known XSS corpora ("safe by default" is a
  headline claim; it gets adversarial tests).
- **CI:** GitHub Actions matrix (Linux/macOS/Windows), `go vet`,
  `staticcheck`, race detector, coverage gate.

## Open-source hygiene

- **Apache-2.0** (already committed): patent grant, business-friendly,
  IDE-embeddable. Add `NOTICE` carrying attributions.
- **Dependencies** are all MIT/BSD (goldmark MIT, chroma MIT, bluemonday
  BSD-3, mermaid MIT, KaTeX MIT) — Apache-2.0-compatible. Embedded JS/CSS
  assets keep their license headers and are listed in `NOTICE` and
  `third_party/README.md`.
- **`CONTRIBUTING.md`** with DCO sign-off (no CLA bureaucracy),
  **`CODE_OF_CONDUCT.md`** (Contributor Covenant), **`SECURITY.md`** with a
  private vulnerability-report contact.
- **SemVer**, `v0.x` until API freeze; Go module discipline makes version
  tags the API contract.

## Roadmap after v1 (documented, not built now)

1. `v1.x`: C-shared FFI builds (`.so`/`.dylib`/`.dll`) with a small stable C
   API, and a WASM build for browser-side hosts.
2. Mobile bindings: gomobile / Flutter FFI packages.
3. Native render-tree renderer for toolkit-native (non-webview) hosts.

All future surfaces consume the same `document` model — which is why it is
the frozen contract.

## v1 deliverables

- `document`, `parser`, `render/html`, `theme` packages
- `mdview` CLI
- Full test suite (golden files, spec suites, fuzz, sanitizer corpus) + CI
- README with embedding examples
- `NOTICE`, `CONTRIBUTING.md`, `CODE_OF_CONDUCT.md`, `SECURITY.md`,
  `third_party/README.md`
