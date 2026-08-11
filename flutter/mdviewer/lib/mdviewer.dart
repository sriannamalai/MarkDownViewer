/// Markdown viewer library (Go core over dart:ffi): sanitized HTML
/// rendering with themes, mermaid, KaTeX, and syntax highlighting — fully
/// offline. See `ffi/README.md` in the markdownviewer repo for the C ABI
/// this plugin binds.
library;

export 'src/mdviewer_base.dart';
export 'src/options.dart';
export 'src/preresolve.dart';
export 'src/exceptions.dart';
export 'src/tree.dart';
export 'src/version_check.dart';
export 'src/widgets/block_builders.dart'
    show MdvCodeBlockView, MdvCopyButton, buildMdvBlock, mdvAdmonitionColor;
export 'src/widgets/builders.dart';
export 'src/widgets/document_view.dart';
export 'src/widgets/inline_spans.dart';
export 'src/widgets/palette.dart';
