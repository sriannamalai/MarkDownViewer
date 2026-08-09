# libmdviewer

Prebuilt C-shared build of [markdownviewer](https://github.com/sriannamalai/markdownviewer)
— render CommonMark+GFM Markdown to sanitized, self-contained HTML from any
language with a C FFI.

## Contents

- `libmdviewer.dylib` / `libmdviewer.so` / `libmdviewer.dll` — the library
- `libmdviewer.h` — header (prototypes; this README is the API reference)
- `LICENSE`

## API

Six symbols. All functions are thread-safe.

    int  mdv_render     (md, md_len, opts_json, &out_html, &out_len, &err);
    int  mdv_parse      (md, md_len, opts_json, &out_json, &out_len, &err);
    int  mdv_render_doc (doc_json, json_len, opts_json, &out_html, &out_len, &err);
    int  mdv_asset      (name, &out, &out_len, &err);   /* embedded assets */
    void mdv_free       (ptr);
    const char* mdv_version(void);   /* static; do not free */

Return 0 = success. On success the out-buffer is UTF-8 with an uncounted
trailing NUL; on failure the return is non-zero and `err` holds a message.
An unexpected internal failure is reported the same way, with a message
prefixed `panic:` — it never unwinds into the host process. The one
exception is true out-of-memory: like the Go runtime itself, the library
aborts if C `malloc` fails.
**Every returned buffer must be freed with `mdv_free`** (not plain `free`).
`mdv_parse` emits a versioned document-AST JSON; feeding it back through
`mdv_render_doc` lets you parse once and re-render many times (e.g. theme
switching).

## Assets

`mdv_asset(name, ...)` returns embedded static assets so fragment-mode
hosts can enable diagrams, math, and syntax highlighting without
vendoring anything. Registry (append-only, case-sensitive):

| Name | Content |
|---|---|
| `mermaid.js` | offline mermaid bundle |
| `katex.js` | offline KaTeX bundle |
| `katex.css` | KaTeX CSS, all fonts inlined as data: URIs |
| `base.css` | structural stylesheet |
| `theme-light.css` | light theme tokens + light syntax-highlight CSS |
| `theme-dark.css` | dark theme tokens + dark syntax-highlight CSS |

Full-page output (the default) already embeds everything; you only need
`mdv_asset` when rendering fragments into your own page. Apply one
`theme-*.css` per document view; if you ship both and switch at
runtime, scope them yourself. The `--md-*` variables in `theme-*.css` take effect through `base.css`'s rules — apply both for full styling; the chroma highlighting rules work standalone.

Note: the generated header is not const-qualified (a cgo limitation);
the library never writes through input pointers — cast as needed.

## Options JSON

`opts_json` is a NUL-terminated JSON object (NULL or empty = defaults):

    {"version": 1, "theme": "auto", "fragment": false, "allowRawHTML": false,
     "mermaid": true, "math": true, "highlighting": true, "maxWidth": "",
     "sourceMap": false, "themeOverrides": {}, "stylesheet": ""}

All fields optional; unknown fields are an error. `theme` is "light",
"dark", or "auto". `maxWidth` takes any CSS length ("860px", "70ch");
empty = fluid. See the repository README for what each option does.
