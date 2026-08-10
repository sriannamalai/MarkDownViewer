// Pure-Dart unit test for the example's resolver logic. A widget test that
// pumps MdviewerExampleApp would need to load the native mdviewer library
// on the host, which this example doesn't ship for desktop — so we test
// demoResolver directly instead.

import 'package:flutter_test/flutter_test.dart';
import 'package:mdviewer/mdviewer.dart';

import 'package:example/main.dart';

void main() {
  test('demoResolver rewrites logo.png to an inline SVG data URI', () {
    final url = demoResolver(MdvResolveKind.image, 'logo.png');
    expect(url, isNotNull);
    expect(url, startsWith('data:image/svg+xml;base64,'));
  });

  test('demoResolver declines every other target', () {
    expect(demoResolver(MdvResolveKind.link, 'docs/guide.md'), isNull);
    expect(demoResolver(MdvResolveKind.wikiLink, 'Wiki Page'), isNull);
    expect(demoResolver(MdvResolveKind.image, 'other.png'), isNull);
  });
}
