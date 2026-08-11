import 'package:flutter/gestures.dart';
import 'package:flutter/material.dart';
import 'package:flutter_math_fork/flutter_math.dart';

import '../tree.dart';
import 'builders.dart';

/// Renders a list of [MdvInline] nodes as one rich text run.
///
/// Owns the [TapGestureRecognizer]s its link spans need (created per
/// build, disposed on rebuild/dispose — recognizers embedded in
/// [TextSpan]s are not disposed by the framework). Selection comes from
/// the enclosing [SelectionArea] that [MdvDocumentView] installs when
/// `selectable` is true; the spans themselves stay plain [Text.rich].
class MdvInlineText extends StatefulWidget {
  const MdvInlineText(
    this.inlines, {
    super.key,
    required this.scope,
    this.style,
    this.textAlign,
  });

  final List<MdvInline> inlines;
  final MdvRenderScope scope;

  /// Effective style for the run; defaults to [MdvRenderScope.baseStyle].
  final TextStyle? style;

  final TextAlign? textAlign;

  @override
  State<MdvInlineText> createState() => _MdvInlineTextState();
}

class _MdvInlineTextState extends State<MdvInlineText> {
  final List<GestureRecognizer> _recognizers = [];

  void _disposeRecognizers() {
    for (final r in _recognizers) {
      r.dispose();
    }
    _recognizers.clear();
  }

  @override
  void dispose() {
    _disposeRecognizers();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    _disposeRecognizers();
    final style = widget.style ?? widget.scope.baseStyle;
    return Text.rich(
      TextSpan(
        style: style,
        children: [for (final node in widget.inlines) _span(node, style)],
      ),
      textAlign: widget.textAlign ?? TextAlign.start,
    );
  }

  InlineSpan _span(MdvInline node, TextStyle style) {
    final scope = widget.scope;
    final palette = scope.palette;
    switch (node) {
      case MdvText t:
        return TextSpan(text: t.value, style: style);
      case MdvSoftBreak _:
        return const TextSpan(text: ' ');
      case MdvHardBreak _:
        return const TextSpan(text: '\n');
      case MdvEmphasis e:
        final s = style.copyWith(fontStyle: FontStyle.italic);
        return TextSpan(children: [for (final c in e.children) _span(c, s)]);
      case MdvStrong e:
        final s = style.copyWith(fontWeight: FontWeight.bold);
        return TextSpan(children: [for (final c in e.children) _span(c, s)]);
      case MdvStrikethrough e:
        final s = style.copyWith(decoration: TextDecoration.lineThrough);
        return TextSpan(children: [for (final c in e.children) _span(c, s)]);
      case MdvCodeSpan c:
        return TextSpan(
          text: c.value,
          style: scope
              .monoStyle(style)
              .copyWith(backgroundColor: palette.codeBackground),
        );
      case MdvLink l:
        if (l.blocked) {
          // Policy-blocked destination: muted, and deliberately NO tap
          // affordance — a blocked link must not be invokable.
          final s = style.copyWith(
            color: palette.quoteForeground,
            decoration: TextDecoration.lineThrough,
          );
          return TextSpan(children: [for (final c in l.children) _span(c, s)]);
        }
        final s = style.copyWith(
          color: palette.accent,
          decoration: TextDecoration.underline,
          decorationColor: palette.accent,
        );
        TapGestureRecognizer? recognizer;
        final onLinkTap = scope.onLinkTap;
        if (onLinkTap != null) {
          recognizer = TapGestureRecognizer()
            ..onTap = () => onLinkTap(l.url, false, l.source);
          _recognizers.add(recognizer);
        }
        return TextSpan(
          children: [
            for (final c in l.children)
              _withRecognizer(_span(c, s), recognizer),
          ],
        );
      case MdvImage i:
        return WidgetSpan(
          alignment: PlaceholderAlignment.middle,
          child: MdvInlineImage(i, scope: scope),
        );
      case MdvMathInline m:
        return WidgetSpan(
          alignment: PlaceholderAlignment.middle,
          baseline: TextBaseline.alphabetic,
          child: Math.tex(
            m.source,
            textStyle: style,
            mathStyle: m.display ? MathStyle.display : MathStyle.text,
            onErrorFallback: (_) => mdvMathFallback(m.source, scope),
          ),
        );
      case MdvHtmlInline h:
        // Never rendered live (matching the htmlBlock policy): the
        // (sanitized or raw) markup shows as muted mono source text.
        return TextSpan(
          text: h.html,
          style: scope
              .monoStyle(style)
              .copyWith(color: palette.quoteForeground),
        );
      case MdvFootnoteRef r:
        return WidgetSpan(
          alignment: PlaceholderAlignment.top,
          child: Text(
            '${r.index}',
            style: style.copyWith(
              color: palette.accent,
              fontSize: (style.fontSize ?? 14) * 0.7,
              fontFeatures: const [FontFeature.superscripts()],
            ),
          ),
        );
      case MdvUnknownInline u:
        assert(() {
          debugPrint('mdviewer: unknown inline kind "${u.kind}" skipped');
          return true;
        }());
        return const TextSpan(text: '');
    }
  }

  /// Attaches [recognizer] throughout a span subtree. Hit-testing
  /// resolves to the LEAF [TextSpan] under the pointer and consults
  /// only that span's recognizer — a recognizer on the link's outer
  /// span alone would never fire — so every text-bearing descendant
  /// (e.g. emphasis inside a link) gets the recognizer attached.
  InlineSpan _withRecognizer(
    InlineSpan span,
    TapGestureRecognizer? recognizer,
  ) {
    if (recognizer == null || span is! TextSpan) return span;
    return TextSpan(
      text: span.text,
      style: span.style,
      recognizer: recognizer,
      children: span.children == null
          ? null
          : [for (final c in span.children!) _withRecognizer(c, recognizer)],
    );
  }
}

/// The shared math-error fallback (spec: unsupported TeX → styled
/// source text, never a crash): the raw TeX source, monospaced, on the
/// code background.
Widget mdvMathFallback(String source, MdvRenderScope scope) {
  return Container(
    padding: const EdgeInsets.symmetric(horizontal: 4, vertical: 2),
    decoration: BoxDecoration(
      color: scope.palette.codeBackground,
      borderRadius: BorderRadius.circular(3),
    ),
    child: Text(source, style: scope.monoStyle()),
  );
}

/// One inline image: resolves through the host's [MdvImageResolver]
/// once (memoized against the resolved url), shows the provided image,
/// and falls back to an alt-text placeholder when the image is
/// policy-blocked, the host declines (null) or errors, no resolver is
/// installed, or the provided bytes fail to decode.
class MdvInlineImage extends StatefulWidget {
  const MdvInlineImage(this.node, {super.key, required this.scope});

  final MdvImage node;
  final MdvRenderScope scope;

  @override
  State<MdvInlineImage> createState() => _MdvInlineImageState();
}

class _MdvInlineImageState extends State<MdvInlineImage> {
  late Future<ImageProvider?> _future;

  @override
  void initState() {
    super.initState();
    _future = _resolve();
  }

  @override
  void didUpdateWidget(MdvInlineImage oldWidget) {
    super.didUpdateWidget(oldWidget);
    if (oldWidget.node.url != widget.node.url ||
        oldWidget.node.blocked != widget.node.blocked ||
        oldWidget.scope.imageProvider != widget.scope.imageProvider) {
      _future = _resolve();
    }
  }

  Future<ImageProvider?> _resolve() {
    final resolver = widget.scope.imageProvider;
    if (widget.node.blocked || resolver == null) {
      return Future<ImageProvider?>.value(null);
    }
    return resolver(widget.node.url, widget.node.alt);
  }

  @override
  Widget build(BuildContext context) {
    return FutureBuilder<ImageProvider?>(
      future: _future,
      builder: (context, snapshot) {
        if (snapshot.connectionState != ConnectionState.done) {
          return const SizedBox.shrink();
        }
        final provider = snapshot.data;
        if (snapshot.hasError || provider == null) return _placeholder();
        return Image(
          image: provider,
          errorBuilder: (context, error, stack) => _placeholder(),
        );
      },
    );
  }

  Widget _placeholder() {
    final palette = widget.scope.palette;
    final alt = widget.node.alt;
    return Container(
      padding: const EdgeInsets.symmetric(horizontal: 8, vertical: 4),
      decoration: BoxDecoration(
        border: Border.all(color: palette.border),
        borderRadius: BorderRadius.circular(4),
      ),
      child: Row(
        mainAxisSize: MainAxisSize.min,
        children: [
          Icon(
            Icons.image_not_supported_outlined,
            size: 16,
            color: palette.quoteForeground,
          ),
          const SizedBox(width: 6),
          Flexible(
            child: Text(
              alt.isEmpty ? 'image' : alt,
              style: widget.scope.baseStyle.copyWith(
                color: palette.quoteForeground,
                fontStyle: FontStyle.italic,
              ),
            ),
          ),
        ],
      ),
    );
  }
}
