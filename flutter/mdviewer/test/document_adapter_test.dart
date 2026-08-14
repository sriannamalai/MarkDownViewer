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
}
