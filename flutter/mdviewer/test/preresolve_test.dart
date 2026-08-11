import 'package:flutter_test/flutter_test.dart';
import 'package:mdviewer/mdviewer.dart';

// Pre-resolve helper coverage: pure-Dart tests against a hand-built AST
// map matching the pinned version-1 document codec (`kind`/`children`,
// `destination` on link/image, `target` on wikiLink), plus host-dylib
// tests running the real parse() to keep the hand-built shape honest.
void main() {
  // Hand-built version-1 document: image, link, wiki-link, a repeated
  // link target inside a nested container, and a footnote-hosted link.
  final doc = <String, dynamic>{
    'version': 1,
    'kind': 'document',
    'children': [
      {
        'kind': 'paragraph',
        'children': [
          {
            'kind': 'image',
            'destination': 'img/photo.png',
            'alt': 'alt',
            'children': [
              {'kind': 'text', 'value': 'alt'},
            ],
          },
        ],
      },
      {
        'kind': 'paragraph',
        'children': [
          {
            'kind': 'link',
            'destination': 'docs/guide.md',
            'children': [
              {'kind': 'text', 'value': 'click'},
            ],
          },
        ],
      },
      {
        'kind': 'paragraph',
        'children': [
          {
            'kind': 'wikiLink',
            'target': 'Wiki Page',
            'children': [
              {'kind': 'text', 'value': 'Wiki Page'},
            ],
          },
        ],
      },
      {
        'kind': 'blockQuote',
        'children': [
          {
            'kind': 'paragraph',
            'children': [
              {'kind': 'text', 'value': 'quoted '},
              {
                'kind': 'link',
                'destination': 'docs/guide.md', // repeat: must dedupe
                'children': [
                  {'kind': 'text', 'value': 'again'},
                ],
              },
            ],
          },
        ],
      },
    ],
    'footnotes': [
      {
        'kind': 'footnoteDef',
        'children': [
          {
            'kind': 'paragraph',
            'children': [
              {
                'kind': 'link',
                'destination': 'refs/source.md',
                'children': [
                  {'kind': 'text', 'value': 'source'},
                ],
              },
            ],
          },
        ],
      },
    ],
  };

  group('collectResolvables (pure Dart, hand-built codec map)', () {
    test('finds all three kinds with ABI ints 0/1/2 and verbatim targets', () {
      final found = collectResolvables(doc);
      expect(found, contains(const Resolvable(0, 'docs/guide.md')));
      expect(found, contains(const Resolvable(1, 'img/photo.png')));
      expect(found, contains(const Resolvable(2, 'Wiki Page')));
    });

    test('dedupes a repeated (kind, target) pair', () {
      final found = collectResolvables(doc);
      final guides = found.where(
        (r) => r.kind == 0 && r.target == 'docs/guide.md',
      );
      expect(guides, hasLength(1));
    });

    test('walks footnote definitions too', () {
      expect(
        collectResolvables(doc),
        contains(const Resolvable(0, 'refs/source.md')),
      );
    });

    test('same target under different kinds stays distinct', () {
      final twoKinds = <String, dynamic>{
        'version': 1,
        'kind': 'document',
        'children': [
          {
            'kind': 'paragraph',
            'children': [
              {'kind': 'link', 'destination': 'shared.png'},
              {'kind': 'image', 'destination': 'shared.png'},
            ],
          },
        ],
      };
      final found = collectResolvables(twoKinds);
      expect(found, hasLength(2));
      expect(found, contains(const Resolvable(0, 'shared.png')));
      expect(found, contains(const Resolvable(1, 'shared.png')));
    });

    test('empty document yields no resolvables', () {
      expect(collectResolvables({'version': 1, 'kind': 'document'}), isEmpty);
    });
  });

  group('Resolvable value semantics', () {
    test('equal fields compare equal and hash equal (Set dedupe works)', () {
      const a = Resolvable(1, 'x.png');
      const b = Resolvable(1, 'x.png');
      expect(a, b);
      expect(a.hashCode, b.hashCode);
      expect(
        <Resolvable>{}
          ..add(a)
          ..add(b),
        hasLength(1),
      );
      expect(a, isNot(const Resolvable(0, 'x.png')));
      expect(a, isNot(const Resolvable(1, 'y.png')));
    });
  });

  group('resolverFromMap', () {
    test('answers a mapped target verbatim', () {
      final r = resolverFromMap({'img/photo.png': 'asset://img/photo.png'});
      expect(r(MdvResolveKind.image, 'img/photo.png'), 'asset://img/photo.png');
    });

    test('declines an unmapped target with null', () {
      final r = resolverFromMap({'img/photo.png': 'asset://img/photo.png'});
      expect(r(MdvResolveKind.link, 'docs/guide.md'), isNull);
    });

    test('kindFilter restricts answering to the listed kinds', () {
      final r = resolverFromMap(
        {'shared.png': 'asset://shared.png'},
        kindFilter: {1}, // images only
      );
      expect(r(MdvResolveKind.image, 'shared.png'), 'asset://shared.png');
      expect(r(MdvResolveKind.link, 'shared.png'), isNull);
      expect(r(MdvResolveKind.wikiLink, 'shared.png'), isNull);
    });
  });

  group('host dylib (real parse output)', () {
    const md =
        '![alt](img/photo.png)\n\n[click](docs/guide.md)\n\n[[Wiki Page]]\n\n'
        '[again](docs/guide.md)\n';

    test('collectResolvables finds all three kinds and dedupes the repeat', () {
      final parsed = Mdviewer.instance.parse(md);
      final found = collectResolvables(parsed);
      expect(found, hasLength(3));
      expect(found, contains(const Resolvable(0, 'docs/guide.md')));
      expect(found, contains(const Resolvable(1, 'img/photo.png')));
      expect(found, contains(const Resolvable(2, 'Wiki Page')));
    });

    test('full async-vault recipe: parse -> collect -> map -> renderDoc', () {
      final mdv = Mdviewer.instance;
      final parsed = mdv.parse(md);
      // Stand-in for the host's async prefetch of each collected target.
      final resolved = {
        for (final r in collectResolvables(parsed))
          if (r.kind == 1) r.target: 'asset://${r.target}',
      };
      final page = mdv.renderDoc(
        parsed,
        options: MdvOptions(
          fragment: true,
          resolver: resolverFromMap(resolved, kindFilter: {1}),
        ),
      );
      expect(page, contains('src="asset://img/photo.png"'));
      expect(page, contains('href="docs/guide.md"')); // declined: default
      expect(page, contains('Wiki Page.md')); // declined: .md fallback
    });
  });
}
