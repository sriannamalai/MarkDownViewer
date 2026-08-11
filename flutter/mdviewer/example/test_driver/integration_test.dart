// Host-side driver for `flutter drive`: saves screenshots the on-device
// integration test captures. Destination directory comes from
// MDV_SHOT_DIR (defaults to ./screenshots).
import 'dart:io';

import 'package:integration_test/integration_test_driver_extended.dart';

Future<void> main() async {
  await integrationDriver(
    onScreenshot: (name, bytes, [args]) async {
      final dir = Platform.environment['MDV_SHOT_DIR'] ?? 'screenshots';
      File('$dir/$name.png')
        ..createSync(recursive: true)
        ..writeAsBytesSync(bytes);
      return true;
    },
  );
}
