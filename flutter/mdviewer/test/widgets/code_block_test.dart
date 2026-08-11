import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:mdviewer/mdviewer.dart';

MdvTree tree(List<MdvBlock> blocks) =>
    MdvTree(version: 1, blocks: blocks, footnotes: const []);

Widget harness(Widget child) => MaterialApp(home: Scaffold(body: child));

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
  final pinned = MdvPalette.light.copyWith(
    tokenColors: {
      'Keyword': const Color(0xFFCF222E),
      'LiteralString': const Color(0xFF0A3069),
    },
  );

  const block = MdvCodeBlock(
    id: 'c',
    language: 'go',
    label: 'go',
    text: 'func x() { s := "hi" }\n',
    runs: [
      MdvTokenRun(text: 'func', tokenType: 'Keyword'),
      MdvTokenRun(text: ' x() { s := ', tokenType: 'Text'),
      MdvTokenRun(text: '"hi"', tokenType: 'LiteralString'),
      MdvTokenRun(text: ' }\n', tokenType: 'Text'),
    ],
  );

  testWidgets('token runs render as colored spans via the palette', (
    tester,
  ) async {
    await tester.pumpWidget(
      harness(MdvDocumentView(tree([block]), palette: pinned)),
    );
    final spans = leafSpans(tester);
    final kw = spans.firstWhere((s) => s.text == 'func');
    expect(kw.style?.color, const Color(0xFFCF222E));
    final str = spans.firstWhere((s) => s.text == '"hi"');
    expect(str.style?.color, const Color(0xFF0A3069));
    // Unknown token type ('Text' is not in the pinned palette) falls
    // back to the default foreground.
    final plain = spans.firstWhere((s) => s.text == ' x() { s := ');
    expect(plain.style?.color, pinned.foreground);
    // Language label from the block, horizontal scroll for the pane.
    expect(find.text('go'), findsOneWidget);
    expect(
      find.descendant(
        of: find.byType(MdvCodeBlockView),
        matching: find.byWidgetPredicate(
          (w) =>
              w is SingleChildScrollView &&
              w.scrollDirection == Axis.horizontal,
        ),
      ),
      findsOneWidget,
    );
  });

  testWidgets('null runs render plain monospace text', (tester) async {
    const plainBlock = MdvCodeBlock(
      id: 'c',
      language: '',
      label: 'code',
      text: 'plain text code\n',
      runs: null,
    );
    await tester.pumpWidget(
      harness(MdvDocumentView(tree([plainBlock]), palette: pinned)),
    );
    expect(find.text('code'), findsOneWidget); // unlabeled display label
    final span = leafSpans(
      tester,
    ).firstWhere((s) => s.text == 'plain text code');
    expect(span.style?.fontFamily, 'monospace');
  });

  testWidgets(
    'copy button: native Clipboard.setData with full text, "Copied" flip',
    (tester) async {
      final calls = <MethodCall>[];
      tester.binding.defaultBinaryMessenger.setMockMethodCallHandler(
        SystemChannels.platform,
        (call) async {
          calls.add(call);
          return null;
        },
      );
      addTearDown(
        () => tester.binding.defaultBinaryMessenger.setMockMethodCallHandler(
          SystemChannels.platform,
          null,
        ),
      );

      await tester.pumpWidget(
        harness(MdvDocumentView(tree([block]), palette: pinned)),
      );
      expect(find.text('Copy'), findsOneWidget);
      expect(find.text('Copied'), findsNothing);

      await tester.tap(find.text('Copy'));
      await tester.pump();

      final setData = calls.where((c) => c.method == 'Clipboard.setData');
      expect(setData, hasLength(1));
      expect(
        (setData.single.arguments as Map)['text'],
        'func x() { s := "hi" }\n', // the FULL literal text, newline included
      );
      expect(find.text('Copied'), findsOneWidget);
      expect(find.text('Copy'), findsNothing);

      // Reverts after the feedback window.
      await tester.pump(const Duration(seconds: 3));
      expect(find.text('Copy'), findsOneWidget);
      expect(find.text('Copied'), findsNothing);
    },
  );
}
