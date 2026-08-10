# mdviewer example

A minimal Flutter app demonstrating the `mdviewer` plugin: renders a single
Markdown document — mermaid diagram, inline/block KaTeX math, a
syntax-highlighted Go fence, a relative link, an image resolved through a
host `MdvResolver`, and a wiki-link left to the library's default `.md`
resolution — into a full-page `webview_flutter` `WebViewWidget`. The AppBar
has a light/dark theme toggle (re-renders from the already-parsed document
via `Mdviewer.renderDoc`, not a re-parse) and shows the linked library's
version string.

Everything is fully offline: mermaid.js, KaTeX, and the highlight CSS are
embedded by the render itself, and the demo image is resolved to an inline
SVG `data:` URI rather than fetched over the network.

## Running it

From this directory:

```bash
../tool/build_binaries.sh   # builds libmdviewer from the repo and installs
                             # it into the plugin's ios/ and android/ dirs
flutter run                 # pick an iOS simulator or Android emulator
```

`webview_flutter` needs JavaScript enabled for mermaid to render — the
example already does this via
`WebViewController..setJavaScriptMode(JavaScriptMode.unrestricted)`.

## What to look for

- The mermaid diagram renders as an actual graph, not raw text.
- Inline (`$...$`) and block (`$$...$$`) math is typeset by KaTeX.
- The Go code fence is syntax-colored.
- The `![logo](logo.png)` image shows a teal "resolved!" box — proof the
  `demoResolver` in `lib/main.dart` rewrote it to an inline SVG.
- The `[[Wiki Page]]` link resolves to `Wiki Page.md` (the library's
  default wiki-link resolution — `demoResolver` declines it).
- Tapping the moon/sun icon in the AppBar re-renders the same parsed
  document in the other theme.
