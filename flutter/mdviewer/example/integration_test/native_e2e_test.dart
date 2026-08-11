// On-device E2E for the native render-tree page: pumps the real app,
// walks the document, exercises the copy button and the light/dark
// toggle, and captures screenshots (collected by
// test_driver/integration_test.dart when run under `flutter drive`).
import 'dart:io';

import 'package:example/main.dart' as app;
import 'package:flutter/material.dart';
import 'package:flutter_math_fork/flutter_math.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:integration_test/integration_test.dart';
import 'package:mdviewer/mdviewer.dart';

void main() {
  final binding = IntegrationTestWidgetsFlutterBinding.ensureInitialized();

  // Android needs the Flutter surface converted to an image before
  // screenshots — and the conversion is one-shot (asserting on repeat).
  var surfaceConverted = false;
  Future<void> shot(WidgetTester tester, String name) async {
    if (Platform.isAndroid && !surfaceConverted) {
      surfaceConverted = true;
      await binding.convertFlutterSurfaceToImage();
      await tester.pump();
    }
    await binding.takeScreenshot(name);
  }

  Future<void> scrollTo(WidgetTester tester, Finder target) async {
    await tester.scrollUntilVisible(
      target,
      250,
      scrollable: find.byType(Scrollable).first,
    );
    await tester.pump(const Duration(milliseconds: 300));
  }

  testWidgets('native page renders, copies, and theme-toggles on device', (
    tester,
  ) async {
    app.main();
    await tester.pumpAndSettle(const Duration(milliseconds: 500));

    // Native page is the initial tab: rich document top.
    expect(find.text('mdviewer example', findRichText: true), findsOneWidget);
    expect(find.byType(MdvDocumentView), findsOneWidget);
    await shot(tester, 'native-top');

    // Math: block math must be laid out natively (flutter_math_fork).
    await scrollTo(tester, find.byType(Math).first);
    expect(find.byType(Math), findsWidgets);
    await shot(tester, 'native-math');

    // Code block: token runs colored, native Copy button flips to
    // "Copied" via Clipboard.setData.
    await scrollTo(tester, find.text('Copy'));
    expect(find.byType(MdvCodeBlockView), findsOneWidget);
    await tester.tap(find.text('Copy'));
    await tester.pump(const Duration(milliseconds: 200));
    expect(find.text('Copied'), findsOneWidget);
    await shot(tester, 'native-copy-copied');
    // Let the "Copied" feedback window elapse (no pending timers).
    await tester.pump(const Duration(seconds: 3));

    // Admonition + table + task checkboxes further down.
    await scrollTo(tester, find.text('Tip'));
    expect(find.byIcon(Icons.check_box), findsWidgets);
    await shot(tester, 'native-blocks');

    // Dark mode: palette swaps via MdvPalette.load(dark: true).
    await tester.tap(find.byIcon(Icons.dark_mode));
    await tester.pumpAndSettle(const Duration(milliseconds: 500));
    await shot(tester, 'native-dark');

    // Back to light for a clean exit.
    await tester.tap(find.byIcon(Icons.light_mode));
    await tester.pumpAndSettle(const Duration(milliseconds: 500));
  });
}
