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
    if (map.isEmpty) return null;
    return jsonEncode(map);
  }
}
