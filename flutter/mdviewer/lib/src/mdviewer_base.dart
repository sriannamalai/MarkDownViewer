import 'dart:convert';
import 'dart:ffi' as ffi;
import 'dart:io';
import 'dart:typed_data';

import 'package:ffi/ffi.dart' as pkgffi;

import 'bindings.dart';
import 'exceptions.dart';
import 'library_loader.dart';
import 'options.dart';
import 'tree.dart';
import 'version_check.dart';

/// Entry point for the mdviewer FFI plugin.
class Mdviewer {
  Mdviewer._(this._b);
  static Mdviewer? _instance;

  /// The singleton instance, created (and its native library loaded) on
  /// first access. That first access also runs the plugin↔library version
  /// handshake — see [checkVersionHandshake] for the contract, the
  /// mismatch [MdviewerException], and the `MDVIEWER_SKIP_VERSION_CHECK=1`
  /// escape hatch. A handshake failure leaves no instance cached, so a
  /// later access (e.g. after fixing the binaries) retries from scratch.
  static Mdviewer get instance {
    final existing = _instance;
    if (existing != null) return existing;
    final created = Mdviewer._(MdvBindings(openMdviewerLibrary()));
    checkVersionHandshake(created.version, environment: Platform.environment);
    return _instance = created;
  }

  final MdvBindings _b;

  /// Library version (embedded at build time).
  String get version => _b.version().cast<pkgffi.Utf8>().toDartString();

  /// Renders [markdown] to sanitized, self-contained HTML (a full page by
  /// default; see [MdvOptions.fragment]). Throws [MdviewerException] on a
  /// boundary error (bad options JSON, malformed input, etc.) with the
  /// exact boundary message.
  String render(String markdown, {MdvOptions? options}) {
    final resolver = options?.resolver;
    if (resolver != null) {
      return _callWithResolver(
        _b.renderR,
        markdown,
        options?.toJson(),
        resolver,
      );
    }
    return _call2(_b.render, markdown, options?.toJson());
  }

  /// Parses [markdown] into a versioned document JSON, decoded to a Dart
  /// map. Feed the result to [renderDoc] to re-render without re-parsing
  /// (e.g. on a theme switch).
  ///
  /// Unlike [render] and [renderDoc], a resolver here is a permanent
  /// error, not a not-yet-wired seam: there is no `mdv_parse_r` symbol —
  /// resolution is a render-time concern, and parsing never resolves.
  Map<String, dynamic> parse(String markdown, {MdvOptions? options}) {
    if (options?.resolver != null) {
      throw ArgumentError(
        'parse does not take a resolver (resolution is a render-time concern)',
      );
    }
    final json = _call2(_b.parse, markdown, options?.toJson());
    return jsonDecode(json) as Map<String, dynamic>;
  }

  /// Re-renders a document previously produced by [parse] — either as
  /// the `Map` it decoded to, or as that map's JSON-encoded `String` form
  /// — without re-parsing the source markdown.
  String renderDoc(Object doc, {MdvOptions? options}) {
    final docJson = doc is String ? doc : jsonEncode(doc);
    final resolver = options?.resolver;
    if (resolver != null) {
      return _callWithResolver(
        _b.renderDocR,
        docJson,
        options?.toJson(),
        resolver,
      );
    }
    return _call2(_b.renderDoc, docJson, options?.toJson());
  }

  /// Builds the version-1 native render tree from [markdown]: the
  /// layout-free, fully RESOLVED semantic tree native hosts render as
  /// platform widgets, returned as the typed [MdvTree] model
  /// ([renderTreeRaw] returns the same tree as its raw wire map).
  ///
  /// Options relevance differs from [render]: `parser`,
  /// `headingAnchors` (anchorId presence), `highlighting` (runs
  /// presence), `math`, `mermaid`, and `allowRawHTML` apply; the
  /// HTML-only fields (`theme`, `themeOverrides`, `fragment`,
  /// `maxWidth`, `sourceMap`, `stylesheet`, `extraCss`, `codeHeader`)
  /// are decoded and ignored — a native render tree has no HTML
  /// page/CSS output to configure, and spans are always included. See
  /// `ffi/README.md`'s options-relevance table. Resolver support is
  /// identical to [render] (trees carry resolved URLs): uses
  /// `mdv_render_tree_r` when [MdvOptions.resolver] is set,
  /// `mdv_render_tree` otherwise. Throws [MdviewerException] on a
  /// boundary error.
  MdvTree renderTree(String markdown, {MdvOptions? options}) =>
      MdvTree.fromMap(renderTreeRaw(markdown, options: options));

  /// [renderTree] returning the raw wire map instead of the typed
  /// model — for hosts that walk the JSON themselves (schema:
  /// `render/tree/tree.go`'s package documentation; `wasm/npm/index.d.ts`
  /// types the same shape).
  Map<String, dynamic> renderTreeRaw(String markdown, {MdvOptions? options}) {
    final resolver = options?.resolver;
    final json = resolver != null
        ? _callWithResolver(
            _b.renderTreeR,
            markdown,
            options?.toJson(),
            resolver,
          )
        : _call2(_b.renderTree, markdown, options?.toJson());
    return jsonDecode(json) as Map<String, dynamic>;
  }

  /// Builds the render tree from a document previously produced by
  /// [parse] — either as the decoded `Map` or its JSON-encoded `String`
  /// form — without re-parsing the source markdown. Same options
  /// relevance as [renderTree], except `parser` is also decoded and
  /// ignored (the document is already parsed, as with [renderDoc]).
  ///
  /// Block ids differ from [renderTree]'s: with no markdown source at
  /// hand for content hashes, every block takes the deterministic
  /// positional fallback id form (see [MdvBlock.id]).
  MdvTree renderTreeDoc(Object doc, {MdvOptions? options}) =>
      MdvTree.fromMap(renderTreeDocRaw(doc, options: options));

  /// [renderTreeDoc] returning the raw wire map instead of the typed
  /// model.
  Map<String, dynamic> renderTreeDocRaw(Object doc, {MdvOptions? options}) {
    final docJson = doc is String ? doc : jsonEncode(doc);
    final resolver = options?.resolver;
    final json = resolver != null
        ? _callWithResolver(
            _b.renderTreeDocR,
            docJson,
            options?.toJson(),
            resolver,
          )
        : _call2(_b.renderTreeDoc, docJson, options?.toJson());
    return jsonDecode(json) as Map<String, dynamic>;
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

  /// Shared calling convention for the five 6-arg `(in, inLen, optsJson,
  /// &out, &outLen, &err)` symbols (`mdv_render`, `mdv_parse`,
  /// `mdv_render_doc`, `mdv_render_tree`, `mdv_render_tree_doc`).
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

  /// Bridges [resolve] across the FFI boundary as a synchronous C callback
  /// (via a per-call [ffi.NativeCallable.isolateLocal]) and calls [fn]
  /// (`mdv_render_r` / `mdv_render_doc_r` / `mdv_render_tree_r` /
  /// `mdv_render_tree_doc_r`) with the same 8-arg out-param convention
  /// [_call2] uses for the plain 6-arg symbols.
  ///
  /// A throwing Dart [resolve] cannot unwind across the native boundary —
  /// that is undefined behavior for a `NativeCallable` — so the callback
  /// catches it, records it in [hostError], and declines (returns `0`)
  /// for every remaining target in this render; declining is always a
  /// contract-valid outcome, so the render itself proceeds (and typically
  /// succeeds) via default resolution for the rest. Once the native call
  /// returns, a non-null [hostError] is rethrown as a [MdviewerException]
  /// regardless of whether the render succeeded or failed on its own —
  /// this reproduces the wasm/JS surface, where a throwing resolver fails
  /// the render outright, even though mechanically on the FFI side the
  /// failing targets were merely declined.
  String _callWithResolver(
    RenderRDart fn,
    String input,
    String? optsJson,
    MdvResolver resolve,
  ) {
    Object? hostError;
    late final ffi.NativeCallable<ResolverFnC> callable;
    callable = ffi.NativeCallable<ResolverFnC>.isolateLocal((
      int kind,
      ffi.Pointer<ffi.Char> target,
      int targetLen,
      ffi.Pointer<ffi.Void> userdata,
      ffi.Pointer<ffi.Pointer<ffi.Char>> outUrl,
      ffi.Pointer<ffi.Size> outUrlLen,
    ) {
      if (hostError != null) return 0; // already failed: decline the rest
      try {
        final t = utf8.decode(target.cast<ffi.Uint8>().asTypedList(targetLen));
        // kind is ABI-frozen 0/1/2 today (link/image/wikiLink; see
        // ffi/README.md), but the C contract allows append-only growth.
        // An index outside the enum we know about must decline rather
        // than crash on MdvResolveKind.values[kind] (RangeError) — a
        // future kind should degrade to "no host opinion", not a fault.
        if (kind < 0 || kind >= MdvResolveKind.values.length) return 0;
        final url = resolve(MdvResolveKind.values[kind], t);
        if (url == null) return 0; // decline: default resolution applies
        final bytes = utf8.encode(url);
        // mdv_alloc: the library takes ownership and frees this buffer
        // itself (including on an invalid-length failure below) — see
        // ffi/README.md's "Memory: mdv_alloc".
        final buf = _b.alloc(bytes.length);
        if (buf.address == 0) return 0; // host allocation failure: decline
        buf.cast<ffi.Uint8>().asTypedList(bytes.length).setAll(0, bytes);
        outUrl.value = buf.cast();
        outUrlLen.value = bytes.length;
        return 1;
      } catch (e) {
        hostError = e;
        return 0; // decline the remainder; rethrown after fn() returns
      }
    }, exceptionalReturn: 0);
    try {
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
        final rc = fn(
          inPtr.cast(),
          inBytes.length,
          optsPtr,
          callable.nativeFunction,
          ffi.Pointer<ffi.Void>.fromAddress(0),
          out,
          outLen,
          outErr,
        );
        if (rc != 0) {
          final err = outErr.value;
          final msg = err.address == 0
              ? 'unknown error (code $rc)'
              : err.cast<pkgffi.Utf8>().toDartString();
          if (err.address != 0) _b.free(err);
          if (hostError != null) {
            throw MdviewerException('resolver threw: $hostError');
          }
          throw MdviewerException(msg);
        }
        final resultPtr = out.value;
        // Copy to a Dart String (utf8.decode allocates its own memory)
        // BEFORE mdv_free — the asTypedList view dies with the buffer.
        final result = utf8.decode(
          resultPtr.cast<ffi.Uint8>().asTypedList(outLen.value),
        );
        _b.free(resultPtr);
        if (hostError != null) {
          throw MdviewerException('resolver threw: $hostError');
        }
        return result;
      } finally {
        pkgffi.malloc.free(inPtr);
        if (optsPtr.address != 0) pkgffi.malloc.free(optsPtr);
        pkgffi.malloc.free(out);
        pkgffi.malloc.free(outLen);
        pkgffi.malloc.free(outErr);
      }
    } finally {
      callable.close(); // per-call callable; must not leak
    }
  }
}
