import 'dart:convert';

/// Resolver callback kind. The index IS the ABI int used by
/// `mdv_render_r` / `mdv_render_doc_r` (`0` link, `1` image, `2`
/// wiki-link) — see `ffi/README.md`'s "Resolver callback" table.
enum MdvResolveKind { link, image, wikiLink }

/// Host resolver callback for [MdvOptions.resolver]. Return a resolved
/// URL, or `null` to decline (default resolution applies). Called
/// synchronously on the calling thread during a render; a thrown error
/// is recorded, declines the remainder of the render, and is rethrown as
/// a boundary exception after the native call returns (see
/// `_callWithResolver` in `mdviewer_base.dart`).
typedef MdvResolver = String? Function(MdvResolveKind kind, String target);

/// Typed, strict-by-construction mirror of the boundary's nested
/// `"parser"` options object: which Markdown syntax extensions the parse
/// enables (a 1:1 mirror of the Go `parser.Config`).
///
/// Omitting [MdvOptions.parser] entirely means "library default" — every
/// extension on. [commonmarkOnly] starts from pure CommonMark (no
/// extensions) instead; the per-extension fields are tristate overrides
/// on top of that base: `null` (unset) keeps the base's setting, `true`
/// enables, `false` disables. So `MdvParserOptions(wikiLinks: false)`
/// renders `[[x]]` as literal text, and
/// `MdvParserOptions(commonmarkOnly: true, tables: true)` is CommonMark
/// plus GFM tables.
///
/// Parse-time only: affects `render` and `parse`; `renderDoc` receives
/// an already-parsed document, so the boundary decodes and ignores it
/// there. Note [math] gates `$x$` syntax recognition at parse time,
/// while [MdvOptions.math] gates KaTeX rendering.
class MdvParserOptions {
  const MdvParserOptions({
    this.commonmarkOnly,
    this.tables,
    this.strikethrough,
    this.taskLists,
    this.linkify,
    this.footnotes,
    this.definitionLists,
    this.frontMatter,
    this.emoji,
    this.wikiLinks,
    this.math,
    this.admonitions,
  });

  final bool? commonmarkOnly;
  final bool? tables;
  final bool? strikethrough;
  final bool? taskLists;
  final bool? linkify;
  final bool? footnotes;
  final bool? definitionLists;
  final bool? frontMatter;
  final bool? emoji;
  final bool? wikiLinks;
  final bool? math;
  final bool? admonitions;

  /// The nested `"parser"` JSON object: only explicitly-set (non-null)
  /// fields serialize, under their exact boundary key names. An empty
  /// map is still meaningful ("parser present, all defaults") and
  /// behaves identically to omitting [MdvOptions.parser].
  Map<String, dynamic> toJson() {
    final map = <String, dynamic>{};
    if (commonmarkOnly != null) map['commonmarkOnly'] = commonmarkOnly;
    if (tables != null) map['tables'] = tables;
    if (strikethrough != null) map['strikethrough'] = strikethrough;
    if (taskLists != null) map['taskLists'] = taskLists;
    if (linkify != null) map['linkify'] = linkify;
    if (footnotes != null) map['footnotes'] = footnotes;
    if (definitionLists != null) map['definitionLists'] = definitionLists;
    if (frontMatter != null) map['frontMatter'] = frontMatter;
    if (emoji != null) map['emoji'] = emoji;
    if (wikiLinks != null) map['wikiLinks'] = wikiLinks;
    if (math != null) map['math'] = math;
    if (admonitions != null) map['admonitions'] = admonitions;
    return map;
  }
}

/// Typed, strict-by-construction options for [Mdviewer.render],
/// [Mdviewer.parse], and [Mdviewer.renderDoc].
///
/// Every field is typed and named; there is no free-form map, so no
/// unknown key can ever reach the boundary. [toJson] serializes only the
/// fields that were explicitly set (non-null) — an omitted field means
/// "library default", matching the boundary's own NULL/omitted-field
/// semantics (see `ffi/README.md`'s Options JSON table). [resolver] is
/// consumed entirely on the Dart side of the call and never serializes.
class MdvOptions {
  const MdvOptions({
    this.theme,
    this.fragment,
    this.allowRawHTML,
    this.mermaid,
    this.math,
    this.highlighting,
    this.maxWidth,
    this.sourceMap,
    this.themeOverrides,
    this.stylesheet,
    this.extraCss,
    this.codeHeader,
    this.headingAnchors,
    this.parser,
    this.resolver,
  });

  final String? theme;
  final bool? fragment;
  final bool? allowRawHTML;
  final bool? mermaid;
  final bool? math;
  final bool? highlighting;
  final String? maxWidth;
  final bool? sourceMap;
  final Map<String, String>? themeOverrides;
  final String? stylesheet;

  /// Extra CSS appended AFTER whatever base styling applied (base+theme,
  /// or [stylesheet] when set — whose replace semantics are unchanged).
  /// Full-page assembly only; no effect in fragment mode.
  final String? extraCss;

  /// When `true`, code blocks are wrapped in an `md-code` container with a
  /// language-label + Copy-button header (full pages also get the inline
  /// clipboard JS). Default `false`: output unchanged.
  final bool? codeHeader;

  /// Slug `id` attributes on headings (`<h1 id="...">`). Library default
  /// `true`; set `false` to omit them (intra-page `#fragment` links to
  /// headings stop resolving). Render-time: applies to `render` and
  /// `renderDoc`.
  final bool? headingAnchors;

  /// Nested parser configuration (which syntax extensions the parse
  /// enables); see [MdvParserOptions]. `null` means library default —
  /// every extension on.
  final MdvParserOptions? parser;

  final MdvResolver? resolver;

  /// The `opts_json` payload for this options set, or `null` when every
  /// non-[resolver] field is null — the NULL-options path, which asks
  /// the boundary for its own defaults rather than sending `{}`.
  String? toJson() {
    final map = <String, dynamic>{};
    if (theme != null) map['theme'] = theme;
    if (fragment != null) map['fragment'] = fragment;
    if (allowRawHTML != null) map['allowRawHTML'] = allowRawHTML;
    if (mermaid != null) map['mermaid'] = mermaid;
    if (math != null) map['math'] = math;
    if (highlighting != null) map['highlighting'] = highlighting;
    if (maxWidth != null) map['maxWidth'] = maxWidth;
    if (sourceMap != null) map['sourceMap'] = sourceMap;
    if (themeOverrides != null) map['themeOverrides'] = themeOverrides;
    if (stylesheet != null) map['stylesheet'] = stylesheet;
    if (extraCss != null) map['extraCss'] = extraCss;
    if (codeHeader != null) map['codeHeader'] = codeHeader;
    if (headingAnchors != null) map['headingAnchors'] = headingAnchors;
    if (parser != null) map['parser'] = parser!.toJson();
    if (map.isEmpty) return null;
    return jsonEncode(map);
  }
}
