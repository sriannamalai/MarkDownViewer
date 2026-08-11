import 'package:flutter/material.dart';

import '../tree.dart';
import 'block_builders.dart';
import 'builders.dart';
import 'palette.dart';

/// Renders an [MdvTree] as native Flutter widgets — no webview, no
/// bridge: a lazy [ListView] of the tree's blocks, with the footnotes
/// section (when any) as a trailing item.
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
/// collision-free. A record `ValueKey((id, n))` would be equally
/// correct (records are value-equal); the composed string is chosen
/// because it reads directly in the widget inspector and in test
/// failures. Content-stable keys keep an edited document's unchanged
/// blocks' state (e.g. code-pane scroll) across re-parses.
///
/// # Ambient requirements
///
/// Expects a [MaterialApp] (or at least `Directionality` + `Material`)
/// ancestor. With [selectable] true (the default) the whole document
/// sits in one [SelectionArea], so cross-block text selection works.
class MdvDocumentView extends StatelessWidget {
  const MdvDocumentView(
    this.tree, {
    super.key,
    this.palette,
    this.builders = const MdvBuilders(),
    this.onLinkTap,
    this.imageProvider,
    this.selectable = true,
    this.padding = const EdgeInsets.all(16),
  });

  /// The typed render tree ([Mdviewer.renderTree] / [Mdviewer.renderTreeDoc]).
  final MdvTree tree;

  /// Colors; null picks [MdvPalette.light] / [MdvPalette.dark] by the
  /// ambient [Theme]'s brightness (baked defaults — load the full
  /// asset-backed palette with [MdvPalette.load] for token colors).
  final MdvPalette? palette;

  /// Per-block-kind rendering overrides.
  final MdvBuilders builders;

  /// Tap callback for (non-blocked) links; null renders links styled
  /// but inert. Blocked links are always inert.
  final MdvLinkTapCallback? onLinkTap;

  /// Host image resolution; null renders every image as its alt-text
  /// placeholder.
  final MdvImageResolver? imageProvider;

  /// Wraps the document in a [SelectionArea] when true (default).
  final bool selectable;

  /// Scroll padding around the document.
  final EdgeInsetsGeometry padding;

  @override
  Widget build(BuildContext context) {
    final palette =
        this.palette ??
        (Theme.of(context).brightness == Brightness.dark
            ? MdvPalette.dark
            : MdvPalette.light);
    final scope = MdvRenderScope(
      palette: palette,
      builders: builders,
      onLinkTap: onLinkTap,
      imageProvider: imageProvider,
      selectable: selectable,
      baseStyle: TextStyle(
        color: palette.foreground,
        fontSize: 16,
        height: 1.5,
      ),
    );

    // (id, occurrenceIndex) keys — see the class doc — plus the
    // reverse key->index map findChildIndexCallback needs: the sliver
    // delegate reconciles children BY INDEX, so without the callback a
    // block's element (and State) would NOT follow its key when an
    // insertion or removal shifts indices — everything downstream would
    // be torn down and rebuilt. With it, keyed elements are relocated
    // to their new indices and block state (an expanded disclosure, a
    // code pane's scroll offset) survives edits elsewhere in the
    // document. This is also what makes the occurrence suffix
    // load-bearing: the map must be injective, which bare duplicate
    // ids could not provide.
    final keys = <ValueKey<String>>[];
    final indexForKey = <Key, int>{};
    final seen = <String, int>{};
    for (var i = 0; i < tree.blocks.length; i++) {
      final n = seen.update(tree.blocks[i].id, (v) => v + 1, ifAbsent: () => 0);
      final key = ValueKey('${tree.blocks[i].id}#$n');
      keys.add(key);
      indexForKey[key] = i;
    }

    final hasFootnotes = tree.footnotes.isNotEmpty;
    final itemCount = tree.blocks.length + (hasFootnotes ? 1 : 0);
    if (hasFootnotes) {
      indexForKey[const ValueKey('mdv-footnotes')] = tree.blocks.length;
    }

    Widget list = ListView.builder(
      padding: padding,
      itemCount: itemCount,
      findChildIndexCallback: (key) => indexForKey[key],
      itemBuilder: (context, i) {
        if (i >= tree.blocks.length) {
          return KeyedSubtree(
            key: const ValueKey('mdv-footnotes'),
            child: _footnotesSection(context, scope),
          );
        }
        return KeyedSubtree(
          key: keys[i],
          child: Padding(
            padding: EdgeInsets.only(bottom: i == itemCount - 1 ? 0 : 12),
            child: buildMdvBlock(context, tree.blocks[i], scope),
          ),
        );
      },
    );
    if (selectable) list = SelectionArea(child: list);
    return ColoredBox(
      color: palette.background,
      child: DefaultTextStyle.merge(style: scope.baseStyle, child: list),
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
