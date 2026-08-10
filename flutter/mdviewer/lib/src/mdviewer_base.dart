import 'package:ffi/ffi.dart' as pkgffi;
import 'bindings.dart';
import 'library_loader.dart';

/// Entry point for the mdviewer FFI plugin.
///
/// This task provides construction and [version] only; rendering (Task 3)
/// is added on top of the same singleton.
class Mdviewer {
  Mdviewer._(this._b);
  static Mdviewer? _instance;
  static Mdviewer get instance =>
      _instance ??= Mdviewer._(MdvBindings(openMdviewerLibrary()));
  final MdvBindings _b;

  /// Library version (embedded at build time).
  String get version => _b.version().cast<pkgffi.Utf8>().toDartString();
}
