import 'dart:typed_data';

import 'package:flutter/gestures.dart';
import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:mdviewer/mdviewer.dart';

MdvTree tree(List<MdvBlock> blocks) =>
    MdvTree(version: 1, blocks: blocks, footnotes: const []);

Widget harness(Widget child) => MaterialApp(home: Scaffold(body: child));

/// A valid 1x1 transparent PNG.
final Uint8List kTransparentPng = Uint8List.fromList(const [
  0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A, 0x00, 0x00, 0x00, 0x0D, //
  0x49, 0x48, 0x44, 0x52, 0x00, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00, 0x01, //
  0x08, 0x06, 0x00, 0x00, 0x00, 0x1F, 0x15, 0xC4, 0x89, 0x00, 0x00, 0x00, //
  0x0A, 0x49, 0x44, 0x41, 0x54, 0x78, 0x9C, 0x63, 0x00, 0x01, 0x00, 0x00, //
  0x05, 0x00, 0x01, 0x0D, 0x0A, 0x2D, 0xB4, 0x00, 0x00, 0x00, 0x00, 0x49, //
  0x45, 0x4E, 0x44, 0xAE, 0x42, 0x60, 0x82,
]);

List<TextSpan> leafSpans(WidgetTester tester) {
  final out = <TextSpan>[];
  void walk(InlineSpan s) {
    if (s is TextSpan) {
      if (s.text != null) out.add(s);
      for (final c in s.children ?? const <InlineSpan>[]) {
        walk(c);
      }
    }
  }

  for (final rich in tester.widgetList<RichText>(find.byType(RichText))) {
    walk(rich.text);
  }
  return out;
}

void main() {
  group('links', () {
    testWidgets('tap fires onLinkTap with url and source kind', (tester) async {
      final taps = <(String, bool, String?)>[];
      await tester.pumpWidget(
        harness(
          MdvDocumentView(
            tree([
              MdvParagraph(
                id: 'p',
                children: const [
                  MdvLink(
                    url: 'https://example.com/x',
                    source: 'wikiLink',
                    children: [MdvText(value: 'go here')],
                  ),
                ],
              ),
            ]),
            onLinkTap: (url, blocked, source) =>
                taps.add((url, blocked, source)),
          ),
        ),
      );
      final span = leafSpans(tester).firstWhere((s) => s.text == 'go here');
      expect(span.style?.color, MdvPalette.light.accent);
      expect(span.style?.decoration, TextDecoration.underline);
      expect(span.recognizer, isA<TapGestureRecognizer>());
      (span.recognizer as TapGestureRecognizer).onTap!();
      expect(taps, [('https://example.com/x', false, 'wikiLink')]);
    });

    testWidgets('nested emphasis inside a link still carries the tap', (
      tester,
    ) async {
      final taps = <String>[];
      await tester.pumpWidget(
        harness(
          MdvDocumentView(
            tree([
              MdvParagraph(
                id: 'p',
                children: const [
                  MdvLink(
                    url: 'https://example.com',
                    children: [
                      MdvEmphasis(children: [MdvText(value: 'fancy link')]),
                    ],
                  ),
                ],
              ),
            ]),
            onLinkTap: (url, blocked, source) => taps.add(url),
          ),
        ),
      );
      final span = leafSpans(tester).firstWhere((s) => s.text == 'fancy link');
      expect(span.style?.fontStyle, FontStyle.italic);
      expect(span.recognizer, isA<TapGestureRecognizer>());
      (span.recognizer as TapGestureRecognizer).onTap!();
      expect(taps, ['https://example.com']);
    });

    testWidgets('blocked link renders muted with NO tap recognizer', (
      tester,
    ) async {
      final taps = <String>[];
      await tester.pumpWidget(
        harness(
          MdvDocumentView(
            tree([
              MdvParagraph(
                id: 'p',
                children: const [
                  MdvLink(
                    url: '',
                    blocked: true,
                    children: [MdvText(value: 'evil link')],
                  ),
                ],
              ),
            ]),
            onLinkTap: (url, blocked, source) => taps.add(url),
          ),
        ),
      );
      final span = leafSpans(tester).firstWhere((s) => s.text == 'evil link');
      expect(span.style?.color, MdvPalette.light.quoteForeground);
      expect(span.recognizer, isNull);
      // A tap on the text goes nowhere.
      await tester.tap(find.text('evil link', findRichText: true));
      await tester.pump();
      expect(taps, isEmpty);
    });

    testWidgets('no onLinkTap callback: link styled but inert', (tester) async {
      await tester.pumpWidget(
        harness(
          MdvDocumentView(
            tree([
              MdvParagraph(
                id: 'p',
                children: const [
                  MdvLink(
                    url: 'https://example.com',
                    children: [MdvText(value: 'plain link')],
                  ),
                ],
              ),
            ]),
          ),
        ),
      );
      final span = leafSpans(tester).firstWhere((s) => s.text == 'plain link');
      expect(span.recognizer, isNull);
    });
  });

  group('images', () {
    const image = MdvParagraph(
      id: 'p',
      children: [MdvImage(url: 'logo.png', alt: 'the logo')],
    );

    testWidgets('no imageProvider: alt-text placeholder', (tester) async {
      await tester.pumpWidget(harness(MdvDocumentView(tree([image]))));
      await tester.pump();
      expect(find.text('the logo'), findsOneWidget);
      expect(find.byType(Image), findsNothing);
    });

    testWidgets('imageProvider returning null: alt-text placeholder', (
      tester,
    ) async {
      final requested = <(String, String)>[];
      await tester.pumpWidget(
        harness(
          MdvDocumentView(
            tree([image]),
            imageProvider: (url, alt) async {
              requested.add((url, alt));
              return null;
            },
          ),
        ),
      );
      await tester.pump();
      expect(requested, [('logo.png', 'the logo')]);
      expect(find.text('the logo'), findsOneWidget);
      expect(find.byType(Image), findsNothing);
    });

    testWidgets('imageProvider error: alt-text placeholder, no crash', (
      tester,
    ) async {
      await tester.pumpWidget(
        harness(
          MdvDocumentView(
            tree([image]),
            imageProvider: (url, alt) async => throw StateError('boom'),
          ),
        ),
      );
      await tester.pump();
      expect(tester.takeException(), isNull);
      expect(find.text('the logo'), findsOneWidget);
    });

    testWidgets('imageProvider returning a provider renders the image', (
      tester,
    ) async {
      final provider = MemoryImage(kTransparentPng);
      await tester.pumpWidget(
        harness(
          MdvDocumentView(
            tree([image]),
            imageProvider: (url, alt) async => provider,
          ),
        ),
      );
      await tester.pump();
      await tester.pump();
      final img = tester.widget<Image>(find.byType(Image));
      expect(img.image, same(provider));
    });

    testWidgets('blocked image: placeholder, host callback NOT invoked', (
      tester,
    ) async {
      var called = false;
      await tester.pumpWidget(
        harness(
          MdvDocumentView(
            tree([
              const MdvParagraph(
                id: 'p',
                children: [
                  MdvImage(url: '', blocked: true, alt: 'blocked pic'),
                ],
              ),
            ]),
            imageProvider: (url, alt) async {
              called = true;
              return MemoryImage(kTransparentPng);
            },
          ),
        ),
      );
      await tester.pump();
      expect(called, isFalse);
      expect(find.text('blocked pic'), findsOneWidget);
    });

    testWidgets('empty alt falls back to a generic placeholder label', (
      tester,
    ) async {
      await tester.pumpWidget(
        harness(
          MdvDocumentView(
            tree([
              const MdvParagraph(
                id: 'p',
                children: [MdvImage(url: 'x.png', alt: '')],
              ),
            ]),
          ),
        ),
      );
      await tester.pump();
      expect(find.text('image'), findsOneWidget);
    });
  });

  group('misc inlines', () {
    testWidgets('hard/soft breaks and htmlInline render as text', (
      tester,
    ) async {
      await tester.pumpWidget(
        harness(
          MdvDocumentView(
            tree([
              const MdvParagraph(
                id: 'p',
                children: [
                  MdvText(value: 'a'),
                  MdvSoftBreak(),
                  MdvText(value: 'b'),
                  MdvHardBreak(),
                  MdvHtmlInline(html: '<kbd>K</kbd>', unsafe: false),
                ],
              ),
            ]),
          ),
        ),
      );
      final spans = leafSpans(tester);
      expect(spans.any((s) => s.text == ' '), isTrue); // soft break
      expect(spans.any((s) => s.text == '\n'), isTrue); // hard break
      final html = spans.firstWhere((s) => s.text == '<kbd>K</kbd>');
      expect(html.style?.fontFamily, 'monospace'); // shown as source, inert
    });

    testWidgets('unknown inline renders nothing, no crash', (tester) async {
      await tester.pumpWidget(
        harness(
          MdvDocumentView(
            tree([
              const MdvParagraph(
                id: 'p',
                children: [
                  MdvText(value: 'kept'),
                  MdvUnknownInline(kind: 'sparkles', raw: {}),
                ],
              ),
            ]),
          ),
        ),
      );
      expect(tester.takeException(), isNull);
      expect(find.textContaining('kept', findRichText: true), findsOneWidget);
    });
  });
}
