# mdviewer

A Flutter FFI plugin for [MarkDownViewer](https://github.com/sriannamalai/markdownviewer):
sanitized, self-contained HTML rendering — mermaid diagrams, TeX math
(KaTeX), and syntax highlighting for 250+ languages — fully offline. The
Dart API is a thin, typed `dart:ffi` layer over the same thirteen-symbol
C ABI (`ffi/README.md` in the monorepo) that every other MarkDownViewer
host binds; nothing here reimplements parsing or policy.

Two ways to display a document:

- **HTML in a webview** (`render`/`renderDoc` + `webview_flutter` or
  similar) — full fidelity including live mermaid diagrams.
- **Native widgets** (`renderTree` + [`MdvDocumentView`](#native-widget-rendering-mdvdocumentview),
  since v0.10) — no webview, no bridge: selectable text, token-run
  syntax highlighting, native KaTeX math, a native copy button.

See [`example/`](example/) for a complete app with both paths side by
side: a "Web" page rendering mermaid + KaTeX + highlighted code + a
resolver-rewritten image + a wiki-link in a `WebViewWidget`, and a
"Native" page rendering the same document through `MdvDocumentView`.

## Status

Pre-pub.dev. `pubspec.yaml` tracks the plugin's release version
(currently `0.10.1` — a Dart-only release that runs against the v0.10.0
native artifacts; the version handshake matches on major.minor) and carries
`publish_to: none` — this plugin is consumed as a path/git dependency
today; publishing to pub.dev is a separately gated step (see the
monorepo's `docs/Design.md` roadmap and `CHANGELOG.md`).

## Install

Path dependency, from within the `markdownviewer` monorepo checkout:

```yaml
dependencies:
  mdviewer:
    path: ../../flutter/mdviewer   # adjust to your app's location
```

Once published, this becomes an ordinary pub.dev version constraint. Until
then, pin a git dependency with a `path:` pointing at `flutter/mdviewer` if
consuming from outside the monorepo — pin the release's `flutter-v<ver>`
tag (e.g. `ref: flutter-v0.8.0`; the commit whose `tool/checksums.txt`
can fetch that release's binaries), never a raw SHA.

## Populating the native binaries

This plugin's platform dirs (`ios/Frameworks/`, `android/src/main/jniLibs/`)
ship empty — the prebuilt `libmdviewer` binaries are populated by a
`tool/` script, not committed to git (see `.gitignore`). Two ways to get
them:

- **`tool/build_binaries.sh`** — builds from the monorepo source
  (`scripts/build-mobile.sh all`) and installs the result. Use this for
  local development against the current checkout:

  ```bash
  flutter/mdviewer/tool/build_binaries.sh
  ```

- **`tool/fetch_binaries.sh`** — downloads the release-pinned,
  checksum-verified artifacts for the version in `pubspec.yaml` (or
  `MDVIEWER_VERSION`) from a GitHub release, for any version with
  checksums in `tool/checksums.txt` (v0.7.0 onward):

  ```bash
  cd flutter/mdviewer && tool/fetch_binaries.sh
  ```

  (During a pre-release window — `pubspec.yaml` already bumped but the
  release not yet cut — set `MDVIEWER_VERSION=0.8.1` (the latest
  released version) or the default invocation fails: there is no
  release, and no checksum entry, for the bumped version yet.)

  Zip names match `.github/workflows/release-ffi.yml` exactly:
  `libmdviewer-<version>-ios.xcframework.zip` and
  `libmdviewer-<version>-android.zip` (the version has no leading `v`; the
  release tag/download URL keeps it). Verified against
  `tool/checksums.txt`; refuses to download an artifact with no checksum
  entry. Idempotent — a `.fetched-binaries-v<version>` marker skips
  re-fetching until removed.

Run one of these before `flutter pub get`/`flutter run` in a consuming app,
or before `flutter test` in this package.

## API tour

```dart
import 'package:mdviewer/mdviewer.dart';

final html = Mdviewer.instance.render(
  '# Hi\n\n```mermaid\ngraph TD; A-->B;\n```\n',
  options: const MdvOptions(theme: 'dark', fragment: false),
);
```

The calls mirror the C ABI's `mdv_render` / `mdv_render_r` / `mdv_parse` /
`mdv_render_doc` / `mdv_render_doc_r` / `mdv_render_tree` /
`mdv_render_tree_r` / `mdv_render_tree_doc` / `mdv_render_tree_doc_r` /
`mdv_asset` family:

- **`Mdviewer.instance.render(markdown, {options})`** — Markdown to
  sanitized HTML. Uses `mdv_render_r` under the hood when `options.resolver`
  is set, `mdv_render` otherwise.
- **`Mdviewer.instance.parse(markdown, {options})`** — Markdown to a
  versioned document JSON, decoded to a `Map<String, dynamic>`. A resolver
  here is a Dart-side `ArgumentError`, not a boundary call — there is no
  `mdv_parse_r` symbol; resolution is a render-time concern, and parsing
  never resolves.
- **`Mdviewer.instance.renderDoc(doc, {options})`** — re-renders a
  previously `parse`d document (accepts either the decoded `Map` or its
  JSON `String` form) without re-parsing the source — the parse-once/
  render-many path, e.g. for a theme toggle. Uses `mdv_render_doc_r` /
  `mdv_render_doc` the same way `render` picks between the two.
- **`Mdviewer.instance.renderTree(markdown, {options})`** — Markdown to
  the version-1 **native render tree**: the layout-free, fully resolved
  semantic tree native hosts render as platform widgets (no webview),
  returned as the typed `MdvTree` model. Uses `mdv_render_tree_r` when
  `options.resolver` is set, `mdv_render_tree` otherwise — resolver
  semantics identical to `render` (trees carry resolved URLs; a
  policy-blocked destination is `url: ''` + `blocked: true`).
  **`renderTreeRaw`** returns the same tree as the raw decoded
  `Map<String, dynamic>` for hosts walking the wire JSON themselves.
- **`Mdviewer.instance.renderTreeDoc(doc, {options})`** — the render
  tree from a previously `parse`d document (decoded `Map` or JSON
  `String`), without re-parsing the markdown; **`renderTreeDocRaw`** is
  its raw-map twin. Block ids differ from `renderTree`'s: with no
  markdown source at hand for content hashes, every block takes the
  deterministic positional fallback id form.
- **`Mdviewer.instance.asset(name)`** — an embedded static asset
  (`mermaid.js`, `katex.js`, `katex.css`, `base.css`, `theme-light.css`,
  `theme-dark.css`, plus `theme-light.json` / `theme-dark.json` — the
  theme palette as version-1 JSON data for native/custom renderers) as
  `Uint8List`, via `mdv_asset`. See `ffi/README.md`'s
  Assets table for the full registry and what the composed theme CSS
  bundles.
- **`Mdviewer.instance.version`** — the linked library's version string
  (`mdv_version`).

All of these throw `MdviewerException` on a boundary error, carrying the
exact boundary message.

### The typed render-tree model

`MdvTree.fromMap` parses the raw wire map into a sealed `MdvBlock`
hierarchy (`MdvHeading`, `MdvParagraph`, `MdvBlockQuote`, `MdvAdmonition`,
`MdvList` + `MdvListItem`, `MdvCodeBlock` + `MdvTokenRun`, `MdvMathBlock`,
`MdvDiagram`, `MdvTable`, `MdvThematicBreak`, `MdvHtmlBlock`,
`MdvDefinitionList` / `MdvDefinitionTerm` / `MdvDefinitionDesc`,
`MdvFootnoteDef`) and a sealed `MdvInline` hierarchy (`MdvText`,
`MdvEmphasis`, `MdvStrong`, `MdvStrikethrough`, `MdvCodeSpan`, `MdvLink`,
`MdvImage`, `MdvMathInline`, `MdvHardBreak`, `MdvSoftBreak`,
`MdvHtmlInline`, `MdvFootnoteRef`), each carrying its span where the wire
has one. Parsing is strict where the version-1 schema promises — a
missing required field or wrong type throws a `FormatException` naming
the offending path (e.g. `blocks[2].items[0].task`) — with exactly one
tolerance for forward compatibility: an unknown block/inline `kind`
decodes as `MdvUnknownBlock` / `MdvUnknownInline` carrying the raw map,
never a throw.

**The id caveat:** block ids are content hashes, so two byte-identical
source blocks share the same id BY DESIGN (that sharing is what makes
ids diff-stable). Hosts needing unique keys — `ListView.builder`, React-
style reconciliation — must key by `(id, occurrenceIndex)`, never by id
alone.

**Options relevance:** for the tree calls only `parser` (renderTree
only), `headingAnchors`, `highlighting`, `math`, `mermaid`, and
`allowRawHTML` apply; the HTML-only fields (`theme`, `themeOverrides`,
`fragment`, `maxWidth`, `sourceMap`, `stylesheet`, `extraCss`,
`codeHeader`) are decoded and ignored, and spans are always included —
see `ffi/README.md`'s options-relevance table.

## Native widget rendering: MdvDocumentView

`MdvDocumentView` renders an `MdvTree` as native Flutter widgets — a
lazy `ListView` of the tree's blocks, no webview, no bridge:

```dart
class ReaderPage extends StatelessWidget {
  const ReaderPage({super.key, required this.tree, required this.palette});

  final MdvTree tree;      // Mdviewer.instance.renderTree(markdown)
  final MdvPalette palette;

  @override
  Widget build(BuildContext context) {
    return MdvDocumentView(
      tree,
      palette: palette,
      onLinkTap: (url, blocked, source) => openUrl(url),
      imageProvider: _resolveImage, // see the stable-callback note below
    );
  }
}

Future<ImageProvider?> _resolveImage(String url, String alt) async {
  final bytes = await vault.imageBytes(url); // your async I/O
  return bytes == null ? null : MemoryImage(bytes);
}
```

What the defaults give you: selectable text across the whole document
(one `SelectionArea`), token-run syntax-highlighted code with a
horizontal scroll and a **native copy button** (`Clipboard.setData` —
no clipboard bridge, unlike the webview `codeHeader` path), native
KaTeX math via `flutter_math_fork`, tables, lists with read-only
checkboxes, blockquotes, colored admonitions, definition lists, and a
trailing footnotes section. Blocks are keyed by `(id, occurrenceIndex)`
internally, so unchanged blocks keep their widget state (e.g. a code
pane's scroll position) across re-parses of an edited document.

### Palette

Colors come from `MdvPalette`. `null` picks the baked-in
`MdvPalette.light`/`MdvPalette.dark` by the ambient `Theme`'s
brightness — correct base colors, but **no syntax-token colors**. Load
the full asset-backed palette once at startup for highlighted code:

```dart
final palette = await MdvPalette.load(dark: true);
// theme-dark.json + highlight-dark.json via Mdviewer.asset(), parsed —
// the same palettes the HTML renderer's CSS uses. Every color is
// host-overridable: palette.copyWith(background: myBg).
```

The loaded palette also carries the highlight asset's font attributes
(`tokenStyles`: bold/italic/underline per token type — e.g. github-dark
italicizes comments), applied to code token runs alongside
`tokenColors`. Assets without a `styles` map (pre-0.10) load fine and
just style nothing.

### Overriding block rendering

Every block kind is overridable via `MdvBuilders`; the built-in child
is constructed and passed in, so an override can decorate rather than
reimplement:

```dart
MdvDocumentView(
  tree,
  builders: MdvBuilders(
    // Wrap every code block in a card:
    codeBlock: (context, node, defaultChild) =>
        Card(clipBehavior: Clip.antiAlias, child: defaultChild),
    // Replace the mermaid placeholder with your own renderer:
    diagram: (context, node, defaultChild) => MyMermaidView(node.source),
  ),
)
```

Two builders exist specifically as host hooks:

- **`diagram`** — mermaid is deliberately *not* rendered natively in
  v0.10: the default is a bordered placeholder (engine label + mono
  source). An offscreen-webview→SVG service is the planned fast-follow;
  until then, hosts wanting live diagrams plug their own builder.
- **`htmlBlock`** — the default **never renders live HTML**: a
  sanitized block shows its tag-stripped plain text, an unsafe (raw)
  block a collapsed "Raw HTML" disclosure with the mono source,
  regardless of `allowRawHTML`. A host that wants live HTML islands
  overrides this builder and owns that risk.

### Composing your own scrollable: MdvDocumentAdapter

`MdvDocumentView` is a sealed `ListView` — no scroll controller, no
positions. When the host needs scroll control (jump to a heading,
observe visible blocks, persist a reading position), use
`MdvDocumentAdapter` (since v0.10.1): the list assembly behind the view,
exposed so the document's items plug into ANY scrolling widget the host
owns. The view is reimplemented on the adapter, so a host-assembled
list renders byte-for-byte the same items — same
`(id, occurrenceIndex)` keys, footnotes item, and document shell. If
you don't need scroll control, keep using `MdvDocumentView`.

The adapter takes the same parameters as the view minus `padding` (the
host owns the scrollable and its outer padding). Example with the
third-party [`scrollable_positioned_list`](https://pub.dev/packages/scrollable_positioned_list)
package — **not a dependency of this plugin**; add it to your own app's
pubspec if you want positioned scrolling:

```dart
class ReaderPage extends StatefulWidget {
  const ReaderPage({super.key, required this.tree});

  final MdvTree tree;

  @override
  State<ReaderPage> createState() => _ReaderPageState();
}

class _ReaderPageState extends State<ReaderPage> {
  final ItemScrollController scrollController = ItemScrollController();

  @override
  Widget build(BuildContext context) {
    final adapter =
        MdvDocumentAdapter(widget.tree); // cheap; construct per build
    return adapter.wrap(
      context,
      ScrollablePositionedList.builder(
        itemScrollController: scrollController,
        itemCount: adapter.itemCount,
        itemBuilder: adapter.itemBuilder,
      ),
    );
  }
}
```

With a delegate that accepts it (`ListView.builder`, slivers), also pass
`findChildIndexCallback: adapter.findChildIndexCallback` so keyed block
state survives index shifts across re-parses
(`scrollable_positioned_list` has no such parameter — it reconciles by
index internally).

Two line-mapping members connect scroll positions to source lines
(`renderTree` trees carry spans; blocks without a recorded span, and
doc-JSON trees without spans, return null):

- **`adapter.blockIndexForLine(line)`** — the item index for a source
  line, by the nearest-PRECEDING-block rule the HTML scrollspy uses.
  Outline jump: map the tapped heading's `span.startLine` to an index,
  then `scrollController.scrollTo(index: i, duration: ...)`.
- **`adapter.startLineForIndex(index)`** — the inverse: the top visible
  item's start line, e.g. from an `ItemPositionsListener`, for
  persisting a reading position by line (null for the trailing
  footnotes item).

### Notes and limitations

- **Pass a stable `imageProvider` callback** (a top-level function,
  static method, or a field held across builds) — each image resolves
  once and re-resolves only when its url or the callback's *function
  identity* changes, so an inline closure (a new identity every
  `build()`) defeats the memoization and re-fires your resolver on
  every rebuild.
- **Footnote refs are superscript-only in v0.10** — definitions render
  in the trailing footnotes section, but tapping a reference does not
  jump to its definition (needs scroll-to-index control the plain
  `ListView.builder` doesn't offer; planned alongside the Mobile
  native-reader work).
- **Math coverage** — `flutter_math_fork` covers most of KaTeX but not
  all of it; TeX it cannot lay out falls back to styled source text
  (never a crash). Hosts needing full parity override
  `MdvBuilders.mathBlock` (the webview path remains the full-fidelity
  option).
- **An image inside a link is not tappable** — recognizers attach to
  text spans only, and image `WidgetSpan`s are deliberately skipped;
  the surrounding link text still responds.

## Options

`MdvOptions` is a typed, strict-by-construction mirror of the boundary's
options JSON — every field is named and typed, so no unknown key can ever
reach the boundary, and only explicitly-set (non-null) fields serialize
(an omitted field means "library default"). See `ffi/README.md`'s Options
JSON table for the full field list, types, and defaults — `MdvOptions`'
fields track it 1:1 (`theme`, `fragment`, `allowRawHTML`, `mermaid`,
`math`, `highlighting`, `maxWidth`, `sourceMap`, `themeOverrides`,
`stylesheet`, `extraCss`, `codeHeader`, `headingAnchors`, `parser`),
plus the Dart-only `resolver` field below.

Two options landed in v0.8.0:

- **`extraCss`** (`String?`) — CSS appended AFTER whatever base styling
  applied: base+theme CSS in the default path, or the custom `stylesheet`
  when set (`stylesheet`'s replace semantics are unchanged). Full-page
  assembly only — no effect in fragment mode. This is the intended hook
  for host text-scale overrides (`body{font-size:117%}`) and webview
  `@font-face` rules with `data:` URI fonts, without fetching and
  concatenating `base.css` yourself.
- **`codeHeader`** (`bool?`, default off) — when `true`, each code block
  is wrapped as `<div class="md-code">` with a header carrying the fence
  language (`<span class="md-code-lang">`, display-uppercased via CSS;
  `code` when unlabeled) and a `<button class="md-code-copy">Copy</button>`.
  Full pages also get a small inline clipboard handler; fragment hosts
  get the markup and classes and wire their own click handler.

Two options landed in v0.9.0:

- **`headingAnchors`** (`bool?`, library default on) — slug `id`
  attributes on headings (`<h1 id="...">`). Set `false` to render
  headings without ids; intra-page `#fragment` links to headings stop
  resolving. Render-time: applies to `render` and `renderDoc`.
- **`parser`** (`MdvParserOptions?`) — nested parser configuration
  selecting which Markdown syntax extensions the parse enables, a 1:1
  strict-by-construction mirror of the boundary's nested `"parser"`
  object (only explicitly-set fields serialize, under exact key names).
  `null` means library default — every extension on.
  `commonmarkOnly: true` starts from pure CommonMark instead; the
  per-extension `bool?` fields (`tables`, `strikethrough`, `taskLists`,
  `linkify`, `footnotes`, `definitionLists`, `frontMatter`, `emoji`,
  `wikiLinks`, `math`, `admonitions`) are tristate overrides on top of
  that base — unset keeps the base's setting, `true` enables, `false`
  disables. So `MdvParserOptions(wikiLinks: false)` renders `[[x]]` as
  literal text, and `MdvParserOptions(commonmarkOnly: true, tables:
  true)` is CommonMark plus GFM tables. Parse-time only: affects
  `render` and `parse`; `renderDoc` (already-parsed document) decodes
  and ignores it. Note `parser.math` gates `$x$` parsing while the
  top-level `math` gates KaTeX rendering.

## codeHeader in webviews: the copy-button bridge

The `codeHeader` full-page copy script uses `navigator.clipboard`, which
browsers only expose in a **secure context** — and a webview fed via
`loadHtmlString`/`srcdoc` is not one, so the Copy button silently no-ops
there. The fix is a host bridge: inject a capture-phase click listener
that posts the code text over a platform channel, and write the clipboard
natively. With `webview_flutter`:

```dart
controller.addJavaScriptChannel('CodeCopy',
    onMessageReceived: (m) => Clipboard.setData(ClipboardData(text: m.message)));
// After the page loads, shadow the built-in handler (capture phase runs first):
controller.runJavaScript('''
  document.addEventListener('click', (e) => {
    const btn = e.target.closest('.md-code-copy');
    if (!btn) return;
    e.stopPropagation();
    const pre = btn.closest('.md-code')?.querySelector('pre');
    if (pre) CodeCopy.postMessage(pre.innerText);
  }, true);
''');
```

This is the CodeCopy bridge pattern the MDViewer.Mobile app proved
on-device (iOS + Android). Fragment hosts wire the same listener — minus
the `stopPropagation`, since there is no built-in handler to shadow.

## Version handshake

The first `Mdviewer.instance` access reads `mdv_version()` from the
loaded library and compares major.minor against this plugin's own version
(`mdviewerPluginVersion`, kept in sync with `pubspec.yaml` by a test). A
mismatched clean release — stale fetched binaries next to a newer plugin,
or vice versa — throws an `MdviewerException` naming both versions up
front, instead of the confusing downstream boundary errors ("unknown
field ...") version skew otherwise produces. Likely fix: re-run
`tool/fetch_binaries.sh` (or `tool/build_binaries.sh`), or rebuild the
host dylib with `scripts/build-ffi.sh`.

Two deliberate escapes:

- **Source builds are exempt.** A library version that is not a clean
  `X.Y.Z` (git-describe stamps like `0.8.1-6-gae98975`, `dev`, or a bare
  commit hash) was built from source alongside the checkout; strict
  matching would reject every pre-tag development build. Release
  artifacts are always stamped clean `X.Y.Z`, so real skew is caught.
- **`MDVIEWER_SKIP_VERSION_CHECK=1`** in the process environment skips
  the check entirely. This is also the escape for the pre-release window
  described under "Populating the native binaries" (plugin version
  already bumped, latest release still older) when fetching rather than
  building from source.

## Resolver contract

`MdvOptions.resolver` is a synchronous Dart callback:

```dart
typedef MdvResolver = String? Function(MdvResolveKind kind, String target);
```

- **`kind`** is one of `MdvResolveKind.link` (0), `.image` (1), or
  `.wikiLink` (2) — the same ABI-frozen, append-only ints `ffi/README.md`
  documents for `mdv_render_r`/`mdv_render_doc_r`.
- **Trusted, verbatim.** A non-null return is emitted into the output HTML
  as-is (HTML-escaped only — no scheme allowlist applied), exactly like
  the Go `Resolver` and the C ABI's resolver contract. The resolver is the
  host's own escape hatch from the default sanitization/scheme policy —
  don't echo attacker-controlled input back unexamined.
- **Decline via `null`.** Returning `null` falls back to the library's
  default resolution for that target (wiki-links get `.md` appended,
  everything else goes through the normal `safeURL` scheme filtering).
- **Throwing semantics.** A thrown error can't unwind across the native
  callback boundary, so it's caught, recorded, and the callback declines
  (`0`) every remaining target for the rest of that render — the render
  itself typically still completes via default resolution for what's left.
  Once the native call returns, the recorded error is rethrown as a
  `MdviewerException('resolver threw: ...')` regardless of whether the
  underlying render call itself succeeded or failed on its own — if a
  thrown resolver happens to coincide with an unrelated render failure,
  the "resolver threw" message wins.
- **You never touch `mdv_alloc` from Dart.** The plugin allocates the
  returned URL's buffer with `mdv_alloc` internally before handing it to
  the native side, so the library can free it — same ownership contract
  `ffi/README.md`'s "Memory: mdv_alloc" section describes for any C
  caller. Nothing in the public Dart API exposes `mdv_alloc`/`mdv_free`
  directly.
- **Per-call, isolate-local, sequential.** Each `render`/`renderDoc` call
  with a resolver creates its own `ffi.NativeCallable.isolateLocal`,
  closed when the call returns — it is not shared across calls or valid
  from another isolate. The native side calls it synchronously, one
  target at a time, on the calling thread — there is no concurrent
  resolver reentrancy to guard against within a single render.

## Async vaults: pre-resolve helpers

`MdvResolver` is synchronous by contract (the C ABI calls it inline during
the render), but real vaults resolve targets with async I/O — database
lookups, network fetches, platform-channel calls. The pre-resolve helpers
bridge that gap without an async ABI: parse once, collect every target the
render would ask about, resolve them with your own async code, then render
with a sync resolver answering from the finished map.

```dart
final mdv = Mdviewer.instance;

// 1. Parse once.
final doc = mdv.parse(markdown);

// 2. Collect the distinct resolvable targets (kind 0 link, 1 image,
//    2 wiki-link — the same ABI-frozen ints the resolver receives).
final targets = collectResolvables(doc);

// 3. Prefetch asynchronously — this part is yours (vault DB, HTTP, ...).
final resolved = <String, String>{};
for (final t in targets.where((t) => t.kind == 1)) {
  resolved[t.target] = await vault.imageDataUri(t.target);
}

// 4. Render with a sync resolver answering from the map; kindFilter {1}
//    answers images only and declines everything else (default
//    resolution applies to declined targets).
final html = mdv.renderDoc(
  doc,
  options: MdvOptions(resolver: resolverFromMap(resolved, kindFilter: {1})),
);
```

- **`collectResolvables(doc)`** walks the parsed version-1 document
  (footnote definitions included) and returns the distinct
  `Resolvable(kind, target)` pairs in document order. Targets are the raw
  authored strings — exactly what the render-time resolver would receive
  (wiki-link targets carry no `.md` fallback).
- **`resolverFromMap(resolved, {kindFilter})`** builds an `MdvResolver`
  that answers mapped targets verbatim (the normal resolver trust
  contract applies — returned URLs bypass the scheme allowlist) and
  declines the rest with `null`. `kindFilter` restricts answering to the
  given ABI kind ints, e.g. `{1}` for images only even when a link shares
  the same target string.

This codifies the pattern the MDViewer.Mobile app's DocImages proved
on-device: parse → collect → async prefetch (images to `data:` URIs) →
render with the map-backed sync resolver.

## Platform loading model

- **Android** — the bundled `.so` (`android/src/main/jniLibs/<abi>/
  libmdviewer.so`), opened via `DynamicLibrary.open('libmdviewer.so')`.
- **iOS** — statically linked into the app binary (the podspec's
  `vendored_frameworks` + `-force_load`, see below), opened via
  `DynamicLibrary.process()` and resolved by symbol name.
- **Host development / plugin tests (macOS, Linux only)** — no bundled
  binary; the loader checks `MDVIEWER_LIBRARY_PATH` first, then walks up
  from the current directory looking for `dist/ffi/<goos>-<goarch>/
  libmdviewer.dylib` (macOS) or `.so` (Linux) — the layout
  `scripts/build-ffi.sh` produces at the monorepo root. There is no
  Windows fallback in this path: the loader only ever looks for `.dylib`
  on macOS and `.so` everywhere else, so a Windows host would need
  `MDVIEWER_LIBRARY_PATH` pointed at a `.dll` explicitly (untested — CI
  covers macOS via `flutter test`, not Windows).

## Known limitations

- **No Intel-Mac (x86_64) iOS Simulator slice.** `scripts/build-mobile.sh`
  cross-compiles the simulator slice for `GOARCH=arm64` only, so the
  shipped xcframework has no `x86_64` simulator slice. The podspec
  compensates with `EXCLUDED_ARCHS[sdk=iphonesimulator*] = x86_64` so
  CocoaPods doesn't fail looking for a slice that isn't there — but that
  means the example app (and any consumer) can only run on an
  Apple-Silicon simulator or a physical device, not an Intel-Mac
  simulator.
- **`-force_load` is required on the consuming app target**, not just
  this pod — `DynamicLibrary.process()` + `dlsym` only find symbols that
  actually made it into the linked binary, and nothing in a typical
  Runner target references `mdv_render` et al. by name for the linker to
  pull in on its own. The podspec sets this via `user_target_xcconfig`
  (see `ios/mdviewer.podspec`'s comments for why it can't be
  `pod_target_xcconfig`); a consumer with an unusual iOS build setup that
  bypasses CocoaPods' xcconfig inheritance would need to replicate it.
- **iOS first build after `pod install` can fail** with "Build input file
  cannot be found" pointing at the `-force_load`ed
  `$(PODS_XCFRAMEWORKS_BUILD_DIR)/mdviewer/libmdviewer.a` — an Xcode
  build-ordering quirk where the app target links before CocoaPods'
  xcframework copy script has produced the archive. Simply build again:
  the retry succeeds and the issue does not recur until the pods are
  reinstalled. (See the matching note in `ios/mdviewer.podspec`.)
- **pub.dev publish is separately gated** — see Status above.

## See also

- [`example/`](example/) — a runnable Flutter app exercising every call
  and the resolver: the same sample document on a webview page and a
  native `MdvDocumentView` page, light/dark switchable.
- `ffi/README.md` (monorepo root) — the C ABI reference this plugin binds:
  full options table, resolver callback C signature, memory ownership,
  and the asset registry.
- `docs/Design.md` (monorepo root) — architecture and roadmap.
