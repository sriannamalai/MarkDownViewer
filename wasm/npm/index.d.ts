/** Resolution kinds passed to a resolver. ABI-frozen. */
export type ResolveKind = 0 /* link */ | 1 /* image */ | 2 /* wiki-link */;

/**
 * Host resolver for link/image/wiki-link targets. Return a string to
 * resolve (the URL is trusted verbatim and emitted as-is, bypassing the
 * safe-URL allowlist) or null/undefined to decline (default resolution
 * applies). Throwing fails the render.
 */
export type Resolver = (kind: ResolveKind, target: string) => string | null | undefined;

/**
 * Nested parser configuration: which Markdown syntax extensions the
 * parse enables. Strictly decoded like the top level (unknown or
 * wrong-case keys are an error). Omitted entirely = library default
 * (every extension on). `commonmarkOnly: true` starts from pure
 * CommonMark (no extensions) instead; the per-extension booleans are
 * tristate overrides on top of that base — omitted keeps the base's
 * setting, `true` enables, `false` disables. Parse-time only: affects
 * `render` and `parse`, and is decoded but ignored by `renderDoc`
 * (the document is already parsed). Note `parser.math` gates `$x$`
 * syntax recognition at parse time; the top-level `math` option gates
 * KaTeX rendering.
 */
export interface ParserOptions {
  commonmarkOnly?: boolean;
  tables?: boolean;
  strikethrough?: boolean;
  taskLists?: boolean;
  linkify?: boolean;
  footnotes?: boolean;
  definitionLists?: boolean;
  frontMatter?: boolean;
  emoji?: boolean;
  wikiLinks?: boolean;
  math?: boolean;
  admonitions?: boolean;
}

/** Strict version-1 options. Unknown fields are an error. */
export interface Options {
  version?: 1;
  theme?: 'auto' | 'light' | 'dark';
  fragment?: boolean;
  allowRawHTML?: boolean;
  mermaid?: boolean;
  math?: boolean;
  highlighting?: boolean;
  maxWidth?: string;
  sourceMap?: boolean;
  themeOverrides?: Record<string, string>;
  stylesheet?: string;
  /**
   * Extra CSS appended after the base styling (or after `stylesheet`
   * when set). Sanitized like `stylesheet`; full-page output only.
   */
  extraCss?: string;
  /**
   * Wrap code blocks in header markup (language label + copy button).
   * Full pages also get inline copy-to-clipboard JS; fragment hosts
   * wire their own handler. Default false.
   */
  codeHeader?: boolean;
  /**
   * Emit slug `id` attributes on headings (`<h1 id="...">`). Default
   * true; set false to omit them (intra-page #fragment links to
   * headings stop resolving). Render-time: applies to `render` and
   * `renderDoc`.
   */
  headingAnchors?: boolean;
  /** Nested parser configuration; see ParserOptions. */
  parser?: ParserOptions;
  /** Not part of the JSON boundary; stripped and passed as a callback. */
  resolver?: Resolver;
}

/* ---- Native render tree (version-1 wire schema) ------------------- */

/**
 * Source position of a node — the same shape (and the same omission
 * rule: absent when the parser recorded no position) as the document
 * codec's spans. Lines are 1-based; offsets are 0-based byte offsets
 * into the markdown source forming the half-open range
 * [startOffset, endOffset).
 */
export interface Span {
  startLine: number;
  endLine: number;
  startOffset: number;
  endOffset: number;
}

/** Table column alignment. */
export type Alignment = 'none' | 'left' | 'center' | 'right';

/**
 * One syntax-highlight token of a code block: a slice of the code text
 * tagged with chroma's canonical token-type name (e.g. "Keyword") —
 * the same names the `highlight-*.json` assets key their colors by.
 */
export interface TokenRun {
  text: string;
  tokenType: string;
}

/** Literal text run (emoji shortcodes already substituted). */
export interface TextInline {
  kind: 'text';
  span?: Span;
  value: string;
}

/** Intra-paragraph line break rendering as whitespace. Never has a span. */
export interface SoftBreak {
  kind: 'softBreak';
}

/** Explicit line break. Never has a span. */
export interface HardBreak {
  kind: 'hardBreak';
}

export interface Emphasis {
  kind: 'emphasis';
  span?: Span;
  children: Inline[];
}

export interface Strong {
  kind: 'strong';
  span?: Span;
  children: Inline[];
}

export interface Strikethrough {
  kind: 'strikethrough';
  span?: Span;
  children: Inline[];
}

export interface CodeSpan {
  kind: 'codeSpan';
  span?: Span;
  value: string;
}

/**
 * Hyperlink with its destination fully resolved: resolver trust,
 * default resolution, and safe-URL filtering have already run. A
 * policy-blocked destination has `url: ""` with `blocked: true`
 * (distinct from an empty-but-allowed destination, where `blocked` is
 * absent). A wiki link resolves into a Link with `source: "wikiLink"`.
 */
export interface Link {
  kind: 'link';
  span?: Span;
  url: string;
  blocked?: true;
  title?: string;
  source?: 'wikiLink';
  children: Inline[];
}

/** Image, destination resolved by the same pipeline as Link. */
export interface Image {
  kind: 'image';
  span?: Span;
  url: string;
  blocked?: true;
  alt: string;
  title?: string;
}

/**
 * Inline math for the host's math engine. `display: true` marks a
 * `$$…$$` span rendering as display math despite sitting inline.
 */
export interface MathInline {
  kind: 'mathInline';
  span?: Span;
  source: string;
  display?: true;
}

/** Raw inline HTML, sanitized/flagged exactly like HTMLBlock. */
export interface HTMLInline {
  kind: 'htmlInline';
  span?: Span;
  html: string;
  unsafe: boolean;
}

/** Footnote reference site; `index` pairs it with a Footnote. Never has a span. */
export interface FootnoteRef {
  kind: 'footnoteRef';
  index: number;
}

/** Inline node union, discriminated by `kind`. */
export type Inline =
  | TextInline
  | SoftBreak
  | HardBreak
  | Emphasis
  | Strong
  | Strikethrough
  | CodeSpan
  | Link
  | Image
  | MathInline
  | HTMLInline
  | FootnoteRef;

export interface Heading {
  kind: 'heading';
  span?: Span;
  id: string;
  level: number;
  /** Absent when headingAnchors is off or the document carries none. */
  anchorId?: string;
  children: Inline[];
}

export interface Paragraph {
  kind: 'paragraph';
  span?: Span;
  id: string;
  children: Inline[];
}

export interface BlockQuote {
  kind: 'blockQuote';
  span?: Span;
  id: string;
  blocks: Block[];
}

/** Callout box; `title` is the derived display title ("Note", …). */
export interface Admonition {
  kind: 'admonition';
  span?: Span;
  id: string;
  variant: string;
  title: string;
  blocks: Block[];
}

/**
 * One list entry. `task` is always present: null for an ordinary item,
 * the checked state (true/false) for a task-list item.
 */
export interface ListItem {
  span?: Span;
  id: string;
  task: boolean | null;
  blocks: Block[];
}

export interface List {
  kind: 'list';
  span?: Span;
  id: string;
  ordered: boolean;
  /** First item's number; meaningful when `ordered`. */
  start: number;
  tight: boolean;
  items: ListItem[];
}

/**
 * Fenced or indented code block. `text` always carries the full
 * literal source; `runs` is null when highlighting is off or no token
 * runs are available, otherwise an array whose `text` fields
 * concatenate to exactly `text`.
 */
export interface CodeBlock {
  kind: 'codeBlock';
  span?: Span;
  id: string;
  language: string;
  label: string;
  runs: TokenRun[] | null;
  text: string;
}

/** Display math (TeX source, without delimiters). */
export interface MathBlock {
  kind: 'mathBlock';
  span?: Span;
  id: string;
  source: string;
  engine: 'katex';
}

/** Diagram definition for the host's diagram engine. */
export interface Diagram {
  kind: 'diagram';
  span?: Span;
  id: string;
  source: string;
  engine: 'mermaid';
}

/** One table cell: its inline content. */
export type Cell = Inline[];

export interface Table {
  kind: 'table';
  span?: Span;
  id: string;
  /** One entry per column. */
  alignments: Alignment[];
  header: Cell[];
  rows: Cell[][];
}

export interface ThematicBreak {
  kind: 'thematicBreak';
  span?: Span;
  id: string;
}

/**
 * Raw HTML block. Without allowRawHTML, `html` is the sanitized form
 * and `unsafe` is false; with it, `html` is the raw source markup and
 * `unsafe` is true (the content was NOT sanitized).
 */
export interface HTMLBlock {
  kind: 'htmlBlock';
  span?: Span;
  id: string;
  html: string;
  unsafe: boolean;
}

export interface DefinitionList {
  kind: 'definitionList';
  span?: Span;
  id: string;
  /** DefinitionTerm / DefinitionDesc, in source order. */
  blocks: Block[];
}

export interface DefinitionTerm {
  kind: 'definitionTerm';
  span?: Span;
  id: string;
  children: Inline[];
}

export interface DefinitionDesc {
  kind: 'definitionDesc';
  span?: Span;
  id: string;
  blocks: Block[];
}

/**
 * Block node union, discriminated by `kind`.
 *
 * Forward compatibility: the wire schema stays version 1 across additive
 * growth, so a newer library may emit kinds this union does not name.
 * Exhaustive switches must carry a default arm (render nothing or a
 * placeholder) — treat the union as open, not closed.
 */
export type Block =
  | Heading
  | Paragraph
  | BlockQuote
  | Admonition
  | List
  | CodeBlock
  | MathBlock
  | Diagram
  | Table
  | ThematicBreak
  | HTMLBlock
  | DefinitionList
  | DefinitionTerm
  | DefinitionDesc;

/**
 * One resolved footnote definition; `index` matches the FootnoteRef
 * inlines citing it and `count` is the number of such sites.
 */
export interface Footnote {
  kind: 'footnoteDef';
  span?: Span;
  id: string;
  index: number;
  count: number;
  blocks: Block[];
}

/**
 * The version-1 native render tree: a layout-free, fully RESOLVED
 * semantic tree (URL policy, sanitizing, admonition titles, footnote
 * pairing, math/mermaid fallbacks all applied library-side) that hosts
 * render as platform widgets. `blocks` and `footnotes` are always
 * arrays.
 *
 * Block ids: via `renderTree`, blocks with source spans get content
 * hashes over their source bytes; spanless blocks — and every block
 * via `renderTreeDoc`, where no source is at hand — fall back to
 * deterministic positional ids. Content-hash ids mean identity IS
 * content: two byte-identical source blocks share the same id BY
 * DESIGN (the basis for host-side diffing), so hosts needing unique
 * keys must disambiguate equal ids by position — e.g. key by
 * `(id, occurrenceIndex)`.
 */
export interface RenderTree {
  version: 1;
  blocks: Block[];
  footnotes: Footnote[];
}

export interface Mdviewer {
  /** Library version (injected at build). */
  version(): string;
  /** Render markdown to HTML (full page by default; fragment via options). */
  render(md: string, options?: Options): string;
  /** Parse markdown to the version-1 document object. No resolver here. */
  parse(md: string, options?: Omit<Options, 'resolver'>): unknown;
  /** Render a version-1 document (object or JSON string) to HTML. */
  renderDoc(doc: unknown, options?: Options): string;
  /**
   * Build the version-1 native render tree from markdown. Options:
   * `parser`, `headingAnchors`, `highlighting`, `math`, `mermaid`, and
   * `allowRawHTML` apply; the HTML-only fields (`theme`,
   * `themeOverrides`, `fragment`, `maxWidth`, `sourceMap`,
   * `stylesheet`, `extraCss`, `codeHeader`) are decoded and ignored —
   * a native render tree has no HTML page/CSS output to configure, and
   * spans are always included. Resolver support is identical to
   * `render` (trees carry resolved URLs).
   */
  renderTree(md: string, options?: Options): RenderTree;
  /**
   * Build the render tree from a version-1 document (object or JSON
   * string). Same options relevance as `renderTree`, except `parser`
   * is also ignored (the document is already parsed, as with
   * `renderDoc`); block ids take the deterministic positional fallback
   * form (no markdown source at hand for content hashes).
   */
  renderTreeDoc(doc: unknown, options?: Options): RenderTree;
  /** Embedded static asset by registry name (e.g. "mermaid.js", "katex.css", "base.css", "theme-light.css"). */
  asset(name: string): Uint8Array;
}

/**
 * Load the wasm module. Repeat calls return the same promise while a load
 * is in flight and after it succeeds; a failed load is not cached, so a
 * later call retries.
 */
export function loadMdviewer(
  wasmSource?: string | URL | ArrayBuffer | Uint8Array | WebAssembly.Module,
): Promise<Mdviewer>;
