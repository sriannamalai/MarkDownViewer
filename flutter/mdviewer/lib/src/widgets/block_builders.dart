import 'dart:async';

import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
import 'package:flutter_math_fork/flutter_math.dart';

import '../tree.dart';
import 'builders.dart';
import 'inline_spans.dart';

/// Builds the widget for one [MdvBlock], routing through the host's
/// [MdvBuilders] override for the kind (the override receives the
/// built-in widget as `defaultChild`).
Widget buildMdvBlock(
  BuildContext context,
  MdvBlock block,
  MdvRenderScope scope,
) {
  switch (block) {
    case MdvHeading b:
      return _apply(scope.builders.heading, context, b, _heading(b, scope));
    case MdvParagraph b:
      return _apply(
        scope.builders.paragraph,
        context,
        b,
        MdvInlineText(b.children, scope: scope),
      );
    case MdvBlockQuote b:
      return _apply(
        scope.builders.blockQuote,
        context,
        b,
        _blockQuote(b, scope),
      );
    case MdvAdmonition b:
      return _apply(
        scope.builders.admonition,
        context,
        b,
        _admonition(b, scope),
      );
    case MdvList b:
      return _apply(scope.builders.list, context, b, _list(b, scope));
    case MdvCodeBlock b:
      return _apply(
        scope.builders.codeBlock,
        context,
        b,
        MdvCodeBlockView(b, scope: scope),
      );
    case MdvMathBlock b:
      return _apply(scope.builders.mathBlock, context, b, _mathBlock(b, scope));
    case MdvDiagram b:
      return _apply(scope.builders.diagram, context, b, _diagram(b, scope));
    case MdvTable b:
      return _apply(scope.builders.table, context, b, _table(b, scope));
    case MdvThematicBreak b:
      return _apply(
        scope.builders.thematicBreak,
        context,
        b,
        Divider(color: scope.palette.border, height: 24, thickness: 1),
      );
    case MdvHtmlBlock b:
      return _apply(scope.builders.htmlBlock, context, b, _htmlBlock(b, scope));
    case MdvDefinitionList b:
      return _apply(
        scope.builders.definitionList,
        context,
        b,
        _definitionList(context, b, scope),
      );
    // Definition halves normally arrive wrapped in MdvDefinitionList,
    // but render sensibly standalone too (forward safety).
    case MdvDefinitionTerm b:
      return MdvInlineText(
        b.children,
        scope: scope,
        style: scope.baseStyle.copyWith(fontWeight: FontWeight.bold),
      );
    case MdvDefinitionDesc b:
      return Padding(
        padding: const EdgeInsets.only(left: 16),
        child: _blockColumn(context, b.blocks, scope),
      );
    case MdvFootnoteDef b:
      return _apply(
        scope.builders.footnoteDef,
        context,
        b,
        _footnoteDef(context, b, scope),
      );
    case MdvUnknownBlock b:
      assert(() {
        debugPrint('mdviewer: unknown block kind "${b.kind}" skipped');
        return true;
      }());
      return _apply(
        scope.builders.unknown,
        context,
        b,
        const SizedBox.shrink(),
      );
  }
}

Widget _apply<T>(
  MdvBlockWidgetBuilder<T>? builder,
  BuildContext context,
  T node,
  Widget defaultChild,
) {
  return builder == null ? defaultChild : builder(context, node, defaultChild);
}

/// A column of child blocks with inter-block spacing — the layout for
/// every container block's body (quote, admonition, list item, ...).
Widget _blockColumn(
  BuildContext context,
  List<MdvBlock> blocks,
  MdvRenderScope scope, {
  double spacing = 8,
}) {
  return Column(
    crossAxisAlignment: CrossAxisAlignment.start,
    mainAxisSize: MainAxisSize.min,
    children: [
      for (var i = 0; i < blocks.length; i++)
        Padding(
          padding: EdgeInsets.only(top: i == 0 ? 0 : spacing),
          child: buildMdvBlock(context, blocks[i], scope),
        ),
    ],
  );
}

/* ---- Heading -------------------------------------------------------- */

/// Type scale over the base size: five sizes — levels 5 and 6 share the
/// smallest step, with h6 muted to keep the levels distinguishable.
const _headingScale = [1.75, 1.5, 1.25, 1.1, 1.0, 1.0];

Widget _heading(MdvHeading b, MdvRenderScope scope) {
  final base = scope.baseStyle;
  final level = b.level.clamp(1, 6);
  final style = base.copyWith(
    fontSize: (base.fontSize ?? 16) * _headingScale[level - 1],
    fontWeight: FontWeight.w700,
    height: 1.25,
    color: level == 6 ? scope.palette.quoteForeground : base.color,
  );
  final heading = MdvInlineText(b.children, scope: scope, style: style);
  if (b.level > 2) return heading;
  // h1/h2 carry the classic underline rule.
  return Container(
    width: double.infinity,
    padding: const EdgeInsets.only(bottom: 6),
    decoration: BoxDecoration(
      border: Border(bottom: BorderSide(color: scope.palette.border)),
    ),
    child: heading,
  );
}

/* ---- Quote / admonition --------------------------------------------- */

Widget _blockQuote(MdvBlockQuote b, MdvRenderScope scope) {
  final quoted = scope.withBaseStyle(
    scope.baseStyle.copyWith(color: scope.palette.quoteForeground),
  );
  return Container(
    padding: const EdgeInsets.only(left: 12),
    decoration: BoxDecoration(
      border: Border(left: BorderSide(color: scope.palette.border, width: 4)),
    ),
    child: Builder(
      builder: (context) => _blockColumn(context, b.blocks, quoted),
    ),
  );
}

/// GitHub's alert tint per admonition variant (light/dark pairs).
Color mdvAdmonitionColor(String variant, Brightness brightness) {
  final dark = brightness == Brightness.dark;
  return switch (variant) {
    'note' => dark ? const Color(0xFF4493F8) : const Color(0xFF0969DA),
    'tip' => dark ? const Color(0xFF3FB950) : const Color(0xFF1A7F37),
    'important' => dark ? const Color(0xFFAB7DF8) : const Color(0xFF8250DF),
    'warning' => dark ? const Color(0xFFD29922) : const Color(0xFF9A6700),
    'caution' => dark ? const Color(0xFFF85149) : const Color(0xFFCF222E),
    _ => dark ? const Color(0xFF4493F8) : const Color(0xFF0969DA),
  };
}

IconData _admonitionIcon(String variant) => switch (variant) {
  'note' => Icons.info_outline,
  'tip' => Icons.lightbulb_outline,
  'important' => Icons.feedback_outlined,
  'warning' => Icons.warning_amber_outlined,
  'caution' => Icons.report_outlined,
  _ => Icons.info_outline,
};

Widget _admonition(MdvAdmonition b, MdvRenderScope scope) {
  final tint = mdvAdmonitionColor(b.variant, scope.palette.brightness);
  return Container(
    width: double.infinity,
    padding: const EdgeInsets.fromLTRB(12, 10, 12, 10),
    decoration: BoxDecoration(
      color: tint.withValues(alpha: 0.06),
      border: Border(left: BorderSide(color: tint, width: 4)),
      borderRadius: const BorderRadius.horizontal(right: Radius.circular(4)),
    ),
    child: Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      mainAxisSize: MainAxisSize.min,
      children: [
        Row(
          mainAxisSize: MainAxisSize.min,
          children: [
            Icon(_admonitionIcon(b.variant), size: 18, color: tint),
            const SizedBox(width: 6),
            Text(
              b.title,
              style: scope.baseStyle.copyWith(
                color: tint,
                fontWeight: FontWeight.w600,
              ),
            ),
          ],
        ),
        const SizedBox(height: 6),
        Builder(builder: (context) => _blockColumn(context, b.blocks, scope)),
      ],
    ),
  );
}

/* ---- Lists ---------------------------------------------------------- */

Widget _list(MdvList b, MdvRenderScope scope) {
  return Column(
    crossAxisAlignment: CrossAxisAlignment.start,
    mainAxisSize: MainAxisSize.min,
    children: [
      for (var i = 0; i < b.items.length; i++)
        Padding(
          padding: EdgeInsets.only(top: i == 0 ? 0 : (b.tight ? 2 : 8)),
          child: _listItem(b, b.items[i], i, scope),
        ),
    ],
  );
}

Widget _listItem(MdvList list, MdvListItem item, int i, MdvRenderScope scope) {
  final Widget marker;
  if (item.task != null) {
    // Read-only task checkbox (the tree is a rendered document, not a
    // form; toggling is a host feature out of v0.10's scope).
    marker = Icon(
      item.task! ? Icons.check_box : Icons.check_box_outline_blank,
      size: 18,
      color: item.task! ? scope.palette.accent : scope.palette.quoteForeground,
    );
  } else if (list.ordered) {
    marker = Text('${list.start + i}.', style: scope.baseStyle);
  } else {
    marker = Text('•', style: scope.baseStyle);
  }
  return Row(
    crossAxisAlignment: CrossAxisAlignment.start,
    children: [
      SizedBox(
        width: 28,
        child: Align(
          alignment: Alignment.topRight,
          child: Padding(
            padding: const EdgeInsets.only(right: 8, top: 1),
            child: marker,
          ),
        ),
      ),
      Expanded(
        child: Builder(
          builder: (context) => _blockColumn(context, item.blocks, scope),
        ),
      ),
    ],
  );
}

/* ---- Code ----------------------------------------------------------- */

/// A code block: header row (language label + native copy button) over
/// a horizontally scrollable pane of token-colored monospace text.
/// Token runs map through [MdvPalette.tokenColor]; a null [MdvCodeBlock.runs]
/// (highlighting off / unknown language) renders the plain text mono.
class MdvCodeBlockView extends StatelessWidget {
  const MdvCodeBlockView(this.block, {super.key, required this.scope});

  final MdvCodeBlock block;
  final MdvRenderScope scope;

  @override
  Widget build(BuildContext context) {
    final palette = scope.palette;
    final mono = scope.monoStyle().copyWith(
      fontSize: (scope.baseStyle.fontSize ?? 16) * 0.875,
      height: 1.45,
    );
    // Trailing newline would render as a trailing empty line.
    final text = block.text.endsWith('\n')
        ? block.text.substring(0, block.text.length - 1)
        : block.text;
    final runs = block.runs;
    final code = runs == null
        ? TextSpan(text: text, style: mono)
        : TextSpan(
            style: mono,
            children: [
              for (final run in _trimmedRuns(runs))
                TextSpan(
                  text: run.text,
                  style: palette.tokenTextStyle(run.tokenType),
                ),
            ],
          );
    return ClipRRect(
      borderRadius: BorderRadius.circular(6),
      child: Container(
        width: double.infinity,
        color: palette.codeBackground,
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          mainAxisSize: MainAxisSize.min,
          children: [
            Container(
              padding: const EdgeInsets.only(left: 12, right: 4),
              decoration: BoxDecoration(
                border: Border(bottom: BorderSide(color: palette.border)),
              ),
              child: Row(
                children: [
                  Expanded(
                    child: Text(
                      block.label,
                      style: mono.copyWith(
                        fontSize: (mono.fontSize ?? 14) * 0.85,
                        color: palette.quoteForeground,
                      ),
                    ),
                  ),
                  MdvCopyButton(text: block.text, scope: scope),
                ],
              ),
            ),
            SingleChildScrollView(
              scrollDirection: Axis.horizontal,
              padding: const EdgeInsets.all(12),
              child: Text.rich(code),
            ),
          ],
        ),
      ),
    );
  }

  /// The wire promises run texts concatenate to exactly [MdvCodeBlock.text];
  /// mirror the trailing-newline trim on the last run.
  List<MdvTokenRun> _trimmedRuns(List<MdvTokenRun> runs) {
    if (runs.isEmpty || !block.text.endsWith('\n')) return runs;
    final last = runs.last;
    if (!last.text.endsWith('\n')) return runs;
    final trimmed = last.text.substring(0, last.text.length - 1);
    return [
      ...runs.take(runs.length - 1),
      if (trimmed.isNotEmpty)
        MdvTokenRun(text: trimmed, tokenType: last.tokenType),
    ];
  }
}

/// Native copy-to-clipboard button ([Clipboard.setData] — no webview,
/// no bridge): flips its label to "Copied" for two seconds after a tap.
class MdvCopyButton extends StatefulWidget {
  const MdvCopyButton({super.key, required this.text, required this.scope});

  /// The text a tap copies (the code block's full literal text).
  final String text;

  final MdvRenderScope scope;

  @override
  State<MdvCopyButton> createState() => _MdvCopyButtonState();
}

class _MdvCopyButtonState extends State<MdvCopyButton> {
  bool _copied = false;
  Timer? _revert;

  @override
  void dispose() {
    _revert?.cancel();
    super.dispose();
  }

  Future<void> _copy() async {
    await Clipboard.setData(ClipboardData(text: widget.text));
    if (!mounted) return;
    setState(() => _copied = true);
    _revert?.cancel();
    _revert = Timer(const Duration(seconds: 2), () {
      if (mounted) setState(() => _copied = false);
    });
  }

  @override
  Widget build(BuildContext context) {
    final palette = widget.scope.palette;
    final color = _copied ? palette.accent : palette.quoteForeground;
    return TextButton.icon(
      onPressed: _copy,
      icon: Icon(_copied ? Icons.check : Icons.copy_outlined, size: 14),
      label: Text(_copied ? 'Copied' : 'Copy'),
      style: TextButton.styleFrom(
        foregroundColor: color,
        padding: const EdgeInsets.symmetric(horizontal: 8),
        minimumSize: const Size(0, 32),
        textStyle: widget.scope.baseStyle.copyWith(fontSize: 12),
        tapTargetSize: MaterialTapTargetSize.shrinkWrap,
      ),
    );
  }
}

/* ---- Math / diagram -------------------------------------------------- */

Widget _mathBlock(MdvMathBlock b, MdvRenderScope scope) {
  return Center(
    child: SingleChildScrollView(
      scrollDirection: Axis.horizontal,
      padding: const EdgeInsets.symmetric(vertical: 4),
      child: Math.tex(
        b.source,
        textStyle: scope.baseStyle.copyWith(
          fontSize: (scope.baseStyle.fontSize ?? 16) * 1.15,
        ),
        mathStyle: MathStyle.display,
        onErrorFallback: (_) => mdvMathFallback(b.source, scope),
      ),
    ),
  );
}

/// Default diagram rendering: a bordered placeholder — engine label +
/// mono source — never a live rendering. Hosts plug a real renderer via
/// [MdvBuilders.diagram] (the offscreen-SVG mermaid service is an
/// explicit post-v0.10 fast-follow).
Widget _diagram(MdvDiagram b, MdvRenderScope scope) {
  final palette = scope.palette;
  return Container(
    width: double.infinity,
    padding: const EdgeInsets.all(12),
    decoration: BoxDecoration(
      border: Border.all(color: palette.border),
      borderRadius: BorderRadius.circular(6),
    ),
    child: Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      mainAxisSize: MainAxisSize.min,
      children: [
        Text(
          'diagram · ${b.engine}',
          style: scope.baseStyle.copyWith(
            fontSize: (scope.baseStyle.fontSize ?? 16) * 0.75,
            color: palette.quoteForeground,
            letterSpacing: 0.5,
          ),
        ),
        const SizedBox(height: 8),
        SingleChildScrollView(
          scrollDirection: Axis.horizontal,
          child: Text(
            b.source.trimRight(),
            style: scope.monoStyle().copyWith(
              fontSize: (scope.baseStyle.fontSize ?? 16) * 0.8,
              color: palette.quoteForeground,
            ),
          ),
        ),
      ],
    ),
  );
}

/* ---- Table ----------------------------------------------------------- */

Widget _table(MdvTable b, MdvRenderScope scope) {
  final palette = scope.palette;

  Alignment cellAlignment(int column) {
    final align = column < b.alignments.length
        ? b.alignments[column]
        : MdvAlignment.none;
    return switch (align) {
      MdvAlignment.center => Alignment.topCenter,
      MdvAlignment.right => Alignment.topRight,
      MdvAlignment.none || MdvAlignment.left => Alignment.topLeft,
    };
  }

  TextAlign cellTextAlign(int column) {
    final align = column < b.alignments.length
        ? b.alignments[column]
        : MdvAlignment.none;
    return switch (align) {
      MdvAlignment.center => TextAlign.center,
      MdvAlignment.right => TextAlign.right,
      MdvAlignment.none || MdvAlignment.left => TextAlign.start,
    };
  }

  TableRow row(List<List<MdvInline>> cells, {required bool header}) {
    return TableRow(
      decoration: header ? BoxDecoration(color: palette.codeBackground) : null,
      children: [
        for (var c = 0; c < cells.length; c++)
          TableCell(
            verticalAlignment: TableCellVerticalAlignment.top,
            child: Align(
              alignment: cellAlignment(c),
              child: Padding(
                padding: const EdgeInsets.symmetric(
                  horizontal: 10,
                  vertical: 6,
                ),
                child: MdvInlineText(
                  cells[c],
                  scope: scope,
                  textAlign: cellTextAlign(c),
                  style: header
                      ? scope.baseStyle.copyWith(fontWeight: FontWeight.w600)
                      : null,
                ),
              ),
            ),
          ),
      ],
    );
  }

  return SingleChildScrollView(
    scrollDirection: Axis.horizontal,
    child: Table(
      defaultColumnWidth: const IntrinsicColumnWidth(),
      border: TableBorder.all(color: palette.border),
      children: [
        row(b.header, header: true),
        for (final r in b.rows) row(r, header: false),
      ],
    ),
  );
}

/* ---- HTML ------------------------------------------------------------ */

/// Safe-by-default HTML policy (documented contract): the widget layer
/// NEVER renders live HTML. A sanitized block ([MdvHtmlBlock.unsafe]
/// false) shows its tag-stripped plain text in the normal body style —
/// the sanitizer already ran library-side, and stripping tags keeps the
/// human-readable content without a layout engine. An unsafe block
/// (`allowRawHTML` renders — content NOT sanitized) shows a collapsed
/// "Raw HTML (not rendered)" disclosure with the mono source, so raw
/// markup is inspectable but inert. Hosts wanting live HTML supply
/// [MdvBuilders.htmlBlock].
Widget _htmlBlock(MdvHtmlBlock b, MdvRenderScope scope) {
  if (!b.unsafe) {
    final text = stripHtmlTags(b.html);
    if (text.isEmpty) return const SizedBox.shrink();
    return Text(text, style: scope.baseStyle);
  }
  return _RawHtmlDisclosure(html: b.html, scope: scope);
}

class _RawHtmlDisclosure extends StatefulWidget {
  const _RawHtmlDisclosure({required this.html, required this.scope});

  final String html;
  final MdvRenderScope scope;

  @override
  State<_RawHtmlDisclosure> createState() => _RawHtmlDisclosureState();
}

class _RawHtmlDisclosureState extends State<_RawHtmlDisclosure> {
  bool _open = false;

  @override
  Widget build(BuildContext context) {
    final palette = widget.scope.palette;
    return Container(
      decoration: BoxDecoration(
        border: Border.all(color: palette.border),
        borderRadius: BorderRadius.circular(6),
      ),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        mainAxisSize: MainAxisSize.min,
        children: [
          InkWell(
            onTap: () => setState(() => _open = !_open),
            child: Padding(
              padding: const EdgeInsets.symmetric(horizontal: 8, vertical: 6),
              child: Row(
                mainAxisSize: MainAxisSize.min,
                children: [
                  Icon(
                    _open ? Icons.expand_more : Icons.chevron_right,
                    size: 18,
                    color: palette.quoteForeground,
                  ),
                  const SizedBox(width: 4),
                  Text(
                    'Raw HTML (not rendered)',
                    style: widget.scope.baseStyle.copyWith(
                      color: palette.quoteForeground,
                      fontSize: (widget.scope.baseStyle.fontSize ?? 16) * 0.85,
                    ),
                  ),
                ],
              ),
            ),
          ),
          if (_open)
            Container(
              width: double.infinity,
              color: palette.codeBackground,
              padding: const EdgeInsets.all(10),
              child: SingleChildScrollView(
                scrollDirection: Axis.horizontal,
                child: Text(
                  widget.html.trimRight(),
                  style: widget.scope.monoStyle().copyWith(
                    fontSize: (widget.scope.baseStyle.fontSize ?? 16) * 0.85,
                  ),
                ),
              ),
            ),
        ],
      ),
    );
  }
}

/// Tag-stripping for sanitized HTML display: removes tags and comments,
/// decodes the few entities bluemonday output realistically contains.
/// Display-only best effort — NOT a parser, and never used on unsafe
/// content (that path shows source verbatim).
String stripHtmlTags(String html) {
  var s = html.replaceAll(RegExp(r'<!--.*?-->', dotAll: true), '');
  s = s.replaceAll(RegExp(r'<[^>]*>'), '');
  s = s
      .replaceAll('&lt;', '<')
      .replaceAll('&gt;', '>')
      .replaceAll('&quot;', '"')
      .replaceAll('&#39;', "'")
      .replaceAll('&#34;', '"')
      .replaceAll('&amp;', '&');
  return s.trim();
}

/* ---- Definitions / footnotes ----------------------------------------- */

Widget _definitionList(
  BuildContext context,
  MdvDefinitionList b,
  MdvRenderScope scope,
) {
  return Column(
    crossAxisAlignment: CrossAxisAlignment.start,
    mainAxisSize: MainAxisSize.min,
    children: [
      for (var i = 0; i < b.blocks.length; i++)
        Padding(
          padding: EdgeInsets.only(
            top: i == 0
                ? 0
                : b.blocks[i] is MdvDefinitionTerm
                ? 10
                : 4,
          ),
          child: buildMdvBlock(context, b.blocks[i], scope),
        ),
    ],
  );
}

Widget _footnoteDef(
  BuildContext context,
  MdvFootnoteDef b,
  MdvRenderScope scope,
) {
  final small = scope.baseStyle.copyWith(
    fontSize: (scope.baseStyle.fontSize ?? 16) * 0.9,
  );
  return Row(
    crossAxisAlignment: CrossAxisAlignment.start,
    children: [
      SizedBox(
        width: 28,
        child: Text(
          '${b.index}.',
          style: small.copyWith(color: scope.palette.accent),
        ),
      ),
      Expanded(
        child: _blockColumn(context, b.blocks, scope.withBaseStyle(small)),
      ),
    ],
  );
}
