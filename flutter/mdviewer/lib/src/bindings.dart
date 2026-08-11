import 'dart:ffi' as ffi;

typedef _RenderC =
    ffi.Int Function(
      ffi.Pointer<ffi.Char> md,
      ffi.Size mdLen,
      ffi.Pointer<ffi.Char> optsJson,
      ffi.Pointer<ffi.Pointer<ffi.Char>> outHtml,
      ffi.Pointer<ffi.Size> outLen,
      ffi.Pointer<ffi.Pointer<ffi.Char>> outErr,
    );
typedef RenderDart =
    int Function(
      ffi.Pointer<ffi.Char>,
      int,
      ffi.Pointer<ffi.Char>,
      ffi.Pointer<ffi.Pointer<ffi.Char>>,
      ffi.Pointer<ffi.Size>,
      ffi.Pointer<ffi.Pointer<ffi.Char>>,
    );

// mdv_resolver_fn: int (*)(int kind, const char* target, size_t target_len,
//                          void* userdata, char** out_url, size_t* out_url_len)
typedef ResolverFnC =
    ffi.Int Function(
      ffi.Int kind,
      ffi.Pointer<ffi.Char> target,
      ffi.Size targetLen,
      ffi.Pointer<ffi.Void> userdata,
      ffi.Pointer<ffi.Pointer<ffi.Char>> outUrl,
      ffi.Pointer<ffi.Size> outUrlLen,
    );

typedef _RenderRC =
    ffi.Int Function(
      ffi.Pointer<ffi.Char> md,
      ffi.Size mdLen,
      ffi.Pointer<ffi.Char> optsJson,
      ffi.Pointer<ffi.NativeFunction<ResolverFnC>> resolver,
      ffi.Pointer<ffi.Void> userdata,
      ffi.Pointer<ffi.Pointer<ffi.Char>> outHtml,
      ffi.Pointer<ffi.Size> outLen,
      ffi.Pointer<ffi.Pointer<ffi.Char>> outErr,
    );
typedef RenderRDart =
    int Function(
      ffi.Pointer<ffi.Char>,
      int,
      ffi.Pointer<ffi.Char>,
      ffi.Pointer<ffi.NativeFunction<ResolverFnC>>,
      ffi.Pointer<ffi.Void>,
      ffi.Pointer<ffi.Pointer<ffi.Char>>,
      ffi.Pointer<ffi.Size>,
      ffi.Pointer<ffi.Pointer<ffi.Char>>,
    );

/// Resolved function pointers for the thirteen `libmdviewer` C symbols
/// (see `ffi/README.md` for the ABI contract).
class MdvBindings {
  MdvBindings(ffi.DynamicLibrary lib)
    : render = lib.lookupFunction<_RenderC, RenderDart>('mdv_render'),
      parse = lib.lookupFunction<_RenderC, RenderDart>('mdv_parse'),
      renderDoc = lib.lookupFunction<_RenderC, RenderDart>('mdv_render_doc'),
      renderTree = lib.lookupFunction<_RenderC, RenderDart>('mdv_render_tree'),
      renderTreeDoc = lib.lookupFunction<_RenderC, RenderDart>(
        'mdv_render_tree_doc',
      ),
      renderR = lib.lookupFunction<_RenderRC, RenderRDart>('mdv_render_r'),
      renderDocR = lib.lookupFunction<_RenderRC, RenderRDart>(
        'mdv_render_doc_r',
      ),
      renderTreeR = lib.lookupFunction<_RenderRC, RenderRDart>(
        'mdv_render_tree_r',
      ),
      renderTreeDocR = lib.lookupFunction<_RenderRC, RenderRDart>(
        'mdv_render_tree_doc_r',
      ),
      asset = lib
          .lookupFunction<
            ffi.Int Function(
              ffi.Pointer<ffi.Char>,
              ffi.Pointer<ffi.Pointer<ffi.Char>>,
              ffi.Pointer<ffi.Size>,
              ffi.Pointer<ffi.Pointer<ffi.Char>>,
            ),
            int Function(
              ffi.Pointer<ffi.Char>,
              ffi.Pointer<ffi.Pointer<ffi.Char>>,
              ffi.Pointer<ffi.Size>,
              ffi.Pointer<ffi.Pointer<ffi.Char>>,
            )
          >('mdv_asset'),
      alloc = lib
          .lookupFunction<
            ffi.Pointer<ffi.Void> Function(ffi.Size),
            ffi.Pointer<ffi.Void> Function(int)
          >('mdv_alloc'),
      free = lib
          .lookupFunction<
            ffi.Void Function(ffi.Pointer<ffi.Char>),
            void Function(ffi.Pointer<ffi.Char>)
          >('mdv_free'),
      version = lib
          .lookupFunction<
            ffi.Pointer<ffi.Char> Function(),
            ffi.Pointer<ffi.Char> Function()
          >('mdv_version');

  final RenderDart render;
  final RenderDart parse;
  final RenderDart renderDoc;
  final RenderDart renderTree;
  final RenderDart renderTreeDoc;
  final RenderRDart renderR;
  final RenderRDart renderDocR;
  final RenderRDart renderTreeR;
  final RenderRDart renderTreeDocR;
  final int Function(
    ffi.Pointer<ffi.Char>,
    ffi.Pointer<ffi.Pointer<ffi.Char>>,
    ffi.Pointer<ffi.Size>,
    ffi.Pointer<ffi.Pointer<ffi.Char>>,
  )
  asset;
  final ffi.Pointer<ffi.Void> Function(int) alloc;
  final void Function(ffi.Pointer<ffi.Char>) free;
  final ffi.Pointer<ffi.Char> Function() version;
}
