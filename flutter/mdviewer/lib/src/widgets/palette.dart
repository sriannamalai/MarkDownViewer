import 'dart:convert';

import 'package:flutter/widgets.dart';

import '../mdviewer_base.dart';

/// The color palette [MdvDocumentView] renders with, built from the
/// library's `theme-*.json` + `highlight-*.json` assets (the same
/// palettes the HTML renderer's CSS uses, exposed as data so native
/// hosts never parse CSS).
///
/// Obtain one with [MdvPalette.load] (fetches + parses the assets via
/// `Mdviewer.asset`), from raw asset JSON with [MdvPalette.fromJson],
/// or use the baked-in [MdvPalette.light] / [MdvPalette.dark] defaults
/// (snapshots of the same GitHub palettes; no native call needed).
/// Every value is host-overridable via the main constructor or
/// [copyWith].
@immutable
class MdvPalette {
  const MdvPalette({
    required this.brightness,
    required this.background,
    required this.foreground,
    required this.accent,
    required this.border,
    required this.codeBackground,
    required this.quoteForeground,
    this.tokenColors = const {},
    this.chromaStyle = '',
  });

  /// Parses the two version-1 asset JSON maps: `theme` is a decoded
  /// `theme-light.json` / `theme-dark.json` (`{version, mode,
  /// chromaStyle, vars}`), `highlight` an optionally-supplied decoded
  /// `highlight-*.json` (`{version, style, colors}`). Throws
  /// [FormatException] on an unsupported version, a missing `--md-*`
  /// var, or an unparsable hex color; omitting `highlight` just leaves
  /// [tokenColors] empty (every token falls back to [foreground]).
  factory MdvPalette.fromJson(
    Map<String, dynamic> theme, {
    Map<String, dynamic>? highlight,
  }) {
    _checkVersion(theme, 'theme');
    final vars = theme['vars'];
    if (vars is! Map<String, dynamic>) {
      throw const FormatException('theme JSON: missing "vars" object');
    }
    final tokenColors = <String, Color>{};
    var chromaStyle = theme['chromaStyle'] is String
        ? theme['chromaStyle'] as String
        : '';
    if (highlight != null) {
      _checkVersion(highlight, 'highlight');
      final colors = highlight['colors'];
      if (colors is! Map<String, dynamic>) {
        throw const FormatException('highlight JSON: missing "colors" object');
      }
      for (final e in colors.entries) {
        tokenColors[e.key] = parseHexColor(e.value as String);
      }
      if (highlight['style'] is String) {
        chromaStyle = highlight['style'] as String;
      }
    }
    return MdvPalette(
      brightness: theme['mode'] == 'dark' ? Brightness.dark : Brightness.light,
      background: _varColor(vars, '--md-bg'),
      foreground: _varColor(vars, '--md-fg'),
      accent: _varColor(vars, '--md-accent'),
      border: _varColor(vars, '--md-border'),
      codeBackground: _varColor(vars, '--md-code-bg'),
      quoteForeground: _varColor(vars, '--md-quote-fg'),
      tokenColors: tokenColors,
      chromaStyle: chromaStyle,
    );
  }

  /// Fetches `theme-<mode>.json` + `highlight-<mode>.json` through
  /// [Mdviewer.asset] and parses them with [MdvPalette.fromJson].
  ///
  /// Async as API contract (hosts should not depend on the asset fetch
  /// being synchronous), though today's FFI asset call completes
  /// synchronously. [mdviewer] defaults to [Mdviewer.instance].
  static Future<MdvPalette> load({
    required bool dark,
    Mdviewer? mdviewer,
  }) async {
    final mdv = mdviewer ?? Mdviewer.instance;
    final mode = dark ? 'dark' : 'light';
    Map<String, dynamic> fetch(String name) =>
        jsonDecode(utf8.decode(mdv.asset(name))) as Map<String, dynamic>;
    return MdvPalette.fromJson(
      fetch('theme-$mode.json'),
      highlight: fetch('highlight-$mode.json'),
    );
  }

  /// Baked-in light defaults — the same values `theme-light.json`
  /// carries (GitHub light), minus token colors (code renders in
  /// [foreground] until a loaded palette supplies them).
  static const light = MdvPalette(
    brightness: Brightness.light,
    background: Color(0xFFFFFFFF),
    foreground: Color(0xFF1F2328),
    accent: Color(0xFF0969DA),
    border: Color(0xFFD1D9E0),
    codeBackground: Color(0xFFF6F8FA),
    quoteForeground: Color(0xFF59636E),
    chromaStyle: 'github',
  );

  /// Baked-in dark defaults — the same values `theme-dark.json`
  /// carries (GitHub dark), minus token colors.
  static const dark = MdvPalette(
    brightness: Brightness.dark,
    background: Color(0xFF0D1117),
    foreground: Color(0xFFE6EDF3),
    accent: Color(0xFF4493F8),
    border: Color(0xFF30363D),
    codeBackground: Color(0xFF161B22),
    quoteForeground: Color(0xFF8D96A0),
    chromaStyle: 'github-dark',
  );

  /// Light or dark, from the theme JSON's `mode`.
  final Brightness brightness;

  /// Page background (`--md-bg`).
  final Color background;

  /// Default text color (`--md-fg`) — also the fallback for unknown
  /// token types (see [tokenColor]).
  final Color foreground;

  /// Link / accent color (`--md-accent`).
  final Color accent;

  /// Rules, table borders, quote bars (`--md-border`).
  final Color border;

  /// Code block + code span background (`--md-code-bg`).
  final Color codeBackground;

  /// Blockquote text color (`--md-quote-fg`).
  final Color quoteForeground;

  /// Chroma `TokenType.String()` name → color, from
  /// `highlight-*.json`. The asset pre-expands chroma's category
  /// inheritance, so a code run's concrete `tokenType` is always a
  /// direct key here when the style colors it at all.
  final Map<String, Color> tokenColors;

  /// The chroma style name these token colors came from (`github`,
  /// `github-dark`, ...).
  final String chromaStyle;

  /// The color for one code token run: the mapped color, or
  /// [foreground] for a token type this palette does not know —
  /// unknown/missing types must never render invisibly.
  Color tokenColor(String tokenType) => tokenColors[tokenType] ?? foreground;

  MdvPalette copyWith({
    Brightness? brightness,
    Color? background,
    Color? foreground,
    Color? accent,
    Color? border,
    Color? codeBackground,
    Color? quoteForeground,
    Map<String, Color>? tokenColors,
    String? chromaStyle,
  }) {
    return MdvPalette(
      brightness: brightness ?? this.brightness,
      background: background ?? this.background,
      foreground: foreground ?? this.foreground,
      accent: accent ?? this.accent,
      border: border ?? this.border,
      codeBackground: codeBackground ?? this.codeBackground,
      quoteForeground: quoteForeground ?? this.quoteForeground,
      tokenColors: tokenColors ?? this.tokenColors,
      chromaStyle: chromaStyle ?? this.chromaStyle,
    );
  }

  static void _checkVersion(Map<String, dynamic> map, String what) {
    final version = map['version'];
    if (version != 1) {
      throw FormatException(
        '$what JSON: unsupported version $version (expected 1)',
      );
    }
  }

  static Color _varColor(Map<String, dynamic> vars, String name) {
    final v = vars[name];
    if (v is! String) {
      throw FormatException('theme JSON: missing var "$name"');
    }
    return parseHexColor(v);
  }
}

/// Parses `#rrggbb` (what the assets emit), plus `#rgb` and `#rrggbbaa`
/// defensively, into an opaque-by-default [Color]. Throws
/// [FormatException] naming the value otherwise.
Color parseHexColor(String hex) {
  final s = hex.startsWith('#') ? hex.substring(1) : hex;
  final value = int.tryParse(s, radix: 16);
  if (value != null) {
    switch (s.length) {
      case 3:
        final r = (value >> 8) & 0xF, g = (value >> 4) & 0xF, b = value & 0xF;
        return Color(0xFF000000 | r * 0x110000 | g * 0x1100 | b * 0x11);
      case 6:
        return Color(0xFF000000 | value);
      case 8:
        // #rrggbbaa (CSS order) -> ARGB.
        return Color(((value & 0xFF) << 24) | (value >> 8));
    }
  }
  throw FormatException('unparsable hex color "$hex"');
}
