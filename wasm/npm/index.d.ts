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

export interface Mdviewer {
  /** Library version (injected at build). */
  version(): string;
  /** Render markdown to HTML (full page by default; fragment via options). */
  render(md: string, options?: Options): string;
  /** Parse markdown to the version-1 document object. No resolver here. */
  parse(md: string, options?: Omit<Options, 'resolver'>): unknown;
  /** Render a version-1 document (object or JSON string) to HTML. */
  renderDoc(doc: unknown, options?: Options): string;
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
