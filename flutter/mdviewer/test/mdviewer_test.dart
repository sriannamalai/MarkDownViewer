import 'dart:convert';

import 'package:flutter_test/flutter_test.dart';
import 'package:mdviewer/mdviewer.dart';

void main() {
  final mdv = Mdviewer.instance;
  const md = '# Hello *world*\n\n- [x] done\n';

  test('version returns a non-empty string', () {
    expect(mdv.version, isNotEmpty);
  });

  test('default render is a full page with content', () {
    final page = mdv.render(md);
    expect(page, contains('<html'));
    expect(page, contains('<h1'));
    expect(page, contains('Hello'));
  });

  test('fragment option renders no <html', () {
    final frag = mdv.render(md, options: const MdvOptions(fragment: true));
    expect(frag, isNot(contains('<html')));
  });

  test('parse returns a version-1 document map', () {
    final doc = mdv.parse(md);
    expect(doc['version'], 1);
  });

  test('render and parse->renderDoc agree byte-for-byte (Map and String)', () {
    final page = mdv.render(md);
    final doc = mdv.parse(md);
    expect(mdv.renderDoc(doc), page);
    expect(mdv.renderDoc(jsonEncode(doc)), page);
  });

  test('unknown asset error message lists valid names', () {
    expect(
      () => mdv.asset('nope.css'),
      throwsA(
        isA<MdviewerException>().having(
          (e) => e.message,
          'message',
          contains('valid:'),
        ),
      ),
    );
  });

  test('asset(mermaid.js) returns non-empty bytes that look like mermaid', () {
    final bytes = mdv.asset('mermaid.js');
    expect(bytes, isNotEmpty);
    expect(utf8.decode(bytes.take(200000).toList()), contains('mermaid'));
  });

  test('asset(theme-dark.css) has theme tokens and chroma highlight CSS', () {
    final css = utf8.decode(mdv.asset('theme-dark.css'));
    expect(css, contains('--md-bg'));
    expect(css, contains('.chroma'));
  });

  test('asset(theme-light.json) is version-1 palette data', () {
    final decoded =
        jsonDecode(utf8.decode(mdv.asset('theme-light.json')))
            as Map<String, dynamic>;
    expect(decoded['version'], 1);
    expect(decoded['mode'], 'light');
    final vars = decoded['vars'] as Map<String, dynamic>;
    expect(vars['--md-bg'], isNotEmpty);
  });

  test(
    'bad doc JSON to renderDoc throws MdviewerException naming the problem',
    () {
      expect(
        () => mdv.renderDoc('{not valid json'),
        throwsA(
          isA<MdviewerException>().having(
            (e) => e.message,
            'message',
            allOf(contains('document'), contains('invalid JSON')),
          ),
        ),
      );
    },
  );

  test('maxWidth flows through to the rendered page', () {
    final page = mdv.render(md, options: const MdvOptions(maxWidth: '860px'));
    expect(page, contains('--md-max-width'));
    expect(page, contains('860px'));
  });

  test('theme option changes the rendered page', () {
    final light = mdv.render(md, options: const MdvOptions(theme: 'light'));
    final dark = mdv.render(md, options: const MdvOptions(theme: 'dark'));
    expect(light, isNot(equals(dark)));
  });

  test('all-null MdvOptions().toJson() is null (defaults path)', () {
    expect(const MdvOptions().toJson(), isNull);
  });

  test('toJson serializes only explicitly-set fields, never "version"', () {
    final json = const MdvOptions(theme: 'dark', sourceMap: false).toJson();
    expect(json, isNotNull);
    final map = jsonDecode(json!) as Map<String, dynamic>;
    expect(map, {'theme': 'dark', 'sourceMap': false});
    expect(map.containsKey('version'), isFalse);
  });

  test(
    'parse with a resolver throws ArgumentError (permanent, not a Task-4 seam)',
    () {
      expect(
        () => mdv.parse(
          md,
          options: MdvOptions(resolver: (kind, target) => null),
        ),
        throwsA(
          isA<ArgumentError>().having(
            (e) => e.message,
            'message',
            'parse does not take a resolver (resolution is a render-time concern)',
          ),
        ),
      );
    },
  );
}
