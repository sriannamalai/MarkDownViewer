import 'options.dart';

/// One resolvable target found in a parsed document by
/// [collectResolvables]: a link, image, or wiki-link the resolver
/// callback would be asked about at render time.
///
/// [kind] is the ABI-frozen resolver kind int — `0` link, `1` image,
/// `2` wiki-link — the same append-only ints `mdv_render_r` /
/// `mdv_render_doc_r` pass to a C resolver and [MdvResolveKind]'s enum
/// indices mirror (see `ffi/README.md`'s "Resolver callback" table).
///
/// Value type: two [Resolvable]s with the same [kind] and [target] are
/// equal and hash equal, so `Set<Resolvable>` dedupes naturally.
class Resolvable {
  const Resolvable(this.kind, this.target);

  /// ABI-frozen kind int: 0 = link, 1 = image, 2 = wiki-link.
  final int kind;

  /// The raw target exactly as authored in the markdown — the same
  /// string a render-time resolver receives (link/image destination, or
  /// the wiki-link's inner target with no `.md` fallback applied).
  final String target;

  @override
  bool operator ==(Object other) =>
      other is Resolvable && other.kind == kind && other.target == target;

  @override
  int get hashCode => Object.hash(kind, target);

  @override
  String toString() => 'Resolvable(kind: $kind, target: $target)';
}

/// Walks a parsed document (the `Map` from `Mdviewer.parse`) and returns
/// every DISTINCT resolvable target — links (kind 0), images (kind 1),
/// and wiki-links (kind 2) — in document order, footnote definitions
/// included.
///
/// This is the first step of the pre-resolve pattern for async vaults:
/// parse once, collect the targets, prefetch/resolve them with the
/// host's own async I/O, then render with the sync resolver
/// [resolverFromMap] builds from the results. See the plugin README's
/// "Async vaults" recipe.
///
/// Reads the pinned version-1 document codec: nodes are maps with a
/// `kind` string and nest via `children` lists; `link`/`image` carry
/// their target in `destination`, `wikiLink` in `target`.
List<Resolvable> collectResolvables(Map<String, dynamic> doc) {
  // A set literal is a LinkedHashSet: dedupes while keeping insertion
  // (document) order.
  final found = <Resolvable>{};
  void walk(Object? node) {
    if (node is List) {
      for (final child in node) {
        walk(child);
      }
      return;
    }
    if (node is! Map) return;
    final kind = node['kind'];
    if (kind == 'link' || kind == 'image') {
      final destination = node['destination'];
      if (destination is String) {
        found.add(
          Resolvable(kind == 'link' ? 0 : 1, destination),
        );
      }
    } else if (kind == 'wikiLink') {
      final target = node['target'];
      if (target is String) found.add(Resolvable(2, target));
    }
    walk(node['children']);
  }

  walk(doc['children']);
  walk(doc['footnotes']);
  return found.toList(growable: false);
}

/// Builds a sync [MdvResolver] that answers from [resolved]
/// (target → URL, emitted verbatim under the resolver trust contract)
/// and declines everything else with `null` (default resolution
/// applies). An optional [kindFilter] of ABI kind ints (0 link, 1 image,
/// 2 wiki-link) restricts answering to those kinds — e.g. `{1}` resolves
/// images only, even when a link shares the same target string.
MdvResolver resolverFromMap(
  Map<String, String> resolved, {
  Set<int>? kindFilter,
}) {
  return (MdvResolveKind kind, String target) {
    if (kindFilter != null && !kindFilter.contains(kind.index)) return null;
    return resolved[target];
  };
}
