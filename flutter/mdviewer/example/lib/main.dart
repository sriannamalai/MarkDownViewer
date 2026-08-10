import 'dart:convert';

import 'package:flutter/material.dart';
import 'package:mdviewer/mdviewer.dart';
import 'package:webview_flutter/webview_flutter.dart';

void main() {
  runApp(const MdviewerExampleApp());
}

/// A minimal offline demo: renders [sampleDoc] through the mdviewer plugin
/// into a full-page [WebViewWidget], with a light/dark theme toggle in the
/// AppBar. No network access is used anywhere — every asset (mermaid.js,
/// KaTeX, highlight CSS, the resolved logo) is embedded by the render
/// itself or supplied inline by [demoResolver].
class MdviewerExampleApp extends StatelessWidget {
  const MdviewerExampleApp({super.key});

  @override
  Widget build(BuildContext context) {
    return MaterialApp(
      title: 'mdviewer example',
      theme: ThemeData(
        colorScheme: ColorScheme.fromSeed(seedColor: Colors.teal),
      ),
      home: const ViewerPage(),
    );
  }
}

/// Markdown source exercising every renderer feature the example is meant
/// to demonstrate: a mermaid diagram, inline and block KaTeX math, a
/// syntax-highlighted Go fence, a relative link, an image resolved by
/// [demoResolver], and a wiki-link left to the library's default
/// resolution (`.md` suffix).
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

## Links and images

A [relative link](docs/guide.md) that keeps its href untouched, a resolved
logo below, and a wiki-link that falls back to the default `.md` suffix.

![logo](logo.png)

See also [[Wiki Page]].
''';

/// Resolves `logo.png` to an inline SVG data URI; declines every other
/// link, image, or wiki-link target so the library's default resolution
/// applies (relative links pass through verbatim, wiki-links get `.md`
/// appended). Fully offline — no network fetch, ever.
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

class ViewerPage extends StatefulWidget {
  const ViewerPage({super.key});

  @override
  State<ViewerPage> createState() => _ViewerPageState();
}

class _ViewerPageState extends State<ViewerPage> {
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
        title: Text('mdviewer v${_mdv.version}'),
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
