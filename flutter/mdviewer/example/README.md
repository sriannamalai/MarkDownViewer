# mdviewer example

A minimal Flutter app demonstrating the `mdviewer` plugin's two display
paths over one sample Markdown document — mermaid diagram, inline/block
KaTeX math, a syntax-highlighted Go fence, a relative link, an image
resolved through a host resolver, and a wiki-link left to the library's
default `.md` resolution:

- **Web page** — sanitized HTML (`Mdviewer.render`/`renderDoc`) in a
  full-page `webview_flutter` `WebViewWidget`, with live mermaid.
- **Native page** (v0.10) — the same document's render tree
  (`Mdviewer.renderTree`) through `MdvDocumentView`: native widgets, no
  webview — selectable text, token-run highlighted code with a native
  copy button, native KaTeX math (`flutter_math_fork`), an async
  `imageProvider`, and the mermaid placeholder (pluggable builder).

Both pages have a light/dark toggle (the web page re-renders from the
already-parsed document via `renderDoc`, not a re-parse; the native page
swaps `MdvPalette`s loaded from the `theme-*.json`/`highlight-*.json`
assets) and the AppBar shows the linked library's version string.

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

On the Native page:

- Text is selectable across blocks; the code fence shows token colors
  from the loaded `highlight-*.json` palette and its Copy button writes
  the clipboard natively (no bridge).
- Math is typeset natively by `flutter_math_fork` — no webview involved.
- The mermaid block shows the bordered placeholder (engine label +
  source): live diagrams are a pluggable builder, not a v0.10 default.
- The theme toggle swaps palettes without re-parsing or rebuilding the
  tree.

On the Web page:

- The mermaid diagram renders as an actual graph, not raw text.
- Inline (`$...$`) and block (`$$...$$`) math is typeset by KaTeX.
- The Go code fence is syntax-colored.
- The `![logo](logo.png)` image shows a teal "resolved!" box — proof the
  `demoResolver` in `lib/main.dart` rewrote it to an inline SVG.
- The `[[Wiki Page]]` link resolves to `Wiki Page.md` (the library's
  default wiki-link resolution — `demoResolver` declines it).
- Tapping the moon/sun icon in the AppBar re-renders the same parsed
  document in the other theme.
