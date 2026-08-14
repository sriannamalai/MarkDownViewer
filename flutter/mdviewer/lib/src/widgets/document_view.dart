import 'package:flutter/material.dart';

import '../tree.dart';
import 'builders.dart';
import 'document_adapter.dart';
import 'palette.dart';

/// Renders an [MdvTree] as native Flutter widgets — no webview, no
/// bridge: a lazy [ListView] of the tree's blocks, with the footnotes
/// section (when any) as a trailing item.
///
/// Implemented on [MdvDocumentAdapter] — hosts that need scroll
/// control (jump to a heading, observe positions, persist a reading
/// position) compose the adapter's items into their own scrolling
/// widget instead of using this sealed view; the items are
/// byte-for-byte identical.
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
    this.baseStyle,
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

  /// Merged OVER the default document text style
  /// `TextStyle(color: palette.foreground, fontSize: 16, height: 1.5)`
  /// — so a fontSize-only override (e.g. a host text-scale setting)
  /// keeps the palette color and line height. Null keeps the default.
  final TextStyle? baseStyle;

  @override
  Widget build(BuildContext context) {
    final adapter = MdvDocumentAdapter(
      tree,
      palette: palette,
      builders: builders,
      onLinkTap: onLinkTap,
      imageProvider: imageProvider,
      selectable: selectable,
      baseStyle: baseStyle,
    );
    return adapter.wrap(
      context,
      ListView.builder(
        padding: padding,
        itemCount: adapter.itemCount,
        findChildIndexCallback: adapter.findChildIndexCallback,
        itemBuilder: adapter.itemBuilder,
      ),
    );
  }
}
