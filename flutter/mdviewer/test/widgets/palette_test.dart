import 'dart:ui';

import 'package:flutter/widgets.dart' show TextStyle;
import 'package:flutter_test/flutter_test.dart';
import 'package:mdviewer/mdviewer.dart';

void main() {
  final themeJson = <String, dynamic>{
    'version': 1,
    'mode': 'light',
    'chromaStyle': 'github',
    'vars': {
      '--md-accent': '#0969da',
      '--md-bg': '#ffffff',
      '--md-border': '#d1d9e0',
      '--md-code-bg': '#f6f8fa',
      '--md-fg': '#1f2328',
      '--md-quote-fg': '#59636e',
    },
  };
  final highlightJson = <String, dynamic>{
    'version': 1,
    'style': 'github',
    'colors': {'Keyword': '#cf222e', 'LiteralString': '#0a3069'},
    'styles': {
      'Comment': {'italic': true},
      'Keyword': {'bold': true},
      'GenericUnderline': {'underline': true},
    },
  };

  test('fromJson parses theme vars and highlight colors', () {
    final p = MdvPalette.fromJson(themeJson, highlight: highlightJson);
    expect(p.brightness, Brightness.light);
    expect(p.background, const Color(0xFFFFFFFF));
    expect(p.foreground, const Color(0xFF1F2328));
    expect(p.accent, const Color(0xFF0969DA));
    expect(p.border, const Color(0xFFD1D9E0));
    expect(p.codeBackground, const Color(0xFFF6F8FA));
    expect(p.quoteForeground, const Color(0xFF59636E));
    expect(p.chromaStyle, 'github');
    expect(p.tokenColors['Keyword'], const Color(0xFFCF222E));
  });

  test('tokenColor falls back to foreground for unknown token types', () {
    final p = MdvPalette.fromJson(themeJson, highlight: highlightJson);
    expect(p.tokenColor('Keyword'), const Color(0xFFCF222E));
    expect(p.tokenColor('SomeFutureTokenType'), p.foreground);
    // No highlight JSON at all: every token type falls back.
    final bare = MdvPalette.fromJson(themeJson);
    expect(bare.tokenColors, isEmpty);
    expect(bare.tokenColor('Keyword'), bare.foreground);
  });

  test('fromJson parses the optional highlight styles map', () {
    final p = MdvPalette.fromJson(themeJson, highlight: highlightJson);
    expect(
      p.tokenStyles['Comment'],
      const TextStyle(fontStyle: FontStyle.italic),
    );
    expect(
      p.tokenStyles['Keyword'],
      const TextStyle(fontWeight: FontWeight.bold),
    );
    expect(
      p.tokenStyles['GenericUnderline'],
      const TextStyle(decoration: TextDecoration.underline),
    );
    // tokenTextStyle merges the attributes with the token color; a
    // colored-but-unstyled type carries color only.
    final kw = p.tokenTextStyle('Keyword');
    expect(kw.color, const Color(0xFFCF222E));
    expect(kw.fontWeight, FontWeight.bold);
    final str = p.tokenTextStyle('LiteralString');
    expect(str.color, const Color(0xFF0A3069));
    expect(str.fontWeight, isNull);
    expect(str.fontStyle, isNull);
  });

  test('styles absent is valid (older assets); wrong type throws', () {
    final noStyles = <String, dynamic>{...highlightJson}..remove('styles');
    final p = MdvPalette.fromJson(themeJson, highlight: noStyles);
    expect(p.tokenStyles, isEmpty);
    expect(p.tokenTextStyle('Keyword').color, const Color(0xFFCF222E));
    expect(p.tokenTextStyle('Keyword').fontWeight, isNull);
    expect(
      () => MdvPalette.fromJson(
        themeJson,
        highlight: {...highlightJson, 'styles': 'bold'},
      ),
      throwsFormatException,
    );
  });

  test('fromJson rejects unsupported versions and missing vars', () {
    expect(
      () => MdvPalette.fromJson({...themeJson, 'version': 2}),
      throwsFormatException,
    );
    expect(
      () => MdvPalette.fromJson(themeJson, highlight: {'version': 2}),
      throwsFormatException,
    );
    final missingVar = <String, dynamic>{
      ...themeJson,
      'vars': {'--md-bg': '#ffffff'},
    };
    expect(() => MdvPalette.fromJson(missingVar), throwsFormatException);
  });

  test('parseHexColor handles #rgb, #rrggbb, #rrggbbaa and rejects junk', () {
    expect(parseHexColor('#abc'), const Color(0xFFAABBCC));
    expect(parseHexColor('#0969da'), const Color(0xFF0969DA));
    expect(parseHexColor('0969da'), const Color(0xFF0969DA));
    expect(parseHexColor('#11223344'), const Color(0x44112233));
    expect(() => parseHexColor('#xyz123'), throwsFormatException);
    expect(() => parseHexColor('red'), throwsFormatException);
    expect(() => parseHexColor('#12345'), throwsFormatException);
  });

  test('baked-in light/dark defaults mirror the shipped theme assets', () {
    expect(MdvPalette.light.accent, const Color(0xFF0969DA));
    expect(MdvPalette.dark.background, const Color(0xFF0D1117));
    expect(MdvPalette.dark.brightness, Brightness.dark);
  });

  test('copyWith overrides individual values', () {
    final p = MdvPalette.light.copyWith(accent: const Color(0xFF123456));
    expect(p.accent, const Color(0xFF123456));
    expect(p.background, MdvPalette.light.background);
  });

  test('load(dark: false/true) parses the real assets via the dylib', () async {
    final light = await MdvPalette.load(dark: false);
    expect(light.brightness, Brightness.light);
    expect(light.chromaStyle, 'github');
    expect(light.tokenColors, isNotEmpty);
    expect(light.tokenColors.containsKey('Keyword'), isTrue);

    final dark = await MdvPalette.load(dark: true);
    expect(dark.brightness, Brightness.dark);
    expect(dark.chromaStyle, 'github-dark');
    expect(dark.background, isNot(light.background));

    // The shipped assets carry the styles map: github-dark italicizes
    // comments and bolds function names; github bolds NameLabel.
    expect(dark.tokenStyles['Comment']?.fontStyle, FontStyle.italic);
    expect(dark.tokenStyles['NameFunction']?.fontWeight, FontWeight.bold);
    expect(light.tokenStyles['NameLabel']?.fontWeight, FontWeight.bold);
  });
}
