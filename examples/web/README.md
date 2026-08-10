# Browser playground

A single self-contained page that loads `dist/wasm/npm/index.js` and
renders whatever markdown you type into a live preview: mermaid diagrams,
KaTeX math, and chroma-highlighted code blocks all render fully offline
(no CDN, no fonts) — mermaid/KaTeX come from the wasm build's embedded
assets, and the rendered page inlines everything it needs.

## Run

From the repository root:

    ./scripts/build-wasm.sh
    python3 -m http.server 8000   # from the repo root
    # open http://localhost:8000/examples/web/

The page must be served from the repo root so it can reach
`/dist/wasm/npm/index.js`; opening `index.html` directly (`file://`) will
not work because ES module imports need an HTTP origin.

This example is local-only, like `examples/dart` — it is not wired into
CI. The Node harness (`examples/node`) is the CI gate for the wasm
surface.
