import 'dart:ffi' as ffi;
import 'dart:io';

/// Resolves the libmdviewer dynamic library per platform.
/// Android: bundled jniLibs .so. iOS: statically linked into the process.
/// Anything else (host dev/tests): MDVIEWER_LIBRARY_PATH env, else the
/// repo's `dist/ffi/<os>-<arch>/` dylib found by walking up from CWD.
ffi.DynamicLibrary openMdviewerLibrary() {
  if (Platform.isAndroid) return ffi.DynamicLibrary.open('libmdviewer.so');
  if (Platform.isIOS) return ffi.DynamicLibrary.process();
  final env = Platform.environment['MDVIEWER_LIBRARY_PATH'];
  if (env != null && env.isNotEmpty) return ffi.DynamicLibrary.open(env);
  final osArch = _hostDistDir();
  final ext = Platform.isMacOS ? 'dylib' : 'so';
  var dir = Directory.current;
  for (var i = 0; i < 6; i++) {
    final candidate = File('${dir.path}/dist/ffi/$osArch/libmdviewer.$ext');
    if (candidate.existsSync()) return ffi.DynamicLibrary.open(candidate.path);
    final parent = dir.parent;
    if (parent.path == dir.path) break;
    dir = parent;
  }
  throw StateError(
    'libmdviewer not found: set MDVIEWER_LIBRARY_PATH or build dist/ffi '
    '(scripts/build-ffi.sh) for host development.',
  );
}

/// Maps `dart:ffi`'s [ffi.Abi.current] (e.g. `macos_arm64`) to the
/// `<goos>-<goarch>` directory naming `scripts/build-ffi.sh` uses under
/// `dist/ffi/` (e.g. `darwin-arm64`).
///
/// This is deliberately *not* `Platform.version.contains('arm64')`: that
/// string is Dart/VM build metadata, not an architecture API — it happens
/// to contain the substring "arm64" on this host today, but nothing about
/// its format is a documented contract. `Abi.current()` is the purpose-built
/// source for the running process's OS/architecture.
String _hostDistDir() {
  final parts = ffi.Abi.current().toString().split('_');
  final os = switch (parts[0]) {
    'macos' => 'darwin',
    'linux' => 'linux',
    'windows' => 'windows',
    final other => other,
  };
  final arch = switch (parts[1]) {
    'x64' => 'amd64',
    'ia32' => '386',
    final other => other, // arm64, arm, riscv64, ... pass through unchanged
  };
  return '$os-$arch';
}
