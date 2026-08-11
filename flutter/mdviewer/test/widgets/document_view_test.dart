import 'package:flutter/material.dart';
import 'package:flutter_math_fork/flutter_math.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:mdviewer/mdviewer.dart';

MdvTree tree(List<MdvBlock> blocks, {List<MdvBlock> footnotes = const []}) =>
    MdvTree(version: 1, blocks: blocks, footnotes: footnotes);

MdvParagraph para(String id, String text) => MdvParagraph(
  id: id,
  children: [MdvText(value: text)],
);

Widget harness(Widget child, {ThemeData? theme}) => MaterialApp(
  theme: theme,
  home: Scaffold(body: child),
);

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

void main() {
  group('MdvDocumentView per-block-kind rendering', () {
    testWidgets('heading renders scaled bold text', (tester) async {
      await tester.pumpWidget(
        harness(
          MdvDocumentView(
            tree([
              MdvHeading(
                id: 'h1',
                level: 1,
                children: const [MdvText(value: 'Title')],
              ),
              MdvHeading(
                id: 'h3',
                level: 3,
                children: const [MdvText(value: 'Sub')],
              ),
            ]),
          ),
        ),
      );
      expect(find.text('Title', findRichText: true), findsOneWidget);
      final title = leafSpan(tester, find.byType(MdvDocumentView), 'Title');
      expect(title.style?.fontSize, 16 * 1.75);
      expect(title.style?.fontWeight, FontWeight.w700);
      final sub = leafSpan(tester, find.byType(MdvDocumentView), 'Sub');
      expect(sub.style?.fontSize, 16 * 1.25);
    });

    testWidgets('paragraph with emphasis/strong/strike/code spans', (
      tester,
    ) async {
      await tester.pumpWidget(
        harness(
          MdvDocumentView(
            tree([
              MdvParagraph(
                id: 'p',
                children: const [
                  MdvEmphasis(children: [MdvText(value: 'it')]),
                  MdvStrong(children: [MdvText(value: 'bold')]),
                  MdvStrikethrough(children: [MdvText(value: 'gone')]),
                  MdvCodeSpan(value: 'x=1'),
                ],
              ),
            ]),
          ),
        ),
      );
      final root = find.byType(MdvDocumentView);
      expect(leafSpan(tester, root, 'it').style?.fontStyle, FontStyle.italic);
      expect(leafSpan(tester, root, 'bold').style?.fontWeight, FontWeight.bold);
      expect(
        leafSpan(tester, root, 'gone').style?.decoration,
        TextDecoration.lineThrough,
      );
      final code = leafSpan(tester, root, 'x=1');
      expect(code.style?.fontFamily, 'monospace');
      expect(code.style?.backgroundColor, MdvPalette.light.codeBackground);
    });

    testWidgets('blockquote gets a left bar and muted text', (tester) async {
      await tester.pumpWidget(
        harness(
          MdvDocumentView(
            tree([
              MdvBlockQuote(id: 'q', blocks: [para('p', 'quoted words')]),
            ]),
          ),
        ),
      );
      expect(find.text('quoted words', findRichText: true), findsOneWidget);
      final span = leafSpan(
        tester,
        find.byType(MdvDocumentView),
        'quoted words',
      );
      expect(span.style?.color, MdvPalette.light.quoteForeground);
    });

    testWidgets('admonition renders tinted panel with derived title', (
      tester,
    ) async {
      await tester.pumpWidget(
        harness(
          MdvDocumentView(
            tree([
              MdvAdmonition(
                id: 'a',
                variant: 'warning',
                title: 'Warning',
                blocks: [para('p', 'careful now')],
              ),
            ]),
          ),
        ),
      );
      expect(find.text('Warning'), findsOneWidget);
      expect(find.text('careful now', findRichText: true), findsOneWidget);
      final title = tester.widget<Text>(find.text('Warning'));
      expect(
        title.style?.color,
        mdvAdmonitionColor('warning', Brightness.light),
      );
    });

    testWidgets('lists: bullets, ordered start, read-only task checkboxes', (
      tester,
    ) async {
      await tester.pumpWidget(
        harness(
          MdvDocumentView(
            tree([
              MdvList(
                id: 'ul',
                ordered: false,
                start: 1,
                tight: true,
                items: [
                  MdvListItem(
                    id: 'i1',
                    task: null,
                    blocks: [para('p1', 'one')],
                  ),
                  MdvListItem(
                    id: 'i2',
                    task: true,
                    blocks: [para('p2', 'done')],
                  ),
                  MdvListItem(
                    id: 'i3',
                    task: false,
                    blocks: [para('p3', 'todo')],
                  ),
                ],
              ),
              MdvList(
                id: 'ol',
                ordered: true,
                start: 4,
                tight: true,
                items: [
                  MdvListItem(
                    id: 'i4',
                    task: null,
                    blocks: [para('p4', 'four')],
                  ),
                  MdvListItem(
                    id: 'i5',
                    task: null,
                    blocks: [para('p5', 'five')],
                  ),
                ],
              ),
            ]),
          ),
        ),
      );
      expect(find.text('•', findRichText: true), findsOneWidget);
      expect(find.byIcon(Icons.check_box), findsOneWidget);
      expect(find.byIcon(Icons.check_box_outline_blank), findsOneWidget);
      // Checkboxes are icons, not interactive Checkbox widgets.
      expect(find.byType(Checkbox), findsNothing);
      // Ordered list numbering honors start.
      expect(find.text('4.', findRichText: true), findsOneWidget);
      expect(find.text('5.', findRichText: true), findsOneWidget);
    });

    testWidgets('nested list content indents inside the parent item', (
      tester,
    ) async {
      await tester.pumpWidget(
        harness(
          MdvDocumentView(
            tree([
              MdvList(
                id: 'outer',
                ordered: false,
                start: 1,
                tight: true,
                items: [
                  MdvListItem(
                    id: 'i1',
                    task: null,
                    blocks: [
                      para('p1', 'outer item'),
                      MdvList(
                        id: 'inner',
                        ordered: false,
                        start: 1,
                        tight: true,
                        items: [
                          MdvListItem(
                            id: 'i2',
                            task: null,
                            blocks: [para('p2', 'inner item')],
                          ),
                        ],
                      ),
                    ],
                  ),
                ],
              ),
            ]),
          ),
        ),
      );
      final outer = tester.getTopLeft(
        find.text('outer item', findRichText: true),
      );
      final inner = tester.getTopLeft(
        find.text('inner item', findRichText: true),
      );
      expect(inner.dx, greaterThan(outer.dx));
    });

    testWidgets('table honors alignments and styles the header', (
      tester,
    ) async {
      await tester.pumpWidget(
        harness(
          MdvDocumentView(
            tree([
              MdvTable(
                id: 't',
                alignments: const [
                  MdvAlignment.left,
                  MdvAlignment.center,
                  MdvAlignment.right,
                ],
                header: const [
                  [MdvText(value: 'A')],
                  [MdvText(value: 'B')],
                  [MdvText(value: 'C')],
                ],
                rows: const [
                  [
                    [MdvText(value: 'a1')],
                    [MdvText(value: 'b1')],
                    [MdvText(value: 'c1')],
                  ],
                ],
              ),
            ]),
          ),
        ),
      );
      expect(find.byType(Table), findsOneWidget);
      final header = leafSpan(tester, find.byType(Table), 'A');
      expect(header.style?.fontWeight, FontWeight.w600);
      final aligns = tester
          .widgetList<Align>(
            find.descendant(
              of: find.byType(Table),
              matching: find.byType(Align),
            ),
          )
          .map((a) => a.alignment)
          .toList();
      expect(aligns, contains(Alignment.topCenter));
      expect(aligns, contains(Alignment.topRight));
    });

    testWidgets('thematic break renders a divider', (tester) async {
      await tester.pumpWidget(
        harness(MdvDocumentView(tree([MdvThematicBreak(id: 'hr')]))),
      );
      expect(find.byType(Divider), findsOneWidget);
    });

    testWidgets('diagram default builder is a placeholder with mono source', (
      tester,
    ) async {
      await tester.pumpWidget(
        harness(
          MdvDocumentView(
            tree([
              MdvDiagram(id: 'd', source: 'graph TD; A-->B', engine: 'mermaid'),
            ]),
          ),
        ),
      );
      expect(find.text('diagram · mermaid'), findsOneWidget);
      expect(find.text('graph TD; A-->B'), findsOneWidget);
    });

    testWidgets('diagram builder override replaces the placeholder', (
      tester,
    ) async {
      await tester.pumpWidget(
        harness(
          MdvDocumentView(
            tree([MdvDiagram(id: 'd', source: 'graph TD', engine: 'mermaid')]),
            builders: MdvBuilders(
              diagram: (context, node, defaultChild) =>
                  Text('custom:${node.engine}'),
            ),
          ),
        ),
      );
      expect(find.text('custom:mermaid'), findsOneWidget);
      expect(find.text('diagram · mermaid'), findsNothing);
    });

    testWidgets('sanitized htmlBlock shows tag-stripped plain text', (
      tester,
    ) async {
      await tester.pumpWidget(
        harness(
          MdvDocumentView(
            tree([
              MdvHtmlBlock(
                id: 'h',
                html: '<p>hello <b>world</b> &amp; co</p>',
                unsafe: false,
              ),
            ]),
          ),
        ),
      );
      expect(find.text('hello world & co'), findsOneWidget);
    });

    testWidgets('unsafe htmlBlock is an inert disclosure, never rendered', (
      tester,
    ) async {
      const raw = '<script>alert(1)</script>';
      await tester.pumpWidget(
        harness(
          MdvDocumentView(
            tree([MdvHtmlBlock(id: 'h', html: raw, unsafe: true)]),
            selectable: false,
          ),
        ),
      );
      expect(find.text('Raw HTML (not rendered)'), findsOneWidget);
      expect(find.text(raw), findsNothing); // collapsed by default
      await tester.tap(find.text('Raw HTML (not rendered)'));
      await tester.pump();
      expect(find.text(raw), findsOneWidget); // source shown, as text only
    });

    testWidgets('definition list: bold terms, indented descriptions', (
      tester,
    ) async {
      await tester.pumpWidget(
        harness(
          MdvDocumentView(
            tree([
              MdvDefinitionList(
                id: 'dl',
                blocks: [
                  const MdvDefinitionTerm(
                    id: 'dt',
                    children: [MdvText(value: 'Term')],
                  ),
                  MdvDefinitionDesc(id: 'dd', blocks: [para('p', 'Meaning')]),
                ],
              ),
            ]),
          ),
        ),
      );
      final root = find.byType(MdvDocumentView);
      expect(leafSpan(tester, root, 'Term').style?.fontWeight, FontWeight.bold);
      final term = tester.getTopLeft(find.text('Term', findRichText: true));
      final desc = tester.getTopLeft(find.text('Meaning', findRichText: true));
      expect(desc.dx, greaterThan(term.dx));
    });

    testWidgets('footnotes: superscript refs, definitions listed at the end', (
      tester,
    ) async {
      await tester.pumpWidget(
        harness(
          MdvDocumentView(
            tree(
              [
                MdvParagraph(
                  id: 'p',
                  children: const [
                    MdvText(value: 'body'),
                    MdvFootnoteRef(index: 1),
                  ],
                ),
              ],
              footnotes: [
                MdvFootnoteDef(
                  id: 'fn',
                  index: 1,
                  refCount: 1,
                  blocks: [para('fp', 'the note')],
                ),
              ],
            ),
          ),
        ),
      );
      expect(find.text('1'), findsOneWidget); // superscript ref
      expect(find.text('1.'), findsOneWidget); // definition marker
      expect(find.text('the note', findRichText: true), findsOneWidget);
      expect(find.byType(Divider), findsOneWidget); // section rule
    });

    testWidgets('unknown block renders nothing and does not crash', (
      tester,
    ) async {
      await tester.pumpWidget(
        harness(
          MdvDocumentView(
            tree([
              const MdvUnknownBlock(id: 'u', kind: 'holoDeck', raw: {}),
              para('p', 'still here'),
            ]),
          ),
        ),
      );
      expect(tester.takeException(), isNull);
      expect(find.text('still here', findRichText: true), findsOneWidget);
    });

    testWidgets('builder override wraps the default child', (tester) async {
      await tester.pumpWidget(
        harness(
          MdvDocumentView(
            tree([para('p', 'inner')]),
            builders: MdvBuilders(
              paragraph: (context, node, defaultChild) => ColoredBox(
                color: const Color(0xFF00FF00),
                child: defaultChild,
              ),
            ),
          ),
        ),
      );
      expect(find.text('inner', findRichText: true), findsOneWidget);
      expect(
        find.ancestor(
          of: find.text('inner', findRichText: true),
          matching: find.byWidgetPredicate(
            (w) => w is ColoredBox && w.color == const Color(0xFF00FF00),
          ),
        ),
        findsOneWidget,
      );
    });
  });

  group('keying / identity', () {
    testWidgets(
      'two byte-identical blocks (shared id) render without key collision',
      (tester) async {
        // The duplicate-id contract: content-hash ids mean byte-identical
        // blocks SHARE an id by design. Keyed by bare id this ListView
        // would throw "Duplicate keys found"; (id, occurrenceIndex)
        // keying must render both.
        await tester.pumpWidget(
          harness(
            MdvDocumentView(
              tree([
                para('deadbeefdeadbeef', 'same text'),
                para('deadbeefdeadbeef', 'same text'),
                para('deadbeefdeadbeef', 'same text'),
              ]),
            ),
          ),
        );
        expect(tester.takeException(), isNull);
        expect(find.text('same text', findRichText: true), findsNWidgets(3));
        expect(
          find.byKey(const ValueKey('deadbeefdeadbeef#0')),
          findsOneWidget,
        );
        expect(
          find.byKey(const ValueKey('deadbeefdeadbeef#1')),
          findsOneWidget,
        );
        expect(
          find.byKey(const ValueKey('deadbeefdeadbeef#2')),
          findsOneWidget,
        );
      },
    );

    testWidgets('block STATE survives a block being inserted above it '
        '(keys drive reconciliation via findChildIndexCallback)', (
      tester,
    ) async {
      // A stateful block: the unsafe raw-HTML disclosure, whose
      // open/closed flag lives in its State. ListView.builder
      // reconciles children BY INDEX unless findChildIndexCallback
      // maps keys back to their new indices — without it, inserting
      // a block above tears down and rebuilds everything below
      // (collapsing the disclosure) despite the stable keys.
      const raw = '<x-widget data="1">';
      const html = MdvHtmlBlock(
        id: 'aaaa11112222bbbb',
        html: raw,
        unsafe: true,
      );
      await tester.pumpWidget(
        harness(
          MdvDocumentView(
            tree([para('cafe000011110000', 'first'), html]),
            selectable: false,
          ),
        ),
      );
      await tester.tap(find.text('Raw HTML (not rendered)'));
      await tester.pump();
      expect(find.text(raw), findsOneWidget); // disclosure now open

      // Re-render with a NEW block inserted above; same doc otherwise.
      await tester.pumpWidget(
        harness(
          MdvDocumentView(
            tree([
              para('feed000022220000', 'inserted above'),
              para('cafe000011110000', 'first'),
              html,
            ]),
            selectable: false,
          ),
        ),
      );
      await tester.pump();
      expect(find.text('inserted above', findRichText: true), findsOneWidget);
      // ELEMENT/STATE SURVIVAL: the disclosure is still open — its
      // State moved with its (id, occurrence) key to the new index
      // instead of being rebuilt collapsed.
      expect(find.text(raw), findsOneWidget);
    });
  });

  group('theming', () {
    testWidgets('null palette follows the ambient theme brightness', (
      tester,
    ) async {
      await tester.pumpWidget(
        harness(
          MdvDocumentView(tree([para('p', 'dark words')])),
          theme: ThemeData(brightness: Brightness.dark),
        ),
      );
      final span = leafSpan(tester, find.byType(MdvDocumentView), 'dark words');
      expect(span.style?.color, MdvPalette.dark.foreground);
    });

    testWidgets('explicit palette wins over ambient theme', (tester) async {
      final palette = MdvPalette.light.copyWith(
        foreground: const Color(0xFF112233),
      );
      await tester.pumpWidget(
        harness(
          MdvDocumentView(tree([para('p', 'tinted')]), palette: palette),
          theme: ThemeData(brightness: Brightness.dark),
        ),
      );
      final span = leafSpan(tester, find.byType(MdvDocumentView), 'tinted');
      expect(span.style?.color, const Color(0xFF112233));
    });

    testWidgets('selectable=true wraps in SelectionArea; false does not', (
      tester,
    ) async {
      await tester.pumpWidget(
        harness(MdvDocumentView(tree([para('p', 'sel')]))),
      );
      expect(find.byType(SelectionArea), findsOneWidget);
      await tester.pumpWidget(
        harness(MdvDocumentView(tree([para('p', 'sel')]), selectable: false)),
      );
      expect(find.byType(SelectionArea), findsNothing);
    });
  });

  group('math', () {
    testWidgets('valid TeX renders through flutter_math_fork', (tester) async {
      await tester.pumpWidget(
        harness(
          MdvDocumentView(
            tree([
              MdvMathBlock(id: 'm', source: r'\frac{1}{2}', engine: 'katex'),
            ]),
          ),
        ),
      );
      expect(find.byType(Math), findsOneWidget);
      // No fallback: the raw source is not shown as text.
      expect(find.text(r'\frac{1}{2}'), findsNothing);
    });

    testWidgets('invalid TeX falls back to styled source text, no crash', (
      tester,
    ) async {
      await tester.pumpWidget(
        harness(
          MdvDocumentView(
            tree([MdvMathBlock(id: 'm', source: r'\frac{1', engine: 'katex')]),
          ),
        ),
      );
      expect(tester.takeException(), isNull);
      expect(find.text(r'\frac{1'), findsOneWidget);
    });

    testWidgets('inline math renders; invalid inline math falls back', (
      tester,
    ) async {
      await tester.pumpWidget(
        harness(
          MdvDocumentView(
            tree([
              MdvParagraph(
                id: 'p',
                children: const [
                  MdvText(value: 'x is '),
                  MdvMathInline(source: 'x^2'),
                  MdvMathInline(source: r'\oops{'),
                ],
              ),
            ]),
          ),
        ),
      );
      expect(tester.takeException(), isNull);
      expect(find.byType(Math), findsNWidgets(2));
      expect(find.text(r'\oops{'), findsOneWidget);
    });
  });
}
