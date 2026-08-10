import 'dart:convert';
import 'dart:ffi' as ffi;
import 'dart:typed_data';

import 'package:ffi/ffi.dart' as pkgffi;

import 'bindings.dart';
import 'exceptions.dart';
import 'library_loader.dart';
import 'options.dart';

/// Entry point for the mdviewer FFI plugin.
class Mdviewer {
  Mdviewer._(this._b);
  static Mdviewer? _instance;
  static Mdviewer get instance =>
      _instance ??= Mdviewer._(MdvBindings(openMdviewerLibrary()));
  final MdvBindings _b;

  /// Library version (embedded at build time).
  String get version => _b.version().cast<pkgffi.Utf8>().toDartString();

  /// Renders [markdown] to sanitized, self-contained HTML (a full page by
  /// default; see [MdvOptions.fragment]). Throws [MdviewerException] on a
  /// boundary error (bad options JSON, malformed input, etc.) with the
  /// exact boundary message.
  String render(String markdown, {MdvOptions? options}) {
    // TODO(task-4): route to mdv_render_r via a NativeCallable wrapping
    // options.resolver, removing this guard.
    if (options?.resolver != null) {
      throw UnimplementedError(
        'MdvOptions.resolver is not wired to the boundary yet (Task 4)',
      );
    }
    return _call2(_b.render, markdown, options?.toJson());
  }

  /// Parses [markdown] into a versioned document JSON, decoded to a Dart
  /// map. Feed the result to [renderDoc] to re-render without re-parsing
  /// (e.g. on a theme switch).
  Map<String, dynamic> parse(String markdown, {MdvOptions? options}) {
    final json = _call2(_b.parse, markdown, options?.toJson());
    return jsonDecode(json) as Map<String, dynamic>;
  }

  /// Re-renders a document previously produced by [parse] — either as
  /// the `Map` it decoded to, or as that map's JSON-encoded `String` form
  /// — without re-parsing the source markdown.
  String renderDoc(Object doc, {MdvOptions? options}) {
    // TODO(task-4): route to mdv_render_doc_r via a NativeCallable
    // wrapping options.resolver, removing this guard.
    if (options?.resolver != null) {
      throw UnimplementedError(
        'MdvOptions.resolver is not wired to the boundary yet (Task 4)',
      );
    }
    final docJson = doc is String ? doc : jsonEncode(doc);
    return _call2(_b.renderDoc, docJson, options?.toJson());
  }

  /// Returns an embedded static asset's bytes (see `ffi/README.md`'s
  /// Assets table for the registry of valid [name]s — e.g.
  /// `'mermaid.js'`, `'theme-dark.css'`). Throws [MdviewerException] for
  /// an unknown name, naming the valid set.
  Uint8List asset(String name) {
    final namePtr = name
        .toNativeUtf8(allocator: pkgffi.malloc)
        .cast<ffi.Char>();
    final out = pkgffi.malloc<ffi.Pointer<ffi.Char>>();
    final outLen = pkgffi.malloc<ffi.Size>();
    final outErr = pkgffi.malloc<ffi.Pointer<ffi.Char>>();
    try {
      final rc = _b.asset(namePtr, out, outLen, outErr);
      if (rc != 0) {
        final err = outErr.value;
        final msg = err.address == 0
            ? 'unknown error (code $rc)'
            : err.cast<pkgffi.Utf8>().toDartString();
        if (err.address != 0) _b.free(err);
        throw MdviewerException(msg);
      }
      final resultPtr = out.value;
      // Copy to Dart memory BEFORE mdv_free — the asTypedList view dies
      // with the buffer.
      final bytes = Uint8List.fromList(
        resultPtr.cast<ffi.Uint8>().asTypedList(outLen.value),
      );
      _b.free(resultPtr);
      return bytes;
    } finally {
      pkgffi.malloc.free(namePtr);
      pkgffi.malloc.free(out);
      pkgffi.malloc.free(outLen);
      pkgffi.malloc.free(outErr);
    }
  }

  /// Shared calling convention for the three 6-arg `(in, inLen, optsJson,
  /// &out, &outLen, &err)` symbols (`mdv_render`, `mdv_parse`,
  /// `mdv_render_doc`).
  String _call2(RenderDart fn, String input, String? optsJson) {
    final inBytes = utf8.encode(input);
    final inPtr = pkgffi.malloc<ffi.Uint8>(inBytes.length);
    inPtr.asTypedList(inBytes.length).setAll(0, inBytes);
    final optsPtr = optsJson == null
        ? ffi.Pointer<ffi.Char>.fromAddress(0)
        : optsJson.toNativeUtf8(allocator: pkgffi.malloc).cast<ffi.Char>();
    final out = pkgffi.malloc<ffi.Pointer<ffi.Char>>();
    final outLen = pkgffi.malloc<ffi.Size>();
    final outErr = pkgffi.malloc<ffi.Pointer<ffi.Char>>();
    try {
      final rc = fn(inPtr.cast(), inBytes.length, optsPtr, out, outLen, outErr);
      if (rc != 0) {
        final err = outErr.value;
        final msg = err.address == 0
            ? 'unknown error (code $rc)'
            : err.cast<pkgffi.Utf8>().toDartString();
        if (err.address != 0) _b.free(err);
        throw MdviewerException(msg);
      }
      final resultPtr = out.value;
      // Copy to a Dart String (utf8.decode allocates its own memory)
      // BEFORE mdv_free — the asTypedList view dies with the buffer.
      final result = utf8.decode(
        resultPtr.cast<ffi.Uint8>().asTypedList(outLen.value),
      );
      _b.free(resultPtr);
      return result;
    } finally {
      pkgffi.malloc.free(inPtr);
      if (optsPtr.address != 0) pkgffi.malloc.free(optsPtr);
      pkgffi.malloc.free(out);
      pkgffi.malloc.free(outLen);
      pkgffi.malloc.free(outErr);
    }
  }
}
