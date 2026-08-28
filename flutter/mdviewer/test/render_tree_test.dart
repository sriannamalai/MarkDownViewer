import 'dart:convert';
import 'dart:ffi' as ffi;

import 'package:ffi/ffi.dart' as pkgffi;
import 'package:flutter_test/flutter_test.dart';
import 'package:mdviewer/mdviewer.dart';
import 'package:mdviewer/src/bindings.dart';
import 'package:mdviewer/src/library_loader.dart';

// Host-dylib e2e coverage for renderTree / renderTreeRaw /
// renderTreeDoc / renderTreeDocRaw: the typed tree over a kitchen-sink
// document, md-vs-doc parity modulo ids, the resolver contract on the
// tree surface, boundary error mapping, and (at the raw-bindings level,
// since strict-by-construction MdvOptions cannot produce one) the
// wrong-case option key error.

const _kitchenSink = r'''
# Title

A paragraph with *em*, **strong**, ~~strike~~, `code`, a
[link](docs/guide.md), inline math $e^{i\pi}$, and a footnote.[^1]

- [x] done
- [ ] todo
- plain

> quoted

> [!WARNING]
> Careful.

| Left | Right |
|:--|--:|
| a | b |

```go
package main
```

```mermaid
graph TD
  A --> B
```

$$
\int_0^1 x^2\,dx
$$

---

Term
: The definition.

![An image](img/photo.png "Title")

See [[Wiki Page]].

<div>raw <b>html</b> block</div>

[^1]: The note.
''';

/// Depth-first list of every inline in [blocks], nested containers,
/// list items, table cells, and footnote bodies included.
List<MdvInline> _allInlines(List<MdvBlock> blocks) {
  final out = <MdvInline>[];
  void inline(MdvInline i) {
    out.add(i);
    final kids = switch (i) {
      MdvEmphasis v => v.children,
      MdvStrong v => v.children,
      MdvStrikethrough v => v.children,
      MdvLink v => v.children,
      _ => const <MdvInline>[],
    };
    kids.forEach(inline);
  }

  void walk(MdvBlock b) {
    final inlines = switch (b) {
      MdvHeading v => v.children,
      MdvParagraph v => v.children,
      MdvDefinitionTerm v => v.children,
      MdvTable v => [
        for (final cell in v.header) ...cell,
        for (final row in v.rows)
          for (final cell in row) ...cell,
      ],
      _ => const <MdvInline>[],
    };
    inlines.forEach(inline);
    final sub = switch (b) {
      MdvBlockQuote v => v.blocks,
      MdvAdmonition v => v.blocks,
      MdvDefinitionList v => v.blocks,
      MdvDefinitionDesc v => v.blocks,
      MdvFootnoteDef v => v.blocks,
      MdvList v => [for (final item in v.items) ...item.blocks],
      _ => const <MdvBlock>[],
    };
    sub.forEach(walk);
  }

  blocks.forEach(walk);
  return out;
}

/// [v] with every map's "id" AND "defId" entries removed, recursively —
/// for comparing trees whose ids are expected to differ (content-hash
/// vs positional fallback). "defId" (a footnoteRef's ref→definition
/// linkage) is exactly as id-shaped as "id" itself — it carries the
/// matching footnote definition's id, so it diverges in VALUE between
/// the two build paths for the same reason "id" does, even though both
/// surfaces populate it (see the parity test below).
Object? _stripIds(Object? v) {
  if (v is Map<String, dynamic>) {
    return {
      for (final e in v.entries)
        if (e.key != 'id' && e.key != 'defId') e.key: _stripIds(e.value),
    };
  }
  if (v is List) return [for (final x in v) _stripIds(x)];
  return v;
}

/// Every "id" value found anywhere in the raw wire tree [v].
List<String> _collectIds(Object? v) {
  final out = <String>[];
  void go(Object? v) {
    if (v is Map<String, dynamic>) {
      final id = v['id'];
      if (id is String) out.add(id);
      v.values.forEach(go);
    } else if (v is List) {
      v.forEach(go);
    }
  }

  go(v);
  return out;
}

/// Every non-empty "defId" value found anywhere in the raw wire tree
/// [v] (a footnoteRef's ref→definition linkage).
List<String> _collectDefIds(Object? v) {
  final out = <String>[];
  void go(Object? v) {
    if (v is Map<String, dynamic>) {
      final defId = v['defId'];
      if (defId is String && defId.isNotEmpty) out.add(defId);
      v.values.forEach(go);
    } else if (v is List) {
      v.forEach(go);
    }
  }

  go(v);
  return out;
}

void main() {
  final mdv = Mdviewer.instance;

  test('kitchen-sink renderTree: typed tree with every expected kind', () {
    final tree = mdv.renderTree(_kitchenSink);
    expect(tree.version, 1);
    expect(tree.blocks.whereType<MdvUnknownBlock>(), isEmpty);

    final h = tree.blocks.whereType<MdvHeading>().single;
    expect(h.level, 1);
    expect(h.anchorId, isNotNull);
    expect((h.children.single as MdvText).value, 'Title');
    expect(h.span, isNotNull);

    final list = tree.blocks.whereType<MdvList>().single;
    expect(list.ordered, isFalse);
    expect(list.items.map((i) => i.task).toList(), [true, false, null]);

    final code = tree.blocks.whereType<MdvCodeBlock>().single;
    expect(code.language, 'go');
    expect(code.runs, isNotNull);
    expect(code.runs!.map((r) => r.text).join(), code.text);
    expect(code.runs!.map((r) => r.tokenType), contains(contains('Keyword')));

    final diagram = tree.blocks.whereType<MdvDiagram>().single;
    expect(diagram.engine, 'mermaid');
    expect(diagram.source, contains('graph TD'));

    final math = tree.blocks.whereType<MdvMathBlock>().single;
    expect(math.engine, 'katex');
    expect(math.source, contains(r'\int_0^1'));

    final table = tree.blocks.whereType<MdvTable>().single;
    expect(table.alignments, [MdvAlignment.left, MdvAlignment.right]);
    expect(table.header, hasLength(2));
    expect(table.rows, hasLength(1));

    expect(tree.blocks.whereType<MdvBlockQuote>(), hasLength(1));
    final adm = tree.blocks.whereType<MdvAdmonition>().single;
    expect(adm.variant, 'warning');
    expect(adm.title, 'Warning');

    expect(tree.blocks.whereType<MdvThematicBreak>(), hasLength(1));
    expect(tree.blocks.whereType<MdvDefinitionList>(), hasLength(1));
    final html = tree.blocks.whereType<MdvHtmlBlock>().single;
    expect(html.unsafe, isFalse);

    final fd = tree.footnotes.single as MdvFootnoteDef;
    expect(fd.index, 1);
    expect(fd.refCount, 1);

    final inlines = _allInlines(tree.blocks);
    expect(inlines.whereType<MdvUnknownInline>(), isEmpty);
    expect(inlines.whereType<MdvEmphasis>(), isNotEmpty);
    expect(inlines.whereType<MdvStrong>(), isNotEmpty);
    expect(inlines.whereType<MdvStrikethrough>(), isNotEmpty);
    expect(inlines.whereType<MdvCodeSpan>(), isNotEmpty);
    expect(inlines.whereType<MdvMathInline>(), hasLength(1));
    expect(inlines.whereType<MdvFootnoteRef>().single.index, 1);

    final image = inlines.whereType<MdvImage>().single;
    expect(image.url, 'img/photo.png');
    expect(image.alt, 'An image');
    expect(image.title, 'Title');

    final wiki = inlines
        .whereType<MdvLink>()
        .where((l) => l.source == 'wikiLink')
        .single;
    expect(wiki.url, 'Wiki Page.md');
    final plainLink = inlines
        .whereType<MdvLink>()
        .where((l) => l.source == null)
        .single;
    expect(plainLink.url, 'docs/guide.md');
    expect(plainLink.blocked, isFalse);
  });

  test('typed model round-trips the raw map (fromMap over renderTreeRaw)', () {
    final raw = mdv.renderTreeRaw(_kitchenSink);
    expect(raw['version'], 1);
    final typed = MdvTree.fromMap(raw);
    expect(typed.blocks, hasLength((raw['blocks'] as List).length));
    expect(typed.footnotes, hasLength((raw['footnotes'] as List).length));
    expect(typed.blocks.first, isA<MdvHeading>());
  });

  test('renderTreeDoc(parse(md)) equals renderTree(md) modulo ids, for '
      'both the Map and JSON-string document forms', () {
    final fromMd = mdv.renderTreeRaw(_kitchenSink);
    final doc = mdv.parse(_kitchenSink);
    final fromDoc = mdv.renderTreeDocRaw(doc);
    expect(_stripIds(fromDoc), _stripIds(fromMd));
    expect(_stripIds(mdv.renderTreeDocRaw(jsonEncode(doc))), _stripIds(fromMd));
    // Both surfaces mint well-formed 16-hex ids (content hashes from
    // markdown, deterministic positional fallbacks from a doc).
    final hex16 = RegExp(r'^[0-9a-f]{16}$');
    for (final id in [..._collectIds(fromMd), ..._collectIds(fromDoc)]) {
      expect(id, matches(hex16));
    }
    // The footnote ref->definition linkage (defId) is populated on
    // BOTH surfaces — not just the content-hash (markdown) path — and
    // every value matches a real footnote definition's id in that same
    // tree, even though the concrete id values differ between the two
    // surfaces (content hash vs positional fallback, like every other
    // id).
    final mdFootnoteIds = _collectIds((fromMd['footnotes'] as List));
    final docFootnoteIds = _collectIds((fromDoc['footnotes'] as List));
    final mdDefIds = _collectDefIds(fromMd);
    final docDefIds = _collectDefIds(fromDoc);
    expect(mdDefIds, isNotEmpty);
    expect(docDefIds, isNotEmpty);
    for (final defId in mdDefIds) {
      expect(mdFootnoteIds, contains(defId));
    }
    for (final defId in docDefIds) {
      expect(docFootnoteIds, contains(defId));
    }
    // And the typed path parses the doc-built tree too.
    expect(mdv.renderTreeDoc(doc).blocks.first, isA<MdvHeading>());
  });

  test('byte-identical blocks share an id (the documented duplicate-id '
      'contract: key by (id, occurrenceIndex), never id alone)', () {
    final tree = mdv.renderTree('dup\n\nother\n\ndup\n');
    final ids = tree.blocks.whereType<MdvParagraph>().map((p) => p.id).toList();
    expect(ids, hasLength(3));
    expect(ids[0], ids[2]);
    expect(ids[0], isNot(ids[1]));
  });

  group('resolver on the tree surface', () {
    const rmd =
        '![alt](img/photo.png)\n\n[click](docs/guide.md)\n\n'
        '[[Wiki Page]]\n\n[e](javascript:x())\n';

    test('image resolved verbatim; declined targets take default '
        'resolution; javascript: link blocked', () {
      final tree = mdv.renderTree(
        rmd,
        options: MdvOptions(
          resolver: (kind, target) =>
              kind == MdvResolveKind.image ? 'asset://$target' : null,
        ),
      );
      final inlines = _allInlines(tree.blocks);
      final image = inlines.whereType<MdvImage>().single;
      expect(image.url, 'asset://img/photo.png');
      expect(image.blocked, isFalse);
      final links = inlines.whereType<MdvLink>().toList();
      expect(links[0].url, 'docs/guide.md'); // declined: default resolution
      expect(links[1].url, 'Wiki Page.md'); // declined wiki: .md fallback
      expect(links[1].source, 'wikiLink');
      final js = links[2];
      expect(js.url, isEmpty); // declined then policy-blocked
      expect(js.blocked, isTrue);
    });

    test('trust contract: a resolver-returned URL is carried verbatim, '
        'bypassing safe-URL policy', () {
      final tree = mdv.renderTree(
        '[e](javascript:x())',
        options: MdvOptions(resolver: (kind, target) => 'app://allowed'),
      );
      final link = _allInlines(tree.blocks).whereType<MdvLink>().single;
      expect(link.url, 'app://allowed');
      expect(link.blocked, isFalse);
    });

    test('resolver works through renderTreeDoc too', () {
      final tree = mdv.renderTreeDoc(
        mdv.parse(rmd),
        options: MdvOptions(
          resolver: (kind, target) =>
              kind == MdvResolveKind.image ? 'asset://$target' : null,
        ),
      );
      final image = _allInlines(tree.blocks).whereType<MdvImage>().single;
      expect(image.url, 'asset://img/photo.png');
    });
  });

  test('bad doc JSON to renderTreeDoc throws MdviewerException naming '
      'the problem', () {
    expect(
      () => mdv.renderTreeDoc('{not valid json'),
      throwsA(
        isA<MdviewerException>().having(
          (e) => e.message,
          'message',
          allOf(contains('document'), contains('invalid JSON')),
        ),
      ),
    );
  });

  test('wrong-case option key surfaces the boundary unknown-field error '
      '(raw bindings call: strict-by-construction MdvOptions cannot emit '
      'an unknown key, mirroring the C harness case)', () {
    final b = MdvBindings(openMdviewerLibrary());
    final md = utf8.encode('# Hi\n');
    final inPtr = pkgffi.malloc<ffi.Uint8>(md.length);
    inPtr.asTypedList(md.length).setAll(0, md);
    final optsPtr = '{"Highlighting": true}'
        .toNativeUtf8(allocator: pkgffi.malloc)
        .cast<ffi.Char>();
    final out = pkgffi.malloc<ffi.Pointer<ffi.Char>>();
    final outLen = pkgffi.malloc<ffi.Size>();
    final outErr = pkgffi.malloc<ffi.Pointer<ffi.Char>>();
    try {
      final rc = b.renderTree(
        inPtr.cast(),
        md.length,
        optsPtr,
        out,
        outLen,
        outErr,
      );
      expect(rc, isNot(0));
      final err = outErr.value;
      expect(err.address, isNot(0));
      final msg = err.cast<pkgffi.Utf8>().toDartString();
      b.free(err);
      expect(msg, contains('unknown field'));
      expect(msg, contains('Highlighting'));
    } finally {
      pkgffi.malloc.free(inPtr);
      pkgffi.malloc.free(optsPtr);
      pkgffi.malloc.free(out);
      pkgffi.malloc.free(outLen);
      pkgffi.malloc.free(outErr);
    }
  });
}
