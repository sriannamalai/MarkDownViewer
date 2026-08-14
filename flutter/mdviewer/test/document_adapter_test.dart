import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:mdviewer/mdviewer.dart';

MdvTree tree(List<MdvBlock> blocks, {List<MdvBlock> footnotes = const []}) =>
    MdvTree(version: 1, blocks: blocks, footnotes: footnotes);

MdvParagraph para(String id, String text) => MdvParagraph(
  id: id,
  children: [MdvText(value: text)],
);

MdvFootnoteDef footnote(String id, int index, String text) => MdvFootnoteDef(
  id: id,
  index: index,
  refCount: 1,
  blocks: [para('$id-p', text)],
);

/// A paragraph whose span starts (and ends) at [line] — offsets are
/// irrelevant to line mapping, so they stay 0. `line: 0` builds the
/// zero span (the tree's spanless fallback shape).
MdvParagraph paraAt(String id, String text, int line) => MdvParagraph(
  id: id,
  span: MdvSpan(startLine: line, endLine: line, startOffset: 0, endOffset: 0),
  children: [MdvText(value: text)],
);

Widget harness(Widget child, {ThemeData? theme}) => MaterialApp(
  theme: theme,
  home: Scaffold(body: child),
);

/// Pumps a bare harness and hands back a [BuildContext] under the
/// [MaterialApp], for driving the adapter's context-taking members
/// directly.
Future<BuildContext> pumpContext(
  WidgetTester tester, {
  ThemeData? theme,
}) async {
  late BuildContext ctx;
  await tester.pumpWidget(
    harness(
      Builder(
        builder: (c) {
          ctx = c;
          return const SizedBox();
        },
      ),
      theme: theme,
    ),
  );
  return ctx;
}

/// Collects every span (depth-first) under all RichText widgets that
/// are descendants of [of].
List<InlineSpan> spansUnder(WidgetTester tester, Finder of) {
  final out = <InlineSpan>[];
  void walk(InlineSpan s) {
    out.add(s);
    if (s is TextSpan) {
      for (final c in s.children ?? const <InlineSpan>[]) {
        walk(c);
      }
    }
  }

  for (final rich in tester.widgetList<RichText>(
    find.descendant(of: of, matching: find.byType(RichText)),
  )) {
    walk(rich.text);
  }
  return out;
}

TextSpan leafSpan(WidgetTester tester, Finder of, String text) {
  return spansUnder(
    tester,
    of,
  ).whereType<TextSpan>().firstWhere((s) => s.text == text);
}

/// The full composition a host would assemble from the adapter:
/// `wrap(ListView.builder(...))`, built inside the widget tree so the
/// adapter resolves the ambient theme.
Widget composed(MdvDocumentAdapter adapter) => Builder(
  builder: (context) => adapter.wrap(
    context,
    ListView.builder(
      itemCount: adapter.itemCount,
      findChildIndexCallback: adapter.findChildIndexCallback,
      itemBuilder: adapter.itemBuilder,
    ),
  ),
);

void main() {
  group('itemCount', () {
    test('no footnotes: one item per top-level block', () {
      final adapter = MdvDocumentAdapter(
        tree([
          para('a000a000a000a000', 'one'),
          para('b000b000b000b000', 'two'),
        ]),
      );
      expect(adapter.itemCount, 2);
    });

    test('footnotes present: one trailing item is added', () {
      final adapter = MdvDocumentAdapter(
        tree(
          [para('a000a000a000a000', 'one')],
          footnotes: [footnote('f000f000f000f000', 1, 'note')],
        ),
      );
      expect(adapter.itemCount, 2);
    });

    test('empty tree: zero items', () {
      expect(MdvDocumentAdapter(tree([])).itemCount, 0);
    });
  });

  group('itemBuilder key scheme', () {
    testWidgets('blocks are keyed "<id>#<occurrence>"', (tester) async {
      final ctx = await pumpContext(tester);
      final adapter = MdvDocumentAdapter(
        tree([
          para('a000a000a000a000', 'one'),
          para('b000b000b000b000', 'two'),
        ]),
      );
      expect(
        adapter.itemBuilder(ctx, 0).key,
        const ValueKey('a000a000a000a000#0'),
      );
      expect(
        adapter.itemBuilder(ctx, 1).key,
        const ValueKey('b000b000b000b000#0'),
      );
    });

    testWidgets('duplicate ids get increasing occurrence suffixes', (
      tester,
    ) async {
      final ctx = await pumpContext(tester);
      final adapter = MdvDocumentAdapter(
        tree([
          para('deadbeefdeadbeef', 'same'),
          para('cafe000011110000', 'other'),
          para('deadbeefdeadbeef', 'same'),
          para('deadbeefdeadbeef', 'same'),
        ]),
      );
      expect(
        [
          for (var i = 0; i < adapter.itemCount; i++)
            adapter.itemBuilder(ctx, i).key,
        ],
        const [
          ValueKey('deadbeefdeadbeef#0'),
          ValueKey('cafe000011110000#0'),
          ValueKey('deadbeefdeadbeef#1'),
          ValueKey('deadbeefdeadbeef#2'),
        ],
      );
    });

    testWidgets('trailing footnotes item is keyed mdv-footnotes', (
      tester,
    ) async {
      final ctx = await pumpContext(tester);
      final adapter = MdvDocumentAdapter(
        tree(
          [para('a000a000a000a000', 'body')],
          footnotes: [footnote('f000f000f000f000', 1, 'note')],
        ),
      );
      expect(adapter.itemBuilder(ctx, 1).key, const ValueKey('mdv-footnotes'));
    });

    testWidgets('12px bottom inter-block padding, 0 on the last item', (
      tester,
    ) async {
      final ctx = await pumpContext(tester);
      EdgeInsetsGeometry paddingOf(Widget item) =>
          (((item as KeyedSubtree).child) as Padding).padding;

      final noFoot = MdvDocumentAdapter(
        tree([
          para('a000a000a000a000', 'one'),
          para('b000b000b000b000', 'two'),
        ]),
      );
      expect(
        paddingOf(noFoot.itemBuilder(ctx, 0)),
        const EdgeInsets.only(bottom: 12),
      );
      expect(paddingOf(noFoot.itemBuilder(ctx, 1)), EdgeInsets.zero);

      // With footnotes the LAST BLOCK is no longer the last item, so it
      // keeps its 12px spacing before the footnotes section.
      final withFoot = MdvDocumentAdapter(
        tree(
          [para('a000a000a000a000', 'one'), para('b000b000b000b000', 'two')],
          footnotes: [footnote('f000f000f000f000', 1, 'note')],
        ),
      );
      expect(
        paddingOf(withFoot.itemBuilder(ctx, 1)),
        const EdgeInsets.only(bottom: 12),
      );
    });
  });

  group('findChildIndexCallback', () {
    testWidgets('round-trips every itemBuilder key, injectively', (
      tester,
    ) async {
      final ctx = await pumpContext(tester);
      final adapter = MdvDocumentAdapter(
        tree(
          [
            para('deadbeefdeadbeef', 'same'),
            para('deadbeefdeadbeef', 'same'),
            para('cafe000011110000', 'other'),
          ],
          footnotes: [footnote('f000f000f000f000', 1, 'note')],
        ),
      );
      final indices = <int>{};
      for (var i = 0; i < adapter.itemCount; i++) {
        final key = adapter.itemBuilder(ctx, i).key!;
        final mapped = adapter.findChildIndexCallback(key);
        expect(mapped, i, reason: 'key $key must map back to index $i');
        indices.add(mapped!);
      }
      expect(indices, hasLength(adapter.itemCount)); // injective
    });

    test('unknown key maps to null', () {
      final adapter = MdvDocumentAdapter(tree([para('a000a000a000a000', 'x')]));
      expect(adapter.findChildIndexCallback(const ValueKey('nope#0')), isNull);
      expect(
        adapter.findChildIndexCallback(const ValueKey('mdv-footnotes')),
        isNull, // no footnotes in this tree
      );
    });
  });

  group('wrap', () {
    testWidgets('selectable=true wraps in SelectionArea; false does not', (
      tester,
    ) async {
      final selTree = tree([para('a000a000a000a000', 'sel')]);
      await tester.pumpWidget(harness(composed(MdvDocumentAdapter(selTree))));
      expect(find.byType(SelectionArea), findsOneWidget);

      await tester.pumpWidget(
        harness(composed(MdvDocumentAdapter(selTree, selectable: false))),
      );
      expect(find.byType(SelectionArea), findsNothing);
    });

    testWidgets('paints the palette background and applies the base style', (
      tester,
    ) async {
      await tester.pumpWidget(
        harness(
          composed(
            MdvDocumentAdapter(tree([para('a000a000a000a000', 'words')])),
          ),
        ),
      );
      final box = tester.widget<ColoredBox>(
        find
            .ancestor(
              of: find.byType(ListView),
              matching: find.byType(ColoredBox),
            )
            .first,
      );
      expect(box.color, MdvPalette.light.background);
      final span = leafSpan(tester, find.byType(ListView), 'words');
      expect(span.style?.color, MdvPalette.light.foreground);
      expect(span.style?.fontSize, 16);
      expect(span.style?.height, 1.5);
    });

    testWidgets('null palette follows the ambient theme brightness', (
      tester,
    ) async {
      await tester.pumpWidget(
        harness(
          composed(
            MdvDocumentAdapter(tree([para('a000a000a000a000', 'dark words')])),
          ),
          theme: ThemeData(brightness: Brightness.dark),
        ),
      );
      final span = leafSpan(tester, find.byType(ListView), 'dark words');
      expect(span.style?.color, MdvPalette.dark.foreground);
      final box = tester.widget<ColoredBox>(
        find
            .ancestor(
              of: find.byType(ListView),
              matching: find.byType(ColoredBox),
            )
            .first,
      );
      expect(box.color, MdvPalette.dark.background);
    });

    testWidgets('theme change re-resolves the palette (no stale cache)', (
      tester,
    ) async {
      final adapterTree = tree([para('a000a000a000a000', 'themed')]);
      final adapter = MdvDocumentAdapter(adapterTree);
      await tester.pumpWidget(
        harness(
          composed(adapter),
          theme: ThemeData(brightness: Brightness.light),
        ),
      );
      expect(
        leafSpan(tester, find.byType(ListView), 'themed').style?.color,
        MdvPalette.light.foreground,
      );
      // Same adapter instance, theme flipped to dark.
      await tester.pumpWidget(
        harness(
          composed(adapter),
          theme: ThemeData(brightness: Brightness.dark),
        ),
      );
      await tester.pumpAndSettle();
      expect(
        leafSpan(tester, find.byType(ListView), 'themed').style?.color,
        MdvPalette.dark.foreground,
      );
    });
  });

  group('baseStyle', () {
    testWidgets(
      'adapter: fontSize-only override keeps palette color and height',
      (tester) async {
        await tester.pumpWidget(
          harness(
            composed(
              MdvDocumentAdapter(
                tree([para('a000a000a000a000', 'scaled')]),
                baseStyle: const TextStyle(fontSize: 20),
              ),
            ),
          ),
        );
        final span = leafSpan(tester, find.byType(ListView), 'scaled');
        expect(span.style?.fontSize, 20);
        expect(span.style?.color, MdvPalette.light.foreground);
        expect(span.style?.height, 1.5);
      },
    );

    testWidgets('view: fontSize-only override keeps palette color and height', (
      tester,
    ) async {
      await tester.pumpWidget(
        harness(
          MdvDocumentView(
            tree([para('a000a000a000a000', 'scaled')]),
            baseStyle: const TextStyle(fontSize: 20),
          ),
        ),
      );
      final span = leafSpan(tester, find.byType(MdvDocumentView), 'scaled');
      expect(span.style?.fontSize, 20);
      expect(span.style?.color, MdvPalette.light.foreground);
      expect(span.style?.height, 1.5);
    });

    testWidgets('baseStyle wins over the default where both set a field', (
      tester,
    ) async {
      await tester.pumpWidget(
        harness(
          composed(
            MdvDocumentAdapter(
              tree([para('a000a000a000a000', 'tinted')]),
              baseStyle: const TextStyle(color: Color(0xFF112233)),
            ),
          ),
        ),
      );
      final span = leafSpan(tester, find.byType(ListView), 'tinted');
      expect(span.style?.color, const Color(0xFF112233));
      expect(span.style?.fontSize, 16); // default retained
    });
  });

  group('view-on-adapter equivalence', () {
    testWidgets('the view renders exactly the adapter item keys, in order', (
      tester,
    ) async {
      final docTree = tree(
        [
          para('deadbeefdeadbeef', 'same'),
          para('cafe000011110000', 'other'),
          para('deadbeefdeadbeef', 'same'),
        ],
        footnotes: [footnote('f000f000f000f000', 1, 'note')],
      );
      await tester.pumpWidget(
        harness(MdvDocumentView(docTree, selectable: false)),
      );
      final rendered = tester
          .widgetList<KeyedSubtree>(
            find.descendant(
              of: find.byType(ListView),
              matching: find.byType(KeyedSubtree),
            ),
          )
          .map((w) => w.key)
          .whereType<ValueKey<String>>()
          .toList();

      final adapter = MdvDocumentAdapter(docTree, selectable: false);
      late BuildContext ctx;
      await tester.pumpWidget(
        harness(
          Builder(
            builder: (c) {
              ctx = c;
              return const SizedBox();
            },
          ),
        ),
      );
      final expected = [
        for (var i = 0; i < adapter.itemCount; i++)
          adapter.itemBuilder(ctx, i).key,
      ];
      expect(rendered, expected);
      expect(rendered.last, const ValueKey('mdv-footnotes'));
    });
  });

  group('itemBuilder bounds', () {
    testWidgets('out-of-range index asserts instead of rendering a phantom '
        'footnotes section', (tester) async {
      final ctx = await pumpContext(tester);
      final noFoot = MdvDocumentAdapter(tree([para('a000a000a000a000', 'x')]));
      expect(() => noFoot.itemBuilder(ctx, 1), throwsAssertionError);
      expect(() => noFoot.itemBuilder(ctx, -1), throwsAssertionError);

      final withFoot = MdvDocumentAdapter(
        tree(
          [para('a000a000a000a000', 'x')],
          footnotes: [footnote('f000f000f000f000', 1, 'note')],
        ),
      );
      expect(() => withFoot.itemBuilder(ctx, 2), throwsAssertionError);
    });
  });

  group('blockIndexForLine', () {
    test('exact hit: a line equal to a block startLine maps to that block', () {
      final adapter = MdvDocumentAdapter(
        tree([
          paraAt('a000a000a000a000', 'one', 1),
          paraAt('b000b000b000b000', 'two', 5),
          paraAt('c000c000c000c000', 'three', 9),
        ]),
      );
      expect(adapter.blockIndexForLine(1), 0);
      expect(adapter.blockIndexForLine(5), 1);
      expect(adapter.blockIndexForLine(9), 2);
    });

    test('between blocks: nearest PRECEDING block wins', () {
      final adapter = MdvDocumentAdapter(
        tree([
          paraAt('a000a000a000a000', 'one', 1),
          paraAt('b000b000b000b000', 'two', 5),
          paraAt('c000c000c000c000', 'three', 9),
        ]),
      );
      expect(adapter.blockIndexForLine(4), 0);
      expect(adapter.blockIndexForLine(7), 1);
      expect(adapter.blockIndexForLine(999), 2); // past the last block
    });

    test('line before the first spanned block maps to null', () {
      final adapter = MdvDocumentAdapter(
        tree([
          paraAt('a000a000a000a000', 'one', 3),
          paraAt('b000b000b000b000', 'two', 7),
        ]),
      );
      expect(adapter.blockIndexForLine(1), isNull);
      expect(adapter.blockIndexForLine(2), isNull);
    });

    test('fully spanless tree (null spans) maps every line to null', () {
      final adapter = MdvDocumentAdapter(
        tree([
          para('a000a000a000a000', 'one'),
          para('b000b000b000b000', 'two'),
        ]),
      );
      expect(adapter.blockIndexForLine(0), isNull);
      expect(adapter.blockIndexForLine(1), isNull);
      expect(adapter.blockIndexForLine(999), isNull);
    });

    test('zero-span and spanless blocks interleaved are skipped, never '
        'treated as line 0', () {
      final adapter = MdvDocumentAdapter(
        tree([
          paraAt('a000a000a000a000', 'spanned', 2),
          paraAt('b000b000b000b000', 'zero', 0), // zero span
          para('c000c000c000c000', 'null span'),
          paraAt('d000d000d000d000', 'spanned too', 8),
        ]),
      );
      // Lines inside the gap resolve to the spanned block at line 2 —
      // never to the zero/null-span blocks that follow it.
      expect(adapter.blockIndexForLine(2), 0);
      expect(adapter.blockIndexForLine(5), 0);
      expect(adapter.blockIndexForLine(7), 0);
      expect(adapter.blockIndexForLine(8), 3);
      // A line before every spanned block is null even though the
      // zero-span block "starts at 0 <= 1" numerically.
      expect(adapter.blockIndexForLine(1), isNull);
    });

    test('single-block doc', () {
      final adapter = MdvDocumentAdapter(
        tree([paraAt('a000a000a000a000', 'only', 4)]),
      );
      expect(adapter.blockIndexForLine(3), isNull);
      expect(adapter.blockIndexForLine(4), 0);
      expect(adapter.blockIndexForLine(400), 0);
    });
  });

  group('startLineForIndex', () {
    test('returns each spanned block startLine', () {
      final adapter = MdvDocumentAdapter(
        tree([
          paraAt('a000a000a000a000', 'one', 1),
          paraAt('b000b000b000b000', 'two', 6),
        ]),
      );
      expect(adapter.startLineForIndex(0), 1);
      expect(adapter.startLineForIndex(1), 6);
    });

    test('footnotes item index maps to null', () {
      final adapter = MdvDocumentAdapter(
        tree(
          [paraAt('a000a000a000a000', 'body', 1)],
          footnotes: [footnote('f000f000f000f000', 1, 'note')],
        ),
      );
      expect(adapter.itemCount, 2);
      expect(adapter.startLineForIndex(1), isNull); // the footnotes item
    });

    test('negative and out-of-range indices map to null', () {
      final adapter = MdvDocumentAdapter(
        tree([paraAt('a000a000a000a000', 'only', 1)]),
      );
      expect(adapter.startLineForIndex(-1), isNull);
      expect(adapter.startLineForIndex(1), isNull);
      expect(adapter.startLineForIndex(999), isNull);
    });

    test('zero and null spans map to null', () {
      final adapter = MdvDocumentAdapter(
        tree([
          paraAt('a000a000a000a000', 'zero', 0),
          para('b000b000b000b000', 'null span'),
        ]),
      );
      expect(adapter.startLineForIndex(0), isNull);
      expect(adapter.startLineForIndex(1), isNull);
    });
  });

  group('line mapping over a real parse', () {
    final mdv = Mdviewer.instance;

    test('renderTree spans drive both directions of the mapping', () {
      final parsed = mdv.renderTree('# Title\n\npara one\n\npara two\n');
      final adapter = MdvDocumentAdapter(parsed);
      expect(parsed.blocks, hasLength(3));

      // The parse annotates real 1-based start lines.
      expect(
        [for (var i = 0; i < 3; i++) adapter.startLineForIndex(i)],
        [1, 3, 5],
      );

      // Exact hits and nearest-preceding, against the parsed spans.
      expect(adapter.blockIndexForLine(1), 0);
      expect(adapter.blockIndexForLine(2), 0);
      expect(adapter.blockIndexForLine(3), 1);
      expect(adapter.blockIndexForLine(4), 1);
      expect(adapter.blockIndexForLine(5), 2);
      expect(adapter.blockIndexForLine(999), 2);

      // Round trip: every block's own startLine maps back to it.
      for (var i = 0; i < parsed.blocks.length; i++) {
        expect(adapter.blockIndexForLine(adapter.startLineForIndex(i)!), i);
      }
    });

    test('real footnotes: the trailing item has no start line', () {
      final parsed = mdv.renderTree('body.[^1]\n\n[^1]: The note.\n');
      final adapter = MdvDocumentAdapter(parsed);
      expect(parsed.footnotes, isNotEmpty);
      expect(adapter.itemCount, parsed.blocks.length + 1);
      expect(adapter.startLineForIndex(parsed.blocks.length), isNull);
    });
  });
}
