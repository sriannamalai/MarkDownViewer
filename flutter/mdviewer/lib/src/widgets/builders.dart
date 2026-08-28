import 'package:flutter/widgets.dart';

import '../tree.dart';
import 'palette.dart';

/// Link-tap callback for [MdvDocumentView]: [url] is the tree's fully
/// resolved destination, [blocked] its URL-policy verdict, [source] the
/// link's origin kind (`wikiLink` for `[[...]]`, null for an ordinary
/// link) — the same fields [MdvLink] carries.
///
/// The default link builder never invokes this for a blocked link
/// (blocked links render muted with NO tap affordance); the [blocked]
/// parameter exists so hosts overriding builders can reuse one callback
/// for both cases.
typedef MdvLinkTapCallback =
    void Function(String url, bool blocked, String? source);

/// Host image resolution for [MdvDocumentView]: given an [MdvImage]'s
/// resolved [url] and [alt] text, produce an [ImageProvider] (asset,
/// memory, file, network — the host's policy), or null to decline.
/// Declined, errored, and policy-blocked images render as an alt-text
/// placeholder. Never invoked for a blocked image.
typedef MdvImageResolver =
    Future<ImageProvider?> Function(String url, String alt);

/// Tap callback for a footnote reference marker (wire kind
/// `footnoteRef`): [index] is the tapped [MdvFootnoteRef.index],
/// matching the [MdvFootnoteDef] a host should scroll/jump to. Null
/// renders every reference marker inert (the pre-existing default —
/// superscript-only, no tap affordance).
typedef MdvFootnoteRefTapCallback = void Function(int index);

/// A per-kind block builder override: receives the typed [node] and the
/// [defaultChild] the built-in builder produced, and returns the widget
/// to show instead (wrap, replace, or return [defaultChild] untouched).
typedef MdvBlockWidgetBuilder<T> =
    Widget Function(BuildContext context, T node, Widget defaultChild);

/// Per-block-kind builder overrides for [MdvDocumentView]. Every kind
/// is overridable; a null field keeps the built-in rendering. The
/// built-in child is always constructed and passed in, so an override
/// can decorate it rather than reimplement the kind.
@immutable
class MdvBuilders {
  const MdvBuilders({
    this.heading,
    this.paragraph,
    this.blockQuote,
    this.admonition,
    this.list,
    this.codeBlock,
    this.mathBlock,
    this.diagram,
    this.table,
    this.thematicBreak,
    this.htmlBlock,
    this.definitionList,
    this.footnoteDef,
    this.unknown,
  });

  final MdvBlockWidgetBuilder<MdvHeading>? heading;
  final MdvBlockWidgetBuilder<MdvParagraph>? paragraph;
  final MdvBlockWidgetBuilder<MdvBlockQuote>? blockQuote;
  final MdvBlockWidgetBuilder<MdvAdmonition>? admonition;
  final MdvBlockWidgetBuilder<MdvList>? list;
  final MdvBlockWidgetBuilder<MdvCodeBlock>? codeBlock;
  final MdvBlockWidgetBuilder<MdvMathBlock>? mathBlock;

  /// Diagram host hook (spec: mermaid stays pluggable in v0.10). The
  /// default is a bordered placeholder — engine label + mono source —
  /// never a live rendering.
  final MdvBlockWidgetBuilder<MdvDiagram>? diagram;

  final MdvBlockWidgetBuilder<MdvTable>? table;
  final MdvBlockWidgetBuilder<MdvThematicBreak>? thematicBreak;

  /// HTML host hook. The default NEVER renders live HTML: a sanitized
  /// block shows its tag-stripped plain text; an unsafe (raw,
  /// unsanitized) block shows a collapsed "Raw HTML" disclosure with
  /// the mono source.
  final MdvBlockWidgetBuilder<MdvHtmlBlock>? htmlBlock;

  final MdvBlockWidgetBuilder<MdvDefinitionList>? definitionList;

  /// Builder for one footnote definition row in the trailing footnotes
  /// section (and for any [MdvFootnoteDef] met inline in [MdvTree.blocks]).
  final MdvBlockWidgetBuilder<MdvFootnoteDef>? footnoteDef;

  /// Builder for forward-compat [MdvUnknownBlock] nodes. Default:
  /// renders nothing (with a debug-mode log).
  final MdvBlockWidgetBuilder<MdvUnknownBlock>? unknown;
}

/// Immutable bundle of everything the recursive block/inline builders
/// need — palette, overrides, callbacks, the inherited text style —
/// threaded down the build instead of ambient inherited widgets so
/// nested contexts (blockquote text color, list nesting) are explicit.
@immutable
class MdvRenderScope {
  const MdvRenderScope({
    required this.palette,
    this.builders = const MdvBuilders(),
    this.onLinkTap,
    this.onFootnoteRefTap,
    this.imageProvider,
    this.selectable = true,
    required this.baseStyle,
  });

  final MdvPalette palette;
  final MdvBuilders builders;
  final MdvLinkTapCallback? onLinkTap;

  /// Tap callback for a footnote reference marker; null keeps it inert.
  final MdvFootnoteRefTapCallback? onFootnoteRefTap;
  final MdvImageResolver? imageProvider;
  final bool selectable;

  /// The ambient text style blocks derive theirs from (blockquotes
  /// re-thread it with [MdvPalette.quoteForeground], headings scale it).
  final TextStyle baseStyle;

  /// This scope with a different ambient text style.
  MdvRenderScope withBaseStyle(TextStyle style) => MdvRenderScope(
    palette: palette,
    builders: builders,
    onLinkTap: onLinkTap,
    onFootnoteRefTap: onFootnoteRefTap,
    imageProvider: imageProvider,
    selectable: selectable,
    baseStyle: style,
  );

  /// The monospace style for code, derived from [style] (or
  /// [baseStyle]): platform monospace with the code background NOT
  /// applied (code blocks paint their own panel; [MdvCodeSpan] adds it
  /// per-span).
  TextStyle monoStyle([TextStyle? style]) => (style ?? baseStyle).copyWith(
    fontFamily: 'monospace',
    fontFamilyFallback: const ['Menlo', 'Consolas', 'Courier New'],
  );
}
