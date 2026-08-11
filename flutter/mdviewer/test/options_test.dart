import 'dart:convert';

import 'package:flutter_test/flutter_test.dart';
import 'package:mdviewer/mdviewer.dart';

// Serialization coverage for the v0.8.0 options (extraCss, codeHeader)
// and the v0.9.0 options (headingAnchors, nested parser object), plus
// host-dylib integration renders proving the Dart → options JSON → FFI
// path lands the options in the output page.
void main() {
  test('extraCss serializes under exactly "extraCss" when set', () {
    final json = const MdvOptions(extraCss: 'body{font-size:117%}').toJson();
    expect(json, isNotNull);
    final map = jsonDecode(json!) as Map<String, dynamic>;
    expect(map, {'extraCss': 'body{font-size:117%}'});
  });

  test('codeHeader serializes under exactly "codeHeader" when set', () {
    final json = const MdvOptions(codeHeader: true).toJson();
    expect(json, isNotNull);
    final map = jsonDecode(json!) as Map<String, dynamic>;
    expect(map, {'codeHeader': true});
  });

  test('codeHeader: false still serializes (explicit set, not default)', () {
    final json = const MdvOptions(codeHeader: false).toJson();
    expect(json, isNotNull);
    expect(jsonDecode(json!), {'codeHeader': false});
  });

  test('all-null options (new fields included) still serialize to null', () {
    expect(const MdvOptions().toJson(), isNull);
  });

  test('new fields compose with existing ones in one payload', () {
    final json = const MdvOptions(
      theme: 'dark',
      extraCss: '.x{}',
      codeHeader: true,
    ).toJson();
    expect(jsonDecode(json!), {
      'theme': 'dark',
      'extraCss': '.x{}',
      'codeHeader': true,
    });
  });

  test('host dylib: extraCss is appended to the page and codeHeader emits the '
      'md-code header markup', () {
    final page = Mdviewer.instance.render(
      '# Hi\n\n```shell\nls -la\n```\n',
      options: const MdvOptions(
        extraCss: 'body{font-size:117%}',
        codeHeader: true,
      ),
    );
    expect(page, contains('body{font-size:117%}'));
    expect(page, contains('class="md-code"'));
    expect(page, contains('md-code-lang'));
    expect(page, contains('md-code-copy'));
    expect(page, contains('shell'));
  });

  test('parser options serialize nested under exactly "parser" with exact '
      'boundary key names, only when set', () {
    final json = const MdvOptions(
      headingAnchors: false,
      parser: MdvParserOptions(
        commonmarkOnly: true,
        tables: true,
        wikiLinks: false,
      ),
    ).toJson();
    expect(json, isNotNull);
    expect(jsonDecode(json!), {
      'headingAnchors': false,
      'parser': {'commonmarkOnly': true, 'tables': true, 'wikiLinks': false},
    });
    // Unset nested fields never serialize; an all-null parser object is
    // an explicit empty {"parser":{}} (boundary default behavior).
    expect(jsonDecode(const MdvOptions(parser: MdvParserOptions()).toJson()!), {
      'parser': <String, dynamic>{},
    });
  });

  test('host dylib: parser.wikiLinks=false renders [[x]] literally and '
      'headingAnchors=false drops heading ids', () {
    final page = Mdviewer.instance.render(
      '# Hi\n\n[[Wiki Page]]\n',
      options: const MdvOptions(
        fragment: true,
        headingAnchors: false,
        parser: MdvParserOptions(wikiLinks: false),
      ),
    );
    expect(page, contains('[[Wiki Page]]'));
    expect(page, isNot(contains('<a ')));
    expect(page, contains('<h1>'));
    expect(page, isNot(contains('id=')));
  });
}
