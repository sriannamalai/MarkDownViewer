import 'dart:convert';

import 'package:flutter/material.dart';
import 'package:mdviewer/mdviewer.dart';
import 'package:webview_flutter/webview_flutter.dart';

void main() {
  runApp(const MdviewerExampleApp());
}

/// A minimal offline demo with two pages: "Web" renders [sampleDoc]
/// through the plugin's HTML renderer into a [WebViewWidget] (the v0.7
/// path), "Native" renders the SAME document's render tree through
/// [MdvDocumentView] — no webview, no bridge. Both offer a light/dark
/// toggle. No network access is used anywhere.
class MdviewerExampleApp extends StatelessWidget {
  const MdviewerExampleApp({super.key});

  @override
  Widget build(BuildContext context) {
    return MaterialApp(
      title: 'mdviewer example',
      theme: ThemeData(
        colorScheme: ColorScheme.fromSeed(seedColor: Colors.teal),
      ),
      home: const HomePage(),
    );
  }
}

/// Markdown source exercising every renderer feature the example is meant
/// to demonstrate: a mermaid diagram, inline and block KaTeX math, a
/// syntax-highlighted Go fence, tables, task lists, an admonition, a
/// footnote, a relative link, an image resolved by the host, and a
/// wiki-link left to the library's default resolution (`.md` suffix).
const sampleDoc = '''
# mdviewer example

A single offline document exercising mermaid, KaTeX, syntax highlighting,
and link/image resolution.

## Diagram

```mermaid
graph TD
    A[Markdown] --> B[mdviewer core]
    B --> C[Sanitized HTML]
```

## Math

Inline: \$E = mc^2\$

Block:

\$\$
\\int_0^1 x^2 \\, dx = \\frac{1}{3}
\$\$

## Code

```go
package main

import "fmt"

func main() {
	fmt.Println("hello from mdviewer")
}
```

## Blocks

> [!TIP]
> The **native** page renders this admonition as a tinted panel —
> no webview involved.

> A regular blockquote, for contrast.

| Feature   | Web page | Native page |
| --------- | :------: | ----------: |
| Mermaid   | live     | placeholder |
| KaTeX     | live     | native      |
| Copy code | JS       | Clipboard   |

- [x] typed render tree
- [ ] mermaid offscreen service (fast-follow)
- plain item with `code span` and ~~strike~~

Footnotes work too.[^1]

[^1]: Rendered at the end of the native page.

## Links and images

A [relative link](docs/guide.md) that keeps its href untouched, a resolved
logo below, and a wiki-link that falls back to the default `.md` suffix.

![logo](logo.png)

See also [[Wiki Page]].
''';

/// Resolves `logo.png` to an inline SVG data URI for the WEB page;
/// declines every other link, image, or wiki-link target so the
/// library's default resolution applies (relative links pass through
/// verbatim, wiki-links get `.md` appended). Fully offline — no network
/// fetch, ever. (The native page resolves images host-side instead, via
/// [MdvDocumentView.imageProvider] — see [NativeViewerPage].)
String? demoResolver(MdvResolveKind kind, String target) {
  if (kind == MdvResolveKind.image && target == 'logo.png') {
    const svg = '''
<svg xmlns="http://www.w3.org/2000/svg" width="160" height="48">
  <rect width="160" height="48" rx="6" fill="#0f766e"/>
  <text x="80" y="30" font-family="sans-serif" font-size="18" fill="white"
        text-anchor="middle">resolved!</text>
</svg>
''';
    final b64 = base64Encode(utf8.encode(svg));
    return 'data:image/svg+xml;base64,$b64';
  }
  return null; // decline: default resolution applies
}

class HomePage extends StatefulWidget {
  const HomePage({super.key});

  @override
  State<HomePage> createState() => _HomePageState();
}

class _HomePageState extends State<HomePage> {
  int _page = 0;

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      body: IndexedStack(
        index: _page,
        children: const [NativeViewerPage(), WebViewerPage()],
      ),
      bottomNavigationBar: NavigationBar(
        selectedIndex: _page,
        onDestinationSelected: (i) => setState(() => _page = i),
        destinations: const [
          NavigationDestination(
            icon: Icon(Icons.widgets_outlined),
            label: 'Native',
          ),
          NavigationDestination(icon: Icon(Icons.public), label: 'Web'),
        ],
      ),
    );
  }
}

/// The v0.10 native path: `renderTree` once, then [MdvDocumentView] —
/// KaTeX math laid out natively (flutter_math_fork), token-run syntax
/// coloring from `highlight-*.json`, a native Clipboard copy button,
/// and images resolved from bundled Flutter assets by the host
/// callback. The mermaid block shows the default placeholder (the
/// offscreen-SVG service is a post-v0.10 fast-follow).
class NativeViewerPage extends StatefulWidget {
  const NativeViewerPage({super.key});

  @override
  State<NativeViewerPage> createState() => _NativeViewerPageState();
}

class _NativeViewerPageState extends State<NativeViewerPage> {
  final _mdv = Mdviewer.instance;
  late final MdvTree _tree;
  MdvPalette? _palette;
  bool _dark = false;

  @override
  void initState() {
    super.initState();
    // Parse + build the tree once; theme toggles swap only the palette.
    _tree = _mdv.renderTree(sampleDoc);
    _loadPalette();
  }

  Future<void> _loadPalette() async {
    final palette = await MdvPalette.load(dark: _dark);
    if (mounted) setState(() => _palette = palette);
  }

  void _toggleTheme() {
    setState(() => _dark = !_dark);
    _loadPalette();
  }

  Future<ImageProvider?> _resolveImage(String url, String alt) async {
    // Host-side image policy: bundled Flutter assets only.
    if (url == 'logo.png') return const AssetImage('assets/logo.png');
    return null; // anything else -> alt-text placeholder
  }

  @override
  Widget build(BuildContext context) {
    final palette = _palette ?? (_dark ? MdvPalette.dark : MdvPalette.light);
    return Scaffold(
      appBar: AppBar(
        title: Text('native v${_mdv.version}'),
        actions: [
          IconButton(
            icon: Icon(_dark ? Icons.light_mode : Icons.dark_mode),
            tooltip: _dark ? 'Switch to light theme' : 'Switch to dark theme',
            onPressed: _toggleTheme,
          ),
        ],
      ),
      body: MdvDocumentView(
        _tree,
        palette: palette,
        imageProvider: _resolveImage,
        onLinkTap: (url, blocked, source) {
          ScaffoldMessenger.of(context)
            ..hideCurrentSnackBar()
            ..showSnackBar(
              SnackBar(
                content: Text(
                  source == 'wikiLink' ? 'wiki link: $url' : 'link: $url',
                ),
                duration: const Duration(seconds: 2),
              ),
            );
        },
      ),
    );
  }
}

/// The v0.7 webview path, unchanged: full-page HTML render (mermaid and
/// KaTeX live via embedded JS) loaded into a WebView.
class WebViewerPage extends StatefulWidget {
  const WebViewerPage({super.key});

  @override
  State<WebViewerPage> createState() => _WebViewerPageState();
}

class _WebViewerPageState extends State<WebViewerPage> {
  final _mdv = Mdviewer.instance;
  late final Map<String, dynamic> _doc;
  late final WebViewController _controller;
  bool _dark = false;

  @override
  void initState() {
    super.initState();
    // Parse once; re-render from the parsed doc on every theme toggle
    // instead of re-parsing the source markdown.
    _doc = _mdv.parse(sampleDoc);
    _controller = WebViewController()
      ..setJavaScriptMode(JavaScriptMode.unrestricted)
      ..setBackgroundColor(const Color(0x00000000));
    _renderInto(_controller);
  }

  void _renderInto(WebViewController controller) {
    final html = _mdv.renderDoc(
      _doc,
      options: MdvOptions(
        theme: _dark ? 'dark' : 'light',
        resolver: demoResolver,
      ),
    );
    controller.loadHtmlString(html);
  }

  void _toggleTheme() {
    setState(() => _dark = !_dark);
    _renderInto(_controller);
  }

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(
        title: Text('web v${_mdv.version}'),
        actions: [
          IconButton(
            icon: Icon(_dark ? Icons.light_mode : Icons.dark_mode),
            tooltip: _dark ? 'Switch to light theme' : 'Switch to dark theme',
            onPressed: _toggleTheme,
          ),
        ],
      ),
      body: WebViewWidget(controller: _controller),
    );
  }
}
