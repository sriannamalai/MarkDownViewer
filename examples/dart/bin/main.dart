// Renders Markdown to HTML through libmdviewer via dart:ffi.
//
// Usage: dart run bin/main.dart [path/to/libmdviewer.{dylib|so|dll}]
// Default library path: ../../dist/ffi/<os>-<arch>/ (see README).
import 'dart:convert';
import 'dart:ffi';
import 'dart:io';
import 'package:ffi/ffi.dart';

typedef MdvRenderC = Int32 Function(
    Pointer<Uint8> md, Size mdLen, Pointer<Utf8> optsJson,
    Pointer<Pointer<Uint8>> outHtml, Pointer<Size> outLen,
    Pointer<Pointer<Utf8>> outErr);
typedef MdvRenderDart = int Function(
    Pointer<Uint8> md, int mdLen, Pointer<Utf8> optsJson,
    Pointer<Pointer<Uint8>> outHtml, Pointer<Size> outLen,
    Pointer<Pointer<Utf8>> outErr);
typedef MdvFreeC = Void Function(Pointer<Void> ptr);
typedef MdvFreeDart = void Function(Pointer<Void> ptr);
typedef MdvVersionC = Pointer<Utf8> Function();

String defaultLibraryPath() {
  // Abi.current().toString() is lowercase snake_case (e.g. 'macos_arm64'),
  // not the 'macosArm64'-style casing the enum names suggest.
  final arch =
      Abi.current().toString().endsWith('arm64') ? 'arm64' : 'amd64';
  if (Platform.isMacOS) return '../../dist/ffi/darwin-$arch/libmdviewer.dylib';
  if (Platform.isWindows) return '../../dist/ffi/windows-$arch/libmdviewer.dll';
  return '../../dist/ffi/linux-$arch/libmdviewer.so';
}

void main(List<String> args) {
  final libPath = args.isNotEmpty ? args.first : defaultLibraryPath();
  final lib = DynamicLibrary.open(libPath);

  final render = lib.lookupFunction<MdvRenderC, MdvRenderDart>('mdv_render');
  final free = lib.lookupFunction<MdvFreeC, MdvFreeDart>('mdv_free');
  final version = lib.lookupFunction<MdvVersionC, MdvVersionC>('mdv_version');

  print('libmdviewer ${version().toDartString()}');

  const markdown = '# Hello from Dart\n\nRendered via `dart:ffi`.\n';
  final mdBytes = utf8.encode(markdown);
  final md = calloc<Uint8>(mdBytes.length);
  md.asTypedList(mdBytes.length).setAll(0, mdBytes);
  final opts = '{"theme": "auto"}'.toNativeUtf8();
  final outHtml = calloc<Pointer<Uint8>>();
  final outLen = calloc<Size>();
  final outErr = calloc<Pointer<Utf8>>();

  try {
    final rc = render(md, mdBytes.length, opts, outHtml, outLen, outErr);
    if (rc != 0) {
      final msg = outErr.value == nullptr ? '(no message)' : outErr.value.toDartString();
      free(outErr.value.cast());
      throw StateError('mdv_render failed: $msg');
    }
    final html = utf8.decode(outHtml.value.asTypedList(outLen.value));
    free(outHtml.value.cast());
    print('rendered ${outLen.value} bytes of HTML');
    // Default rendering embeds the full theme stylesheet before the body,
    // so a short byte-offset prefix would only ever show CSS. Print the
    // full document instead — it's still short enough to be readable and
    // it actually demonstrates the rendered <h1>.
    print(html);
  } finally {
    calloc.free(md); malloc.free(opts);
    calloc.free(outHtml); calloc.free(outLen); calloc.free(outErr);
  }
}
