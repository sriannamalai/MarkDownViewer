import 'package:flutter_test/flutter_test.dart';
import 'package:mdviewer/mdviewer.dart';

// Pure-Dart coverage for MdvTree.fromMap: strict parsing of hand-built
// version-1 wire maps per kind (every optional-field state), the
// forward-compat unknown-kind rule (Unknown nodes, never a throw), and
// malformed-structure FormatExceptions naming the offending path. No
// dylib involved — the e2e path is render_tree_test.dart.

Map<String, dynamic> _env(
  List<Map<String, dynamic>> blocks, [
  List<Map<String, dynamic>> footnotes = const [],
]) => {'version': 1, 'blocks': blocks, 'footnotes': footnotes};

const _span = {'startLine': 1, 'endLine': 2, 'startOffset': 0, 'endOffset': 10};

Matcher _formatExceptionNaming(String path) => throwsA(
  isA<FormatException>().having((e) => e.message, 'message', startsWith(path)),
);

void main() {
  group('envelope', () {
    test('empty tree parses with empty block/footnote lists', () {
      final tree = MdvTree.fromMap(_env([]));
      expect(tree.version, 1);
      expect(tree.blocks, isEmpty);
      expect(tree.footnotes, isEmpty);
    });

    test('missing version throws naming "version"', () {
      expect(
        () => MdvTree.fromMap({'blocks': [], 'footnotes': []}),
        _formatExceptionNaming('version'),
      );
    });

    test('unsupported version throws naming "version"', () {
      expect(
        () => MdvTree.fromMap({'version': 2, 'blocks': [], 'footnotes': []}),
        _formatExceptionNaming('version'),
      );
    });

    test('non-list blocks throws naming "blocks"', () {
      expect(
        () => MdvTree.fromMap({'version': 1, 'blocks': 'x', 'footnotes': []}),
        _formatExceptionNaming('blocks'),
      );
    });

    test('missing footnotes throws naming "footnotes"', () {
      expect(
        () => MdvTree.fromMap({'version': 1, 'blocks': []}),
        _formatExceptionNaming('footnotes'),
      );
    });
  });

  group('heading', () {
    test('with anchorId and span', () {
      final tree = MdvTree.fromMap(
        _env([
          {
            'kind': 'heading',
            'span': _span,
            'id': 'a' * 16,
            'level': 2,
            'anchorId': 'hi',
            'children': [
              {'kind': 'text', 'value': 'Hi'},
            ],
          },
        ]),
      );
      final h = tree.blocks.single as MdvHeading;
      expect(h.id, 'a' * 16);
      expect(h.level, 2);
      expect(h.anchorId, 'hi');
      expect(h.span!.startLine, 1);
      expect(h.span!.endOffset, 10);
      expect((h.children.single as MdvText).value, 'Hi');
    });

    test('without anchorId and without span', () {
      final tree = MdvTree.fromMap(
        _env([
          {'kind': 'heading', 'id': 'x', 'level': 1, 'children': []},
        ]),
      );
      final h = tree.blocks.single as MdvHeading;
      expect(h.anchorId, isNull);
      expect(h.span, isNull);
    });

    test('missing level throws naming the path', () {
      expect(
        () => MdvTree.fromMap(
          _env([
            {'kind': 'heading', 'id': 'x', 'children': []},
          ]),
        ),
        _formatExceptionNaming('blocks[0].level'),
      );
    });

    test('wrong-typed level throws naming the path', () {
      expect(
        () => MdvTree.fromMap(
          _env([
            {'kind': 'heading', 'id': 'x', 'level': 'one', 'children': []},
          ]),
        ),
        _formatExceptionNaming('blocks[0].level'),
      );
    });
  });

  group('paragraph + inline containers', () {
    test('emphasis/strong/strikethrough/codeSpan/breaks nest', () {
      final tree = MdvTree.fromMap(
        _env([
          {
            'kind': 'paragraph',
            'id': 'p',
            'children': [
              {
                'kind': 'emphasis',
                'children': [
                  {
                    'kind': 'strong',
                    'children': [
                      {'kind': 'text', 'value': 'both'},
                    ],
                  },
                ],
              },
              {'kind': 'softBreak'},
              {
                'kind': 'strikethrough',
                'children': [
                  {'kind': 'text', 'value': 'gone'},
                ],
              },
              {'kind': 'hardBreak'},
              {'kind': 'codeSpan', 'value': 'x+y'},
            ],
          },
        ]),
      );
      final p = tree.blocks.single as MdvParagraph;
      final em = p.children[0] as MdvEmphasis;
      final strong = em.children.single as MdvStrong;
      expect((strong.children.single as MdvText).value, 'both');
      expect(p.children[1], isA<MdvSoftBreak>());
      expect(p.children[3], isA<MdvHardBreak>());
      expect((p.children[4] as MdvCodeSpan).value, 'x+y');
      expect(p.children[1].span, isNull);
    });

    test('malformed nested inline names the full path', () {
      expect(
        () => MdvTree.fromMap(
          _env([
            {
              'kind': 'paragraph',
              'id': 'p',
              'children': [
                {
                  'kind': 'emphasis',
                  'children': [
                    {'kind': 'text'},
                  ],
                },
              ],
            },
          ]),
        ),
        _formatExceptionNaming('blocks[0].children[0].children[0].value'),
      );
    });
  });

  group('list', () {
    test('task tri-state: null, false, true', () {
      final tree = MdvTree.fromMap(
        _env([
          {
            'kind': 'list',
            'id': 'l',
            'ordered': true,
            'start': 3,
            'tight': false,
            'items': [
              {'id': 'i0', 'task': null, 'blocks': []},
              {'id': 'i1', 'task': false, 'blocks': []},
              {
                'id': 'i2',
                'span': _span,
                'task': true,
                'blocks': [
                  {'kind': 'paragraph', 'id': 'p', 'children': []},
                ],
              },
            ],
          },
        ]),
      );
      final l = tree.blocks.single as MdvList;
      expect(l.ordered, isTrue);
      expect(l.start, 3);
      expect(l.tight, isFalse);
      expect(l.items.map((i) => i.task).toList(), [null, false, true]);
      expect(l.items[2].span, isNotNull);
      expect(l.items[2].blocks.single, isA<MdvParagraph>());
    });

    test('missing task key throws naming the item path', () {
      expect(
        () => MdvTree.fromMap(
          _env([
            {
              'kind': 'list',
              'id': 'l',
              'ordered': false,
              'start': 1,
              'tight': true,
              'items': [
                {'id': 'i0', 'blocks': []},
              ],
            },
          ]),
        ),
        _formatExceptionNaming('blocks[0].items[0].task'),
      );
    });

    test('non-bool task throws naming the item path', () {
      expect(
        () => MdvTree.fromMap(
          _env([
            {
              'kind': 'list',
              'id': 'l',
              'ordered': false,
              'start': 1,
              'tight': true,
              'items': [
                {'id': 'i0', 'task': 'yes', 'blocks': []},
              ],
            },
          ]),
        ),
        _formatExceptionNaming('blocks[0].items[0].task'),
      );
    });
  });

  group('blockQuote', () {
    test('nests blocks', () {
      final tree = MdvTree.fromMap(
        _env([
          {
            'kind': 'blockQuote',
            'id': 'q',
            'blocks': [
              {'kind': 'thematicBreak', 'id': 't'},
            ],
          },
        ]),
      );
      final q = tree.blocks.single as MdvBlockQuote;
      expect(q.blocks.single, isA<MdvThematicBreak>());
    });
  });

  group('codeBlock', () {
    test('runs null (highlighting off/unavailable)', () {
      final tree = MdvTree.fromMap(
        _env([
          {
            'kind': 'codeBlock',
            'id': 'c',
            'language': '',
            'label': 'code',
            'runs': null,
            'text': 'plain\n',
          },
        ]),
      );
      final c = tree.blocks.single as MdvCodeBlock;
      expect(c.runs, isNull);
      expect(c.text, 'plain\n');
      expect(c.label, 'code');
    });

    test('runs array of token runs', () {
      final tree = MdvTree.fromMap(
        _env([
          {
            'kind': 'codeBlock',
            'id': 'c',
            'language': 'go',
            'label': 'go',
            'runs': [
              {'text': 'package', 'tokenType': 'Keyword'},
              {'text': ' main\n', 'tokenType': 'Text'},
            ],
            'text': 'package main\n',
          },
        ]),
      );
      final c = tree.blocks.single as MdvCodeBlock;
      expect(c.runs, hasLength(2));
      expect(c.runs![0].tokenType, 'Keyword');
      expect(c.runs!.map((r) => r.text).join(), c.text);
    });

    test('missing runs key throws naming the path', () {
      expect(
        () => MdvTree.fromMap(
          _env([
            {
              'kind': 'codeBlock',
              'id': 'c',
              'language': '',
              'label': 'code',
              'text': '',
            },
          ]),
        ),
        _formatExceptionNaming('blocks[0].runs'),
      );
    });

    test('malformed run names the run path', () {
      expect(
        () => MdvTree.fromMap(
          _env([
            {
              'kind': 'codeBlock',
              'id': 'c',
              'language': '',
              'label': 'code',
              'runs': [
                {'text': 'x'},
              ],
              'text': 'x',
            },
          ]),
        ),
        _formatExceptionNaming('blocks[0].runs[0].tokenType'),
      );
    });
  });

  group('admonition', () {
    test('variant, title, blocks', () {
      final tree = MdvTree.fromMap(
        _env([
          {
            'kind': 'admonition',
            'id': 'a',
            'variant': 'warning',
            'title': 'Warning',
            'blocks': [
              {'kind': 'paragraph', 'id': 'p', 'children': []},
            ],
          },
        ]),
      );
      final a = tree.blocks.single as MdvAdmonition;
      expect(a.variant, 'warning');
      expect(a.title, 'Warning');
      expect(a.blocks, hasLength(1));
    });
  });

  group('table', () {
    test('all four alignments, header and rows of inline cells', () {
      final tree = MdvTree.fromMap(
        _env([
          {
            'kind': 'table',
            'id': 't',
            'alignments': ['none', 'left', 'center', 'right'],
            'header': [
              [
                {'kind': 'text', 'value': 'H'},
              ],
              [],
              [],
              [],
            ],
            'rows': [
              [
                [
                  {'kind': 'text', 'value': 'a'},
                ],
                [],
                [],
                [],
              ],
            ],
          },
        ]),
      );
      final t = tree.blocks.single as MdvTable;
      expect(t.alignments, MdvAlignment.values);
      expect((t.header.first.single as MdvText).value, 'H');
      expect((t.rows.single.first.single as MdvText).value, 'a');
    });

    test('unknown alignment string throws naming the path', () {
      expect(
        () => MdvTree.fromMap(
          _env([
            {
              'kind': 'table',
              'id': 't',
              'alignments': ['middle'],
              'header': [],
              'rows': [],
            },
          ]),
        ),
        _formatExceptionNaming('blocks[0].alignments[0]'),
      );
    });

    test('malformed row cell names the row path', () {
      expect(
        () => MdvTree.fromMap(
          _env([
            {
              'kind': 'table',
              'id': 't',
              'alignments': [],
              'header': [],
              'rows': [
                ['not-a-cell'],
              ],
            },
          ]),
        ),
        _formatExceptionNaming('blocks[0].rows[0][0]'),
      );
    });
  });

  group('mathBlock / diagram / htmlBlock', () {
    test('mathBlock and diagram carry source + engine', () {
      final tree = MdvTree.fromMap(
        _env([
          {'kind': 'mathBlock', 'id': 'm', 'source': 'x^2', 'engine': 'katex'},
          {
            'kind': 'diagram',
            'id': 'd',
            'source': 'graph TD',
            'engine': 'mermaid',
          },
        ]),
      );
      final m = tree.blocks[0] as MdvMathBlock;
      expect(m.source, 'x^2');
      expect(m.engine, 'katex');
      final d = tree.blocks[1] as MdvDiagram;
      expect(d.engine, 'mermaid');
    });

    test('htmlBlock sanitized vs unsafe states', () {
      final tree = MdvTree.fromMap(
        _env([
          {
            'kind': 'htmlBlock',
            'id': 'h1',
            'html': '<b>x</b>',
            'unsafe': false,
          },
          {
            'kind': 'htmlBlock',
            'id': 'h2',
            'html': '<script>x</script>',
            'unsafe': true,
          },
        ]),
      );
      expect((tree.blocks[0] as MdvHtmlBlock).unsafe, isFalse);
      expect((tree.blocks[1] as MdvHtmlBlock).unsafe, isTrue);
    });

    test('missing html throws naming the path', () {
      expect(
        () => MdvTree.fromMap(
          _env([
            {'kind': 'htmlBlock', 'id': 'h', 'unsafe': false},
          ]),
        ),
        _formatExceptionNaming('blocks[0].html'),
      );
    });
  });

  group('definition list family', () {
    test('definitionList of terms and descs', () {
      final tree = MdvTree.fromMap(
        _env([
          {
            'kind': 'definitionList',
            'id': 'dl',
            'blocks': [
              {
                'kind': 'definitionTerm',
                'id': 'dt',
                'children': [
                  {'kind': 'text', 'value': 'Term'},
                ],
              },
              {
                'kind': 'definitionDesc',
                'id': 'dd',
                'blocks': [
                  {'kind': 'paragraph', 'id': 'p', 'children': []},
                ],
              },
            ],
          },
        ]),
      );
      final dl = tree.blocks.single as MdvDefinitionList;
      expect(dl.blocks[0], isA<MdvDefinitionTerm>());
      expect(dl.blocks[1], isA<MdvDefinitionDesc>());
    });
  });

  group('footnotes', () {
    test('footnoteDef in the envelope with index/refCount', () {
      final tree = MdvTree.fromMap(
        _env(
          [
            {
              'kind': 'paragraph',
              'id': 'p',
              'children': [
                {'kind': 'footnoteRef', 'index': 1},
              ],
            },
          ],
          [
            {
              'kind': 'footnoteDef',
              'id': 'f',
              'index': 1,
              'refCount': 2,
              'blocks': [
                {'kind': 'paragraph', 'id': 'p2', 'children': []},
              ],
            },
          ],
        ),
      );
      final f = tree.footnotes.single as MdvFootnoteDef;
      expect(f.index, 1);
      expect(f.refCount, 2);
      final p = tree.blocks.single as MdvParagraph;
      expect((p.children.single as MdvFootnoteRef).index, 1);
    });

    test('malformed footnote names the footnotes path', () {
      expect(
        () => MdvTree.fromMap(
          _env([], [
            {'kind': 'footnoteDef', 'id': 'f', 'refCount': 1, 'blocks': []},
          ]),
        ),
        _formatExceptionNaming('footnotes[0].index'),
      );
    });
  });

  group('link and image', () {
    test('link: every optional-field state', () {
      final tree = MdvTree.fromMap(
        _env([
          {
            'kind': 'paragraph',
            'id': 'p',
            'children': [
              {'kind': 'link', 'url': 'a.md', 'children': []},
              {
                'kind': 'link',
                'url': '',
                'blocked': true,
                'children': [
                  {'kind': 'text', 'value': 'evil'},
                ],
              },
              {
                'kind': 'link',
                'url': 'b.md',
                'title': 'Bee',
                'source': 'wikiLink',
                'children': [],
              },
            ],
          },
        ]),
      );
      final p = tree.blocks.single as MdvParagraph;
      final plain = p.children[0] as MdvLink;
      expect(plain.url, 'a.md');
      expect(plain.blocked, isFalse);
      expect(plain.title, isNull);
      expect(plain.source, isNull);
      final blocked = p.children[1] as MdvLink;
      expect(blocked.url, isEmpty);
      expect(blocked.blocked, isTrue);
      final wiki = p.children[2] as MdvLink;
      expect(wiki.title, 'Bee');
      expect(wiki.source, 'wikiLink');
    });

    test('link missing url throws naming the path', () {
      expect(
        () => MdvTree.fromMap(
          _env([
            {
              'kind': 'paragraph',
              'id': 'p',
              'children': [
                {'kind': 'link', 'children': []},
              ],
            },
          ]),
        ),
        _formatExceptionNaming('blocks[0].children[0].url'),
      );
    });

    test('image: url and alt required, blocked/title optional', () {
      final tree = MdvTree.fromMap(
        _env([
          {
            'kind': 'paragraph',
            'id': 'p',
            'children': [
              {'kind': 'image', 'url': 'x.png', 'alt': 'x'},
              {
                'kind': 'image',
                'url': '',
                'blocked': true,
                'alt': 'y',
                'title': 'why',
              },
            ],
          },
        ]),
      );
      final p = tree.blocks.single as MdvParagraph;
      final plain = p.children[0] as MdvImage;
      expect(plain.blocked, isFalse);
      expect(plain.title, isNull);
      final blocked = p.children[1] as MdvImage;
      expect(blocked.blocked, isTrue);
      expect(blocked.title, 'why');
    });
  });

  group('mathInline / htmlInline', () {
    test('display absent defaults false; present true carries', () {
      final tree = MdvTree.fromMap(
        _env([
          {
            'kind': 'paragraph',
            'id': 'p',
            'children': [
              {'kind': 'mathInline', 'source': 'x'},
              {'kind': 'mathInline', 'source': 'y', 'display': true},
              {'kind': 'htmlInline', 'html': '<i>', 'unsafe': false},
            ],
          },
        ]),
      );
      final p = tree.blocks.single as MdvParagraph;
      expect((p.children[0] as MdvMathInline).display, isFalse);
      expect((p.children[1] as MdvMathInline).display, isTrue);
      expect((p.children[2] as MdvHtmlInline).html, '<i>');
    });
  });

  group('span strictness', () {
    test('span with a missing field names the span path', () {
      expect(
        () => MdvTree.fromMap(
          _env([
            {
              'kind': 'thematicBreak',
              'id': 't',
              'span': {'startLine': 1, 'endLine': 1, 'startOffset': 0},
            },
          ]),
        ),
        _formatExceptionNaming('blocks[0].span.endOffset'),
      );
    });
  });

  group('forward compat: unknown kinds', () {
    test('unknown block kind decodes as MdvUnknownBlock with the raw map', () {
      final raw = {'kind': 'hologram', 'span': _span, 'id': 'h', 'shimmer': 11};
      final tree = MdvTree.fromMap(_env([raw]));
      final u = tree.blocks.single as MdvUnknownBlock;
      expect(u.kind, 'hologram');
      expect(u.id, 'h');
      expect(u.span, isNotNull);
      expect(u.raw, same(raw));
    });

    test('unknown block without id/span still never throws', () {
      final tree = MdvTree.fromMap(
        _env([
          {'kind': 'hologram', 'span': 'garbled'},
        ]),
      );
      final u = tree.blocks.single as MdvUnknownBlock;
      expect(u.id, isEmpty);
      expect(u.span, isNull);
    });

    test('unknown inline kind decodes as MdvUnknownInline, nested', () {
      final raw = {'kind': 'sparkle', 'value': '*'};
      final tree = MdvTree.fromMap(
        _env([
          {
            'kind': 'paragraph',
            'id': 'p',
            'children': [
              {
                'kind': 'emphasis',
                'children': [raw],
              },
            ],
          },
        ]),
      );
      final p = tree.blocks.single as MdvParagraph;
      final em = p.children.single as MdvEmphasis;
      final u = em.children.single as MdvUnknownInline;
      expect(u.kind, 'sparkle');
      expect(u.raw, same(raw));
    });

    test('unknown kind in footnotes decodes as MdvUnknownBlock', () {
      final tree = MdvTree.fromMap(
        _env([], [
          {'kind': 'endnoteDef', 'id': 'e'},
        ]),
      );
      expect(tree.footnotes.single, isA<MdvUnknownBlock>());
    });
  });

  group('malformed structure', () {
    test('block that is not an object names its index', () {
      expect(
        () => MdvTree.fromMap(_env([])..['blocks'] = ['nope']),
        _formatExceptionNaming('blocks[0]'),
      );
    });

    test('missing kind names the path', () {
      expect(
        () => MdvTree.fromMap(
          _env([
            {'id': 'x'},
          ]),
        ),
        _formatExceptionNaming('blocks[0].kind'),
      );
    });

    test('non-list children names the path', () {
      expect(
        () => MdvTree.fromMap(
          _env([
            {'kind': 'paragraph', 'id': 'p', 'children': 'text'},
          ]),
        ),
        _formatExceptionNaming('blocks[0].children'),
      );
    });

    test('missing block id names the path', () {
      expect(
        () => MdvTree.fromMap(
          _env([
            {'kind': 'paragraph', 'children': []},
          ]),
        ),
        _formatExceptionNaming('blocks[0].id'),
      );
    });
  });
}
