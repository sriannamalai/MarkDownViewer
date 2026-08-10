/** Resolution kinds passed to a resolver. ABI-frozen. */
export type ResolveKind = 0 /* link */ | 1 /* image */ | 2 /* wiki-link */;

/**
 * Host resolver for link/image/wiki-link targets. Return a string to
 * resolve (the URL is trusted verbatim and emitted as-is, bypassing the
 * safe-URL allowlist) or null/undefined to decline (default resolution
 * applies). Throwing fails the render.
 */
export type Resolver = (kind: ResolveKind, target: string) => string | null | undefined;

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
