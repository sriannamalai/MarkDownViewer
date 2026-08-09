# Dart FFI example

Consumes libmdviewer from Dart via `dart:ffi` — the same pattern a Flutter
app uses (bundle the platform library and `DynamicLibrary.open` it).

## Run

From the repository root:

    ./scripts/build-ffi.sh
    cd examples/dart
    dart pub get
    dart run bin/main.dart

Pass an explicit library path as the first argument if yours is elsewhere
(e.g. a downloaded release artifact):

    dart run bin/main.dart /path/to/libmdviewer.dylib

Memory rules: buffers returned by the library are freed with `mdv_free`
(never Dart's allocator); buffers you allocate for inputs are yours to
free. `mdv_version()`'s string is static — don't free it.
