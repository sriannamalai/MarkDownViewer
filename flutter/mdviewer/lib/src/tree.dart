/// Typed Dart model of the version-1 native render tree — the
/// layout-free, fully RESOLVED semantic tree `Mdviewer.renderTree` /
/// `Mdviewer.renderTreeDoc` return, with everything policy-heavy (URL
/// resolution and filtering, raw-HTML sanitizing, admonition titles,
/// footnote pairing) already applied library-side.
///
/// [MdvTree.fromMap] parses the raw wire map (as returned by
/// `Mdviewer.renderTreeRaw`) strictly: a structural violation of the
/// version-1 schema — a missing required field, a wrong type — throws a
/// [FormatException] naming the offending path (e.g.
/// `blocks[2].items[0].task`). The ONE tolerance is forward
/// compatibility in kind space: a block or inline whose `kind` this
/// model does not know decodes as [MdvUnknownBlock] / [MdvUnknownInline]
/// carrying the raw map, never a throw — a newer library adding node
/// kinds must not break an older host.
library;

/// Source position of a node — the same shape (and the same omission
/// rule: null when the parser recorded no position) as the document
/// codec's spans. Lines are 1-based; offsets are 0-based byte offsets
/// into the markdown source forming the half-open range
/// [startOffset, endOffset).
class MdvSpan {
  const MdvSpan({
    required this.startLine,
    required this.endLine,
    required this.startOffset,
    required this.endOffset,
  });

  final int startLine;
  final int endLine;
  final int startOffset;
  final int endOffset;
}

/// Table column alignment (the wire's `"none"|"left"|"center"|"right"`).
enum MdvAlignment { none, left, center, right }

/// One syntax-highlight token of a code block: a slice of the code text
/// tagged with chroma's canonical token-type name (e.g. `Keyword`) —
/// the same names the `highlight-*.json` assets key their colors by.
class MdvTokenRun {
  const MdvTokenRun({required this.text, required this.tokenType});

  final String text;
  final String tokenType;
}

/// A block-level node of the render tree.
///
/// Sealed: exhaustive `switch`es over the version-1 kinds plus
/// [MdvUnknownBlock] (the forward-compatibility case) cover every value.
sealed class MdvBlock {
  const MdvBlock({required this.id, this.span});

  /// Stable block identity for host-side diffing / itemized rebuilds.
  ///
  /// Via `renderTree`, blocks with source spans get content hashes over
  /// their source bytes; spanless blocks — and every block via
  /// `renderTreeDoc`, where no source is at hand — fall back to
  /// deterministic positional ids. Content-hash ids mean identity IS
  /// content: two byte-identical source blocks share the same id BY
  /// DESIGN, so hosts needing unique keys must disambiguate equal ids
  /// by position — key by `(id, occurrenceIndex)`, never by id alone.
  final String id;

  /// Source position, or null when the node carries none.
  final MdvSpan? span;
}

/// An inline node of the render tree.
///
/// Sealed: exhaustive `switch`es over the version-1 kinds plus
/// [MdvUnknownInline] (the forward-compatibility case) cover every
/// value.
sealed class MdvInline {
  const MdvInline({this.span});

  /// Source position, or null when the node carries none.
  final MdvSpan? span;
}

/* ---- Blocks -------------------------------------------------------- */

/// A section heading (wire kind `heading`).
class MdvHeading extends MdvBlock {
  const MdvHeading({
    required super.id,
    super.span,
    required this.level,
    this.anchorId,
    required this.children,
  });

  /// 1-6.
  final int level;

  /// Slugified per-document-unique anchor; null when `headingAnchors`
  /// is off or the document carries none.
  final String? anchorId;

  final List<MdvInline> children;
}

/// A run of inline content (wire kind `paragraph`).
class MdvParagraph extends MdvBlock {
  const MdvParagraph({required super.id, super.span, required this.children});

  final List<MdvInline> children;
}

/// Quoted block content (wire kind `blockQuote`).
class MdvBlockQuote extends MdvBlock {
  const MdvBlockQuote({required super.id, super.span, required this.blocks});

  final List<MdvBlock> blocks;
}

/// A callout box (wire kind `admonition`). [title] is the derived
/// display title ("Note", …) — the SAME title the HTML renderer shows.
class MdvAdmonition extends MdvBlock {
  const MdvAdmonition({
    required super.id,
    super.span,
    required this.variant,
    required this.title,
    required this.blocks,
  });

  /// Normalized variant: note|tip|important|warning|caution.
  final String variant;

  /// Derived display title ("Note", …).
  final String title;

  final List<MdvBlock> blocks;
}

/// One list entry. Not itself an [MdvBlock]: items exist only inside an
/// [MdvList], but they carry [id]/[span] like blocks do (same identity
/// contract as [MdvBlock.id], duplicates included).
class MdvListItem {
  const MdvListItem({
    required this.id,
    this.span,
    required this.task,
    required this.blocks,
  });

  /// Same identity contract as [MdvBlock.id].
  final String id;

  final MdvSpan? span;

  /// Tri-state, always present on the wire: null for an ordinary item,
  /// the checked state (true/false) for a task-list item.
  final bool? task;

  final List<MdvBlock> blocks;
}

/// An ordered or unordered list (wire kind `list`).
class MdvList extends MdvBlock {
  const MdvList({
    required super.id,
    super.span,
    required this.ordered,
    required this.start,
    required this.tight,
    required this.items,
  });

  final bool ordered;

  /// First item's number; meaningful when [ordered].
  final int start;

  final bool tight;
  final List<MdvListItem> items;
}

/// A fenced or indented code block (wire kind `codeBlock`). [text]
/// always carries the full literal source; [runs] is null when
/// highlighting is off or no token runs are available (unknown/missing
/// language), otherwise an array whose [MdvTokenRun.text] fields
/// concatenate to exactly [text].
class MdvCodeBlock extends MdvBlock {
  const MdvCodeBlock({
    required super.id,
    super.span,
    required this.language,
    required this.label,
    required this.runs,
    required this.text,
  });

  /// Fence info-string language tag, `''` if none.
  final String language;

  /// Display label (the fence tag, or `code` when unlabeled).
  final String label;

  final List<MdvTokenRun>? runs;
  final String text;
}

/// Display math for the host's math engine (wire kind `mathBlock`).
class MdvMathBlock extends MdvBlock {
  const MdvMathBlock({
    required super.id,
    super.span,
    required this.source,
    required this.engine,
  });

  /// TeX source, without delimiters.
  final String source;

  /// `katex` under the version-1 schema.
  final String engine;
}

/// A diagram definition for the host's diagram engine (wire kind
/// `diagram`).
class MdvDiagram extends MdvBlock {
  const MdvDiagram({
    required super.id,
    super.span,
    required this.source,
    required this.engine,
  });

  final String source;

  /// `mermaid` under the version-1 schema.
  final String engine;
}

/// A pipe table (wire kind `table`). A cell is its inline content
/// (`List<MdvInline>`).
class MdvTable extends MdvBlock {
  const MdvTable({
    required super.id,
    super.span,
    required this.alignments,
    required this.header,
    required this.rows,
  });

  /// One entry per column.
  final List<MdvAlignment> alignments;

  /// The header row's cells.
  final List<List<MdvInline>> header;

  /// The body rows.
  final List<List<List<MdvInline>>> rows;
}

/// A horizontal rule (wire kind `thematicBreak`).
class MdvThematicBreak extends MdvBlock {
  const MdvThematicBreak({required super.id, super.span});
}

/// A raw HTML block (wire kind `htmlBlock`). Without `allowRawHTML`,
/// [html] is the bluemonday-sanitized form and [unsafe] is false; with
/// it, [html] is the raw source markup and [unsafe] is true — the
/// content was NOT sanitized.
class MdvHtmlBlock extends MdvBlock {
  const MdvHtmlBlock({
    required super.id,
    super.span,
    required this.html,
    required this.unsafe,
  });

  final String html;
  final bool unsafe;
}

/// A definition list (wire kind `definitionList`): [MdvDefinitionTerm] /
/// [MdvDefinitionDesc] blocks in source order (the pairing stays
/// positional, mirroring the document model).
class MdvDefinitionList extends MdvBlock {
  const MdvDefinitionList({
    required super.id,
    super.span,
    required this.blocks,
  });

  final List<MdvBlock> blocks;
}

/// The term half of a definition-list entry (wire kind
/// `definitionTerm`).
class MdvDefinitionTerm extends MdvBlock {
  const MdvDefinitionTerm({
    required super.id,
    super.span,
    required this.children,
  });

  final List<MdvInline> children;
}

/// The description half of a definition-list entry (wire kind
/// `definitionDesc`).
class MdvDefinitionDesc extends MdvBlock {
  const MdvDefinitionDesc({
    required super.id,
    super.span,
    required this.blocks,
  });

  final List<MdvBlock> blocks;
}

/// One resolved footnote definition in the envelope's `footnotes` array
/// (wire kind `footnoteDef`): [index] matches the [MdvFootnoteRef]
/// inlines that cite it, and [count] is the number of such sites (for
/// hosts rendering per-reference backlinks).
class MdvFootnoteDef extends MdvBlock {
  const MdvFootnoteDef({
    required super.id,
    super.span,
    required this.index,
    required this.count,
    required this.blocks,
  });

  /// 1-based, matches [MdvFootnoteRef.index].
  final int index;

  /// Number of reference sites citing this definition.
  final int count;

  final List<MdvBlock> blocks;
}

/// A block whose `kind` this model does not know — the forward-compat
/// escape hatch (never a throw): [raw] carries the complete wire map;
/// [id]/[span] are decoded best-effort ([id] is `''` when the map
/// carries none).
class MdvUnknownBlock extends MdvBlock {
  const MdvUnknownBlock({
    required super.id,
    super.span,
    required this.kind,
    required this.raw,
  });

  /// The unrecognized wire kind.
  final String kind;

  /// The complete raw wire map, untouched.
  final Map<String, dynamic> raw;
}

/* ---- Inlines ------------------------------------------------------- */

/// A literal text run (wire kind `text`; emoji shortcodes already
/// substituted by the parser).
class MdvText extends MdvInline {
  const MdvText({super.span, required this.value});

  final String value;
}

/// An intra-paragraph line break rendering as whitespace (wire kind
/// `softBreak`). Never carries a span.
class MdvSoftBreak extends MdvInline {
  const MdvSoftBreak({super.span});
}

/// An explicit line break (wire kind `hardBreak`). Never carries a span.
class MdvHardBreak extends MdvInline {
  const MdvHardBreak({super.span});
}

/// Emphasized (italic) content (wire kind `emphasis`).
class MdvEmphasis extends MdvInline {
  const MdvEmphasis({super.span, required this.children});

  final List<MdvInline> children;
}

/// Strongly-emphasized (bold) content (wire kind `strong`).
class MdvStrong extends MdvInline {
  const MdvStrong({super.span, required this.children});

  final List<MdvInline> children;
}

/// Struck-through content (wire kind `strikethrough`).
class MdvStrikethrough extends MdvInline {
  const MdvStrikethrough({super.span, required this.children});

  final List<MdvInline> children;
}

/// Inline code (wire kind `codeSpan`).
class MdvCodeSpan extends MdvInline {
  const MdvCodeSpan({super.span, required this.value});

  final String value;
}

/// A hyperlink (wire kind `link`) with its destination fully resolved:
/// resolver trust, default resolution, and safe-URL filtering have
/// already run. A policy-blocked destination has [url] `''` with
/// [blocked] true — distinct from an empty-but-allowed destination
/// ([url] `''`, [blocked] false). Resolver-returned URLs are
/// host-trusted and carried verbatim. A wiki link resolves INTO a link
/// with [source] `wikiLink` (and the default `.md` fallback when the
/// resolver declines); [source] is null for an ordinary link.
class MdvLink extends MdvInline {
  const MdvLink({
    super.span,
    required this.url,
    this.blocked = false,
    this.title,
    this.source,
    required this.children,
  });

  /// Always present on the wire (possibly empty).
  final String url;

  /// True when the destination was blocked by URL policy.
  final bool blocked;

  final String? title;

  /// `wikiLink` when resolved from `[[…]]`, else null.
  final String? source;

  final List<MdvInline> children;
}

/// An image (wire kind `image`), destination resolved by the same
/// pipeline as [MdvLink] (a blocked destination yields [url] `''` +
/// [blocked] true; a declined/absent resolver takes the default
/// resolution path).
class MdvImage extends MdvInline {
  const MdvImage({
    super.span,
    required this.url,
    this.blocked = false,
    required this.alt,
    this.title,
  });

  final String url;
  final bool blocked;
  final String alt;
  final String? title;
}

/// Inline math for the host's math engine (wire kind `mathInline`).
/// [display] marks a `$$…$$` span rendering as display math despite
/// sitting inline.
class MdvMathInline extends MdvInline {
  const MdvMathInline({super.span, required this.source, this.display = false});

  /// TeX source, without delimiters.
  final String source;

  final bool display;
}

/// A raw inline HTML span (wire kind `htmlInline`), sanitized/flagged
/// exactly like [MdvHtmlBlock].
class MdvHtmlInline extends MdvInline {
  const MdvHtmlInline({super.span, required this.html, required this.unsafe});

  final String html;
  final bool unsafe;
}

/// A footnote reference site (wire kind `footnoteRef`); [index] pairs
/// it with the [MdvFootnoteDef] carrying the same index. Never carries
/// a span.
class MdvFootnoteRef extends MdvInline {
  const MdvFootnoteRef({super.span, required this.index});

  /// 1-based, matches [MdvFootnoteDef.index].
  final int index;
}

/// An inline whose `kind` this model does not know — the forward-compat
/// escape hatch (never a throw): [raw] carries the complete wire map.
class MdvUnknownInline extends MdvInline {
  const MdvUnknownInline({super.span, required this.kind, required this.raw});

  /// The unrecognized wire kind.
  final String kind;

  /// The complete raw wire map, untouched.
  final Map<String, dynamic> raw;
}

/* ---- Envelope + parsing -------------------------------------------- */

/// The version-1 native render tree envelope.
///
/// Block ids ([MdvBlock.id], [MdvListItem.id]) are the basis for
/// host-side diffing and itemized rebuilds, with one caveat baked into
/// the schema: content-hash ids mean two byte-identical source blocks
/// share the same id BY DESIGN, so hosts needing unique keys (e.g.
/// `ListView.builder` keys) must disambiguate equal ids by position —
/// key by `(id, occurrenceIndex)`, never by id alone.
class MdvTree {
  const MdvTree({
    required this.version,
    required this.blocks,
    required this.footnotes,
  });

  /// Parses a raw render-tree wire map (as returned by
  /// `Mdviewer.renderTreeRaw` / `renderTreeDocRaw`) into the typed
  /// model.
  ///
  /// Strict where the version-1 schema promises: a missing required
  /// field or a wrong type throws a [FormatException] naming the
  /// offending path (e.g. `blocks[2].items[0].task`), as does an
  /// envelope version other than 1. Tolerant ONLY in kind space: an
  /// unknown block or inline `kind` decodes as [MdvUnknownBlock] /
  /// [MdvUnknownInline] carrying the raw map — never a throw.
  factory MdvTree.fromMap(Map<String, dynamic> map) {
    final version = _field<int>(map, 'version', '');
    if (version != 1) {
      throw FormatException(
        'version: unsupported render-tree version $version '
        '(this model understands version 1)',
      );
    }
    final blocksRaw = _field<List<dynamic>>(map, 'blocks', '');
    final footnotesRaw = _field<List<dynamic>>(map, 'footnotes', '');
    return MdvTree(
      version: version,
      blocks: [
        for (var i = 0; i < blocksRaw.length; i++)
          _parseBlock(blocksRaw[i], 'blocks[$i]'),
      ],
      footnotes: [
        for (var i = 0; i < footnotesRaw.length; i++)
          _parseBlock(footnotesRaw[i], 'footnotes[$i]'),
      ],
    );
  }

  /// Always 1 (the only version [MdvTree.fromMap] accepts).
  final int version;

  /// Top-level blocks, in document order.
  final List<MdvBlock> blocks;

  /// Footnote definitions, in first-reference order. Under the
  /// version-1 schema every entry is an [MdvFootnoteDef]; an entry of a
  /// future unknown kind decodes as [MdvUnknownBlock] rather than
  /// throwing (the same forward-compat rule as [blocks]).
  final List<MdvBlock> footnotes;
}

Never _fail(String path, String message) =>
    throw FormatException('$path: $message');

String _join(String path, String key) => path.isEmpty ? key : '$path.$key';

String _typeName(Object? v) => v == null ? 'null' : v.runtimeType.toString();

Map<String, dynamic> _asMap(Object? v, String path) {
  if (v is Map<String, dynamic>) return v;
  _fail(path, 'expected an object, got ${_typeName(v)}');
}

T _field<T>(Map<String, dynamic> map, String key, String path) {
  final v = map[key];
  if (v is T) return v;
  if (!map.containsKey(key)) _fail(_join(path, key), 'missing required field');
  _fail(_join(path, key), 'expected $T, got ${_typeName(v)}');
}

T? _optField<T>(Map<String, dynamic> map, String key, String path) {
  final v = map[key];
  if (v == null) return null;
  if (v is T) return v;
  _fail(_join(path, key), 'expected $T, got ${_typeName(v)}');
}

MdvSpan? _optSpan(Map<String, dynamic> node, String path) {
  final v = node['span'];
  if (v == null) return null;
  final m = _asMap(v, '$path.span');
  return MdvSpan(
    startLine: _field<int>(m, 'startLine', '$path.span'),
    endLine: _field<int>(m, 'endLine', '$path.span'),
    startOffset: _field<int>(m, 'startOffset', '$path.span'),
    endOffset: _field<int>(m, 'endOffset', '$path.span'),
  );
}

/// Best-effort span for unknown-kind nodes, which must never throw.
MdvSpan? _lenientSpan(Map<String, dynamic> node, String path) {
  try {
    return _optSpan(node, path);
  } on FormatException {
    return null;
  }
}

List<MdvInline> _inlines(Map<String, dynamic> node, String key, String path) {
  final v = _field<List<dynamic>>(node, key, path);
  return [
    for (var i = 0; i < v.length; i++)
      _parseInline(v[i], '${_join(path, key)}[$i]'),
  ];
}

List<MdvBlock> _blocks(Map<String, dynamic> node, String key, String path) {
  final v = _field<List<dynamic>>(node, key, path);
  return [
    for (var i = 0; i < v.length; i++)
      _parseBlock(v[i], '${_join(path, key)}[$i]'),
  ];
}

MdvAlignment _alignment(Object? v, String path) {
  return switch (v) {
    'none' => MdvAlignment.none,
    'left' => MdvAlignment.left,
    'center' => MdvAlignment.center,
    'right' => MdvAlignment.right,
    _ => _fail(
      path,
      'expected "none"|"left"|"center"|"right", '
      'got ${v is String ? '"$v"' : _typeName(v)}',
    ),
  };
}

List<List<MdvInline>> _cellRow(Object? v, String path) {
  if (v is! List<dynamic>) {
    _fail(path, 'expected an array of cells, got ${_typeName(v)}');
  }
  return [for (var i = 0; i < v.length; i++) _cell(v[i], '$path[$i]')];
}

List<MdvInline> _cell(Object? v, String path) {
  if (v is! List<dynamic>) {
    _fail(path, 'expected a cell (array of inlines), got ${_typeName(v)}');
  }
  return [for (var i = 0; i < v.length; i++) _parseInline(v[i], '$path[$i]')];
}

MdvListItem _parseListItem(Object? v, String path) {
  final node = _asMap(v, path);
  if (!node.containsKey('task')) {
    _fail('$path.task', 'missing required field');
  }
  final task = node['task'];
  if (task != null && task is! bool) {
    _fail('$path.task', 'expected bool or null, got ${_typeName(task)}');
  }
  return MdvListItem(
    id: _field<String>(node, 'id', path),
    span: _optSpan(node, path),
    task: task,
    blocks: _blocks(node, 'blocks', path),
  );
}

List<MdvTokenRun>? _parseRuns(Map<String, dynamic> node, String path) {
  if (!node.containsKey('runs')) {
    _fail('$path.runs', 'missing required field');
  }
  final v = node['runs'];
  if (v == null) return null;
  if (v is! List<dynamic>) {
    _fail('$path.runs', 'expected an array or null, got ${_typeName(v)}');
  }
  final runs = <MdvTokenRun>[];
  for (var i = 0; i < v.length; i++) {
    final run = _asMap(v[i], '$path.runs[$i]');
    runs.add(
      MdvTokenRun(
        text: _field<String>(run, 'text', '$path.runs[$i]'),
        tokenType: _field<String>(run, 'tokenType', '$path.runs[$i]'),
      ),
    );
  }
  return runs;
}

MdvBlock _parseBlock(Object? v, String path) {
  final node = _asMap(v, path);
  final kind = _field<String>(node, 'kind', path);
  switch (kind) {
    case 'heading':
      return MdvHeading(
        id: _field<String>(node, 'id', path),
        span: _optSpan(node, path),
        level: _field<int>(node, 'level', path),
        anchorId: _optField<String>(node, 'anchorId', path),
        children: _inlines(node, 'children', path),
      );
    case 'paragraph':
      return MdvParagraph(
        id: _field<String>(node, 'id', path),
        span: _optSpan(node, path),
        children: _inlines(node, 'children', path),
      );
    case 'blockQuote':
      return MdvBlockQuote(
        id: _field<String>(node, 'id', path),
        span: _optSpan(node, path),
        blocks: _blocks(node, 'blocks', path),
      );
    case 'admonition':
      return MdvAdmonition(
        id: _field<String>(node, 'id', path),
        span: _optSpan(node, path),
        variant: _field<String>(node, 'variant', path),
        title: _field<String>(node, 'title', path),
        blocks: _blocks(node, 'blocks', path),
      );
    case 'list':
      final items = _field<List<dynamic>>(node, 'items', path);
      return MdvList(
        id: _field<String>(node, 'id', path),
        span: _optSpan(node, path),
        ordered: _field<bool>(node, 'ordered', path),
        start: _field<int>(node, 'start', path),
        tight: _field<bool>(node, 'tight', path),
        items: [
          for (var i = 0; i < items.length; i++)
            _parseListItem(items[i], '$path.items[$i]'),
        ],
      );
    case 'codeBlock':
      return MdvCodeBlock(
        id: _field<String>(node, 'id', path),
        span: _optSpan(node, path),
        language: _field<String>(node, 'language', path),
        label: _field<String>(node, 'label', path),
        runs: _parseRuns(node, path),
        text: _field<String>(node, 'text', path),
      );
    case 'mathBlock':
      return MdvMathBlock(
        id: _field<String>(node, 'id', path),
        span: _optSpan(node, path),
        source: _field<String>(node, 'source', path),
        engine: _field<String>(node, 'engine', path),
      );
    case 'diagram':
      return MdvDiagram(
        id: _field<String>(node, 'id', path),
        span: _optSpan(node, path),
        source: _field<String>(node, 'source', path),
        engine: _field<String>(node, 'engine', path),
      );
    case 'table':
      final alignments = _field<List<dynamic>>(node, 'alignments', path);
      final rows = _field<List<dynamic>>(node, 'rows', path);
      return MdvTable(
        id: _field<String>(node, 'id', path),
        span: _optSpan(node, path),
        alignments: [
          for (var i = 0; i < alignments.length; i++)
            _alignment(alignments[i], '$path.alignments[$i]'),
        ],
        header: _cellRow(
          _field<List<dynamic>>(node, 'header', path),
          '$path.header',
        ),
        rows: [
          for (var i = 0; i < rows.length; i++)
            _cellRow(rows[i], '$path.rows[$i]'),
        ],
      );
    case 'thematicBreak':
      return MdvThematicBreak(
        id: _field<String>(node, 'id', path),
        span: _optSpan(node, path),
      );
    case 'htmlBlock':
      return MdvHtmlBlock(
        id: _field<String>(node, 'id', path),
        span: _optSpan(node, path),
        html: _field<String>(node, 'html', path),
        unsafe: _field<bool>(node, 'unsafe', path),
      );
    case 'definitionList':
      return MdvDefinitionList(
        id: _field<String>(node, 'id', path),
        span: _optSpan(node, path),
        blocks: _blocks(node, 'blocks', path),
      );
    case 'definitionTerm':
      return MdvDefinitionTerm(
        id: _field<String>(node, 'id', path),
        span: _optSpan(node, path),
        children: _inlines(node, 'children', path),
      );
    case 'definitionDesc':
      return MdvDefinitionDesc(
        id: _field<String>(node, 'id', path),
        span: _optSpan(node, path),
        blocks: _blocks(node, 'blocks', path),
      );
    case 'footnoteDef':
      return MdvFootnoteDef(
        id: _field<String>(node, 'id', path),
        span: _optSpan(node, path),
        index: _field<int>(node, 'index', path),
        count: _field<int>(node, 'count', path),
        blocks: _blocks(node, 'blocks', path),
      );
    default:
      final id = node['id'];
      return MdvUnknownBlock(
        id: id is String ? id : '',
        span: _lenientSpan(node, path),
        kind: kind,
        raw: node,
      );
  }
}

MdvInline _parseInline(Object? v, String path) {
  final node = _asMap(v, path);
  final kind = _field<String>(node, 'kind', path);
  switch (kind) {
    case 'text':
      return MdvText(
        span: _optSpan(node, path),
        value: _field<String>(node, 'value', path),
      );
    case 'softBreak':
      return MdvSoftBreak(span: _optSpan(node, path));
    case 'hardBreak':
      return MdvHardBreak(span: _optSpan(node, path));
    case 'emphasis':
      return MdvEmphasis(
        span: _optSpan(node, path),
        children: _inlines(node, 'children', path),
      );
    case 'strong':
      return MdvStrong(
        span: _optSpan(node, path),
        children: _inlines(node, 'children', path),
      );
    case 'strikethrough':
      return MdvStrikethrough(
        span: _optSpan(node, path),
        children: _inlines(node, 'children', path),
      );
    case 'codeSpan':
      return MdvCodeSpan(
        span: _optSpan(node, path),
        value: _field<String>(node, 'value', path),
      );
    case 'link':
      return MdvLink(
        span: _optSpan(node, path),
        url: _field<String>(node, 'url', path),
        blocked: _optField<bool>(node, 'blocked', path) ?? false,
        title: _optField<String>(node, 'title', path),
        source: _optField<String>(node, 'source', path),
        children: _inlines(node, 'children', path),
      );
    case 'image':
      return MdvImage(
        span: _optSpan(node, path),
        url: _field<String>(node, 'url', path),
        blocked: _optField<bool>(node, 'blocked', path) ?? false,
        alt: _field<String>(node, 'alt', path),
        title: _optField<String>(node, 'title', path),
      );
    case 'mathInline':
      return MdvMathInline(
        span: _optSpan(node, path),
        source: _field<String>(node, 'source', path),
        display: _optField<bool>(node, 'display', path) ?? false,
      );
    case 'htmlInline':
      return MdvHtmlInline(
        span: _optSpan(node, path),
        html: _field<String>(node, 'html', path),
        unsafe: _field<bool>(node, 'unsafe', path),
      );
    case 'footnoteRef':
      return MdvFootnoteRef(
        span: _optSpan(node, path),
        index: _field<int>(node, 'index', path),
      );
    default:
      return MdvUnknownInline(
        span: _lenientSpan(node, path),
        kind: kind,
        raw: node,
      );
  }
}
