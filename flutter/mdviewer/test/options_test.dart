import 'dart:convert';

import 'package:flutter_test/flutter_test.dart';
import 'package:mdviewer/mdviewer.dart';

// Serialization coverage for the v0.8.0 options (extraCss, codeHeader),
// plus one host-dylib integration render proving the Dart → options JSON →
// FFI path lands both options in the output page.
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
}
