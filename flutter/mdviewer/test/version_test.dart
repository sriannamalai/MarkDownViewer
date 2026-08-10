import 'package:flutter_test/flutter_test.dart';
import 'package:mdviewer/mdviewer.dart';

void main() {
  test('version returns a non-empty string', () {
    expect(Mdviewer.instance.version, isNotEmpty);
  });
}
