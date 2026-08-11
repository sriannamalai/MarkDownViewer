# libmdviewer (wasm)

[markdownviewer](https://github.com/sriannamalai/markdownviewer) compiled
to WebAssembly — the same `internal/boundary` used by the C FFI build
(`ffi/`), exposed here as a plain ESM module. Render CommonMark+GFM
Markdown to sanitized, self-contained HTML entirely client-side or in
Node, with no CDN and no native dependency.

## Contents

- `index.js` — ESM wrapper (no bundler required)
- `index.d.ts` — TypeScript declarations
- `mdviewer.wasm` — the compiled library
- `wasm_exec.js` — Go's `js/wasm` runtime glue (from the Go toolchain
  that built `mdviewer.wasm`)
- `LICENSE`

## Usage

### Browser

```html
<script type="module">
  import { loadMdviewer } from './index.js';

  const mdv = await loadMdviewer(); // fetches ./mdviewer.wasm next to index.js
  document.body.innerHTML = mdv.render('# Hello *world*');
</script>
```

### Node (>= 20)

```js
import { loadMdviewer } from 'libmdviewer';
import { fileURLToPath } from 'node:url';

// Node's fetch() can't read file: URLs for local wasm off-package-root,
// but a URL pointing at mdviewer.wasm next to index.js resolves fine via
// the package's own import.meta.url. Pass an explicit source only if you
// need to load the .wasm from somewhere else (e.g. bundled elsewhere):
const mdv = await loadMdviewer(new URL('./mdviewer.wasm', import.meta.url));
console.log(mdv.render('# Hello *world*', { fragment: true }));
```

`loadMdviewer(wasmSource?)` accepts a URL string, a `URL`, an
`ArrayBuffer`/`Uint8Array`, a precompiled `WebAssembly.Module`, or
nothing (defaults to `./mdviewer.wasm` next to `index.js`). It's a
singleton — repeated calls return the same promise, so call it once and
reuse the resolved `Mdviewer` object. If `wasmSource` is corrupt or was
built with a mismatched Go toolchain (so the Go runtime fails to start),
the returned promise **rejects** with a descriptive `Error` — it never
hangs forever and never crashes the process with an unhandled rejection.
A rejected load is not cached: calling `loadMdviewer()` again retries
from scratch, while a successful load stays cached for the module's
lifetime.

### Bundlers

There is no bundler config shipped and none is required — `index.js` is
plain ESM with a relative `import './wasm_exec.js'` and a
`new URL('./mdviewer.wasm', import.meta.url)` default. Most bundlers
(Vite, webpack 5, esbuild) resolve that URL pattern to an asset URL
automatically. If yours doesn't, either configure it to copy
`mdviewer.wasm` alongside the built output, or inline it (e.g. as a
base64 asset) and pass the resulting `ArrayBuffer`/`Uint8Array` to
`loadMdviewer()` explicitly. The Node-only `file:` branch inside
`index.js` uses a guarded dynamic `import('node:fs/promises')` —
browsers never execute that branch, and bundlers that statically
resolve all dynamic imports should treat it as external/ignored
(standard handling in modern bundlers).

## API

```ts
const mdv = await loadMdviewer();

mdv.version(): string
mdv.render(md: string, options?: Options): string
mdv.parse(md: string, options?: Options): unknown        // version-1 document JSON, parsed
mdv.renderDoc(doc: unknown, options?: Options): string    // doc: object or JSON string
mdv.asset(name: string): Uint8Array
```

All four calls throw a plain `Error` on failure (parse errors, decode
errors, or an internal panic — the wasm side is always contained, it
never traps/crashes the module).

### Options

Same strict version-1 options JSON as the C FFI and the rest of the
library — see [`ffi/README.md`](https://github.com/sriannamalai/markdownviewer/blob/main/ffi/README.md#options-json) for
the full field table (`theme`, `fragment`, `allowRawHTML`, `mermaid`,
`math`, `highlighting`, `maxWidth`, `sourceMap`, `themeOverrides`,
`stylesheet`, `extraCss`, `codeHeader`, `headingAnchors`, `parser`).
Passing `options` omitted or
`undefined` uses library
defaults (equivalent to `null` at the JSON boundary — no version is
injected on your behalf). `options.resolver` is the one field that
isn't part of the JSON: it's stripped out client-side and passed to the
wasm module as a plain callback. Every other option value must be
JSON-serializable — `undefined` or a function value throws a
`TypeError` rather than being silently dropped by `JSON.stringify`.

Two options added in v0.8.0:

- `extraCss` (string) — host CSS appended after the base styling (or
  after a custom `stylesheet`, whose replace semantics are unchanged).
  Sanitized like `stylesheet`; full-page output only — fragments carry
  no styling.
- `codeHeader` (boolean, default `false`) — wraps each code block in
  header markup (`md-code` > `md-code-header` with a `md-code-lang`
  language label and a `md-code-copy` button). Full pages also include
  inline copy-to-clipboard JS; fragment hosts get the markup+classes
  and wire their own click handler.

Two options added in v0.9.0:

- `headingAnchors` (boolean, default `true`) — slug `id` attributes on
  headings. Set `false` to render headings without ids (intra-page
  `#fragment` links to headings stop resolving). Render-time: applies
  to `render` and `renderDoc`.
- `parser` (object) — nested parser configuration selecting which
  Markdown syntax extensions the parse enables, strictly decoded like
  the top level (unknown/wrong-case nested keys throw, named as
  `parser.<key>`). Omitted = every extension on (the library default).
  `{ commonmarkOnly: true }` starts from pure CommonMark instead; the
  per-extension booleans (`tables`, `strikethrough`, `taskLists`,
  `linkify`, `footnotes`, `definitionLists`, `frontMatter`, `emoji`,
  `wikiLinks`, `math`, `admonitions`) are tristate overrides on top of
  that base — e.g. `{ parser: { wikiLinks: false } }` renders `[[x]]`
  as literal text, and `{ parser: { commonmarkOnly: true, tables: true } }`
  is CommonMark plus GFM tables. Parse-time only: affects `render` and
  `parse`; `renderDoc` decodes and ignores it (the document is already
  parsed). `parser.math` gates `$x$` parsing; top-level `math` gates
  KaTeX rendering.

### Resolver

```ts
mdv.render(md, {
  resolver: (kind, target) => {
    if (kind === 1 /* image */) return `https://cdn.example.com/${target}`;
    return null; // decline: falls back to default resolution
  },
});
```

- `kind`: `0` = link, `1` = image, `2` = wiki-link (ABI-frozen, same
  values as the C FFI's `mdv_resolver_fn`).
- Return a `string` to resolve — it is emitted into the output HTML
  **verbatim** (HTML-escaped only; no safe-URL scheme filtering is
  applied to it, unlike default resolution).
- Return `null`/`undefined` to decline — default resolution applies
  (wiki-links get `.md` appended, then normal `safeURL` scheme
  filtering).
- Throwing from the resolver fails the render with a descriptive
  `Error`; it never crashes the wasm module.
- `parse()` does not take a resolver — resolution is a render-time
  concern.

## Assets

`mdv.asset(name)` returns embedded static assets (as `Uint8Array`) so
fragment-mode hosts can enable diagrams, math, and syntax highlighting
without vendoring anything — `mermaid.js`, `katex.js`, `katex.css`,
`base.css`, `theme-light.css`, `theme-dark.css` — plus
`theme-light.json` / `theme-dark.json`, the theme palette as version-1
JSON data for native/custom renderers. Full-page output (the
default) already embeds everything; you only need `asset()` when
rendering fragments into your own page. See
[`ffi/README.md`](https://github.com/sriannamalai/markdownviewer/blob/main/ffi/README.md#assets) for the full registry
table. Everything is offline — no CDN fetch, ever.

## Size

`mdviewer.wasm` is tens of megabytes uncompressed: it bundles the Go
runtime, goldmark, chroma, and the embedded mermaid/KaTeX assets
(including fonts). Serve it with gzip or brotli — transport compression
cuts this substantially — and cache it aggressively; there is no "lite"
build that trims features.

## Concurrency

The `js/wasm` target is single-threaded: one render/parse/renderDoc
call is in flight at a time inside the module. Concurrent calls from
the host queue on the Go runtime's own scheduler rather than running in
parallel.
