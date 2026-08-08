# Third-party assets and dependencies

## Vendored viewer assets (embedded in the public `assets` package)

These are fetched offline by `scripts/fetch-assets.sh` and embedded via
`go:embed` so rendered pages work with zero network access. KaTeX's fonts
are additionally inlined as `data:` URIs into `katex.inline.css` (see
`scripts/inlinefonts/main.go`) so no `fonts/*.woff2` requests are made
either.

| Component | Version | License | Upstream | Local license copy |
|---|---|---|---|---|
| mermaid | 11.4.1 | MIT | https://github.com/mermaid-js/mermaid | `third_party/mermaid/LICENSE` |
| KaTeX (JS + CSS + fonts) | 0.16.21 | MIT | https://github.com/KaTeX/KaTeX | `third_party/katex/LICENSE` |

KaTeX's fonts are distributed under the same MIT license as the rest of the
KaTeX project (they are not separately licensed under SIL OFL upstream).

## Go module dependencies (`go.mod`)

| Module | License | Upstream |
|---|---|---|
| github.com/yuin/goldmark | MIT | https://github.com/yuin/goldmark |
| github.com/yuin/goldmark-emoji | MIT | https://github.com/yuin/goldmark-emoji |
| github.com/alecthomas/chroma/v2 | MIT | https://github.com/alecthomas/chroma |
| github.com/microcosm-cc/bluemonday | BSD-3-Clause | https://github.com/microcosm-cc/bluemonday |
| go.abhg.dev/goldmark/frontmatter | BSD-3-Clause | https://github.com/abhinav/goldmark-frontmatter |
| go.abhg.dev/goldmark/wikilink | BSD-3-Clause | https://github.com/abhinav/goldmark-wikilink |

Indirect dependencies (BurntSushi/toml, aymerick/douceur, dlclark/regexp2/v2,
gorilla/css, golang.org/x/net, gopkg.in/yaml.v3) are pulled in transitively
by the above and carry their own permissive (BSD/MIT-style) licenses; see
each module's source for details.

## Upgrading vendored assets

Run `./scripts/fetch-assets.sh` after bumping `MERMAID_VERSION` and/or
`KATEX_VERSION` in that script. It re-fetches the pinned files from
`cdn.jsdelivr.net`, re-inlines the KaTeX fonts, and refreshes the
`third_party/*/LICENSE` files. Commit the resulting `assets/*`
and `third_party/*/LICENSE` changes together.
