import 'package:flutter/material.dart';

import '../tree.dart';
import 'block_builders.dart';
import 'builders.dart';
import 'palette.dart';

/// The list assembly behind [MdvDocumentView], exposed so hosts can
/// plug a document's items into ANY scrolling widget they choose
/// (`ListView.builder`, slivers, a positioned-scrolling package — the
/// host owns the scrollable, its controller, and its padding).
///
/// A host composes the same document the sealed view renders:
///
/// ```dart
/// final adapter = MdvDocumentAdapter(tree);
/// return adapter.wrap(
///   context,
///   ListView.builder(
///     itemCount: adapter.itemCount,
///     findChildIndexCallback: adapter.findChildIndexCallback,
///     itemBuilder: adapter.itemBuilder,
///   ),
/// );
/// ```
///
/// [MdvDocumentView] itself is reimplemented on this adapter, so the
/// items a host assembles are byte-for-byte what the view renders.
/// Construct one per build — construction is cheap (the key table over
/// the tree's blocks), and the ambient theme is deliberately NOT read
/// at construction: [itemBuilder] and [wrap] take a [BuildContext] and
/// resolve a null [palette] from `Theme.of(context).brightness` at
/// call time, so a theme change re-renders with the right colors.
///
/// # Item keys and duplicate ids
///
/// Blocks are keyed by `(id, occurrenceIndex)`, NOT by bare
/// [MdvBlock.id]: block ids are content hashes, so two byte-identical
/// source blocks SHARE an id by design (see the [MdvBlock.id] contract)
/// — bare-id keys would be duplicate keys, which crashes Flutter's
/// element reconciliation. The pair is encoded as the composed string
/// `"<id>#<occurrence>"` in a [ValueKey]: ids are 16 lowercase-hex
/// chars, so `#` can never appear in one and the encoding is
/// collision-free. The trailing footnotes item (present when
/// [MdvTree.footnotes] is non-empty) carries `ValueKey('mdv-footnotes')`.
/// Content-stable keys keep an edited document's unchanged blocks'
/// state (e.g. code-pane scroll) across re-parses — provided the host's
/// list delegate routes [findChildIndexCallback], which maps those keys
/// back to their (new) indices.
class MdvDocumentAdapter {
  MdvDocumentAdapter(
    this.tree, {
    this.palette,
    this.builders = const MdvBuilders(),
    this.onLinkTap,
    this.imageProvider,
    this.selectable = true,
    this.baseStyle,
  }) {
    // (id, occurrenceIndex) keys — see the class doc — plus the
    // reverse key->index map findChildIndexCallback serves: sliver
    // delegates reconcile children BY INDEX, so without the callback a
    // block's element (and State) would NOT follow its key when an
    // insertion or removal shifts indices — everything downstream would
    // be torn down and rebuilt. With it, keyed elements are relocated
    // to their new indices and block state (an expanded disclosure, a
    // code pane's scroll offset) survives edits elsewhere in the
    // document. This is also what makes the occurrence suffix
    // load-bearing: the map must be injective, which bare duplicate
    // ids could not provide.
    final seen = <String, int>{};
    for (var i = 0; i < tree.blocks.length; i++) {
      final n = seen.update(tree.blocks[i].id, (v) => v + 1, ifAbsent: () => 0);
      final key = ValueKey('${tree.blocks[i].id}#$n');
      _keys.add(key);
      _indexForKey[key] = i;
    }
    if (tree.footnotes.isNotEmpty) {
      _indexForKey[const ValueKey('mdv-footnotes')] = tree.blocks.length;
    }
  }

  /// The typed render tree ([Mdviewer.renderTree] / [Mdviewer.renderTreeDoc]).
  final MdvTree tree;

  /// Colors; null picks [MdvPalette.light] / [MdvPalette.dark] by the
  /// ambient [Theme]'s brightness at [itemBuilder]/[wrap] time (baked
  /// defaults — load the full asset-backed palette with
  /// [MdvPalette.load] for token colors).
  final MdvPalette? palette;

  /// Per-block-kind rendering overrides.
  final MdvBuilders builders;

  /// Tap callback for (non-blocked) links; null renders links styled
  /// but inert. Blocked links are always inert.
  final MdvLinkTapCallback? onLinkTap;

  /// Host image resolution; null renders every image as its alt-text
  /// placeholder.
  final MdvImageResolver? imageProvider;

  /// Whether [wrap] adds a [SelectionArea] (default true).
  final bool selectable;

  /// Merged OVER the default document text style
  /// `TextStyle(color: palette.foreground, fontSize: 16, height: 1.5)`
  /// — so a fontSize-only override (e.g. a host text-scale setting)
  /// keeps the palette color and line height. Null keeps the default.
  final TextStyle? baseStyle;

  final _keys = <ValueKey<String>>[];
  final _indexForKey = <Key, int>{};

  /// One item per top-level block, plus one trailing footnotes item
  /// when [MdvTree.footnotes] is non-empty.
  int get itemCount => tree.blocks.length + (tree.footnotes.isNotEmpty ? 1 : 0);

  /// Builds the keyed item at [index]: block [index] wrapped in its
  /// 12px bottom inter-block padding (0 on the last item) under its
  /// `"<id>#<occurrence>"` [ValueKey]; index `tree.blocks.length`
  /// (when footnotes exist) is the footnotes section under
  /// `ValueKey('mdv-footnotes')`. Matches
  /// `SliverChildBuilderDelegate`'s builder signature.
  Widget itemBuilder(BuildContext context, int index) {
    final scope = _scopeOf(context);
    if (index >= tree.blocks.length) {
      return KeyedSubtree(
        key: const ValueKey('mdv-footnotes'),
        child: _footnotesSection(context, scope),
      );
    }
    return KeyedSubtree(
      key: _keys[index],
      child: Padding(
        padding: EdgeInsets.only(bottom: index == itemCount - 1 ? 0 : 12),
        child: buildMdvBlock(context, tree.blocks[index], scope),
      ),
    );
  }

  /// The injective key→index map for the host's list delegate
  /// (`SliverChildBuilderDelegate.findChildIndexCallback`); null for a
  /// key no item carries. Without it, keyed block state would not
  /// survive index shifts — see the class doc.
  int? findChildIndexCallback(Key key) => _indexForKey[key];

  /// Applies the document shell around the host's assembled [list],
  /// exactly as [MdvDocumentView] does: an optional [SelectionArea]
  /// (per [selectable]), the palette background, and the base text
  /// style ([baseStyle] merged over the default).
  Widget wrap(BuildContext context, Widget list) {
    final palette = _paletteOf(context);
    Widget child = list;
    if (selectable) child = SelectionArea(child: child);
    return ColoredBox(
      color: palette.background,
      child: DefaultTextStyle.merge(style: _baseStyleOf(palette), child: child),
    );
  }

  MdvPalette _paletteOf(BuildContext context) =>
      palette ??
      (Theme.of(context).brightness == Brightness.dark
          ? MdvPalette.dark
          : MdvPalette.light);

  TextStyle _baseStyleOf(MdvPalette palette) => TextStyle(
    color: palette.foreground,
    fontSize: 16,
    height: 1.5,
  ).merge(baseStyle);

  MdvRenderScope _scopeOf(BuildContext context) {
    final palette = _paletteOf(context);
    return MdvRenderScope(
      palette: palette,
      builders: builders,
      onLinkTap: onLinkTap,
      imageProvider: imageProvider,
      selectable: selectable,
      baseStyle: _baseStyleOf(palette),
    );
  }

  /// The trailing footnotes section: a rule, then every definition in
  /// first-reference order. v1 keeps references superscript-only —
  /// tapping a ref does not yet scroll to its definition.
  Widget _footnotesSection(BuildContext context, MdvRenderScope scope) {
    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      mainAxisSize: MainAxisSize.min,
      children: [
        Divider(color: scope.palette.border, height: 24, thickness: 1),
        for (final def in tree.footnotes)
          Padding(
            padding: const EdgeInsets.only(bottom: 8),
            child: buildMdvBlock(context, def, scope),
          ),
      ],
    );
  }
}
