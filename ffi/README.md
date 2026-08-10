# libmdviewer

Prebuilt C-shared build of [markdownviewer](https://github.com/sriannamalai/markdownviewer)
— render CommonMark+GFM Markdown to sanitized, self-contained HTML from any
language with a C FFI.

## Contents

- `libmdviewer.dylib` / `libmdviewer.so` / `libmdviewer.dll` — the library
- `libmdviewer.h` — header (prototypes; this README is the API reference)
- `LICENSE`

## API

Nine symbols. All functions are thread-safe; this is unchanged by the
resolver callback — the callback itself runs synchronously on the
calling thread, one call at a time per render (see "Resolver callback"
below).

    int  mdv_render         (md, md_len, opts_json, &out_html, &out_len, &err);
    int  mdv_render_r       (md, md_len, opts_json, resolver, userdata, &out_html, &out_len, &err);
    int  mdv_parse          (md, md_len, opts_json, &out_json, &out_len, &err);
    int  mdv_render_doc     (doc_json, json_len, opts_json, &out_html, &out_len, &err);
    int  mdv_render_doc_r   (doc_json, json_len, opts_json, resolver, userdata, &out_html, &out_len, &err);
    int  mdv_asset          (name, &out, &out_len, &err);   /* embedded assets */
    void* mdv_alloc         (n);                            /* library-heap allocator */
    void mdv_free           (ptr);
    const char* mdv_version (void);   /* static; do not free */

Return 0 = success. On success the out-buffer is UTF-8 with an uncounted
trailing NUL; on failure the return is non-zero and `err` holds a message.
An unexpected internal failure is reported the same way, with a message
prefixed `panic:` — it never unwinds into the host process. The one
exception is true out-of-memory: like the Go runtime itself, the library
aborts if C `malloc` fails.
**Every returned buffer must be freed with `mdv_free`** (not plain `free`).
`mdv_parse` emits a versioned document-AST JSON; feeding it back through
`mdv_render_doc` lets you parse once and re-render many times (e.g. theme
switching). `mdv_render_r` and `mdv_render_doc_r` are their plain
counterparts plus a host resolver callback (below); a NULL `resolver`
behaves identically to `mdv_render` / `mdv_render_doc`.

**Compiler note**: the generated `libmdviewer.h` embeds two static
helper functions from the cgo preamble (`mdv_call_resolver`,
`mdv_malloc_raw`). Hosts compiling with `-Wall` will see two
`-Wunused-function` warnings from the header if they don't reference
both — harmless; silence with `-Wno-unused-function` on the include or
ignore.

## Resolver callback

`mdv_render_r` and `mdv_render_doc_r` accept a host callback that
intercepts link, image, and wiki-link target resolution before default
resolution applies:

    typedef int (*mdv_resolver_fn)(int kind, const char* target,
                                   size_t target_len, void* userdata,
                                   char** out_url, size_t* out_url_len);

Contract:

- **Return value**: `1` = resolved (the library reads `*out_url` /
  `*out_url_len` and copies them; you retain ownership only until the
  call returns — allocate `*out_url` with `mdv_alloc`, never a
  language-native allocator, since the library frees it itself). `0` =
  declined; default resolution applies. Any other return value is a
  contract violation and **fails the render** with a descriptive error.
- **1 with a NULL `*out_url`** is also a contract violation and fails
  the render.
- **`*out_url_len` must be the exact byte length of the buffer** you
  allocated — a length larger than the allocation causes an
  out-of-bounds read (standard C contract territory; the library
  cannot detect it). A length that does not fit in a signed host `int`
  is a contract violation and fails the render rather than being
  truncated.
- Ownership of a non-NULL `*out_url` transfers to the library **only**
  when the callback returns `1` — including when `*out_url_len` turns
  out to be invalid (oversized): the library still frees the buffer,
  even though the render then fails. On **any other** return (`0`, or
  a contract-violating code outside `{0, 1}`), the library never reads
  or frees the out params — it has no ownership claim on a pointer the
  callback didn't hand it via the `1` path. A host that allocates a
  buffer and then declines, or returns an invalid code, keeps that
  allocation: freeing it (or not setting `*out_url` at all) is the
  host's responsibility, and freeing it inside the library would risk
  an invalid free on memory the library never owned.
- `target` is **not NUL-terminated** and is **only valid for the
  duration of the call** — copy it if you need it afterward.
- The callback runs **synchronously on the calling thread** during
  render, one call at a time per render; it must not unwind across the
  boundary (no `longjmp`, no C++ exceptions).
- `userdata` is passed through untouched — use it for callback context.

`kind` (ABI-frozen, append-only):

| Value | Meaning |
|---|---|
| `0` | link |
| `1` | image |
| `2` | wiki-link |

Trust contract: exactly as with the Go `Options.Resolver`, a resolved
URL is emitted **verbatim** into the output HTML (HTML-escaped only —
no scheme allowlist is applied to it). Declined targets fall back to
the library's default resolution (wiki-links get `.md` appended) and
the normal `safeURL` scheme filtering.

## Memory: mdv_alloc

`mdv_alloc(n)` allocates `n` bytes on the **library's** heap. Use it
for every buffer you hand back to the library — currently, the
resolver's `*out_url`. This matters because on Windows the host's CRT
heap and the library's CRT heap can be different heaps: freeing a
host-allocated pointer inside the library (or vice versa) is undefined
behavior. `mdv_alloc` / `mdv_free` are the same allocator on both sides
of that boundary, so pairing them is the only safe way to transfer
ownership across it.

- `mdv_alloc(0)` returns a valid non-NULL pointer.
- `NULL` return means allocation failure — a resolver callback that
  gets NULL back from `mdv_alloc` should return `0` (decline) rather
  than pass NULL as `*out_url`, since 1-with-NULL is a contract
  violation.

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
