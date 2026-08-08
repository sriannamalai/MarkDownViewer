# Security Policy

## Supported versions

MarkDownViewer is currently pre-1.0 / early v0.x. Only the latest released
`v0.x` version is supported with security fixes. Once v1.0 ships, this
policy will be updated with a formal support matrix.

## Reporting a vulnerability

Please report security issues privately — **do not open a public GitHub
issue**. Email:

**mail@sriannamalai.com**

Include:
- A description of the issue and its potential impact.
- Steps to reproduce (a minimal Markdown input that triggers the issue is
  ideal).
- The version/commit you tested against.

We aim to acknowledge reports within a few business days and follow a
**90-day coordinated disclosure** timeline: we will work with you to
validate, fix, and release a patch before any public disclosure, extending
only by mutual agreement if a fix genuinely needs more time.

## Scope

This library renders untrusted Markdown (and, depending on options, raw
HTML) into HTML meant to be displayed in a browser or webview. The
following are explicitly in scope and of particular interest:

- **Sanitizer bypasses**: any input that produces script execution, style
  injection, or other XSS-capable output through the default (safe) render
  path, i.e. without `AllowRawHTML()` / `-unsafe`.
- **URL scheme allowlist bypasses**: any input that causes a disallowed
  URL scheme (e.g. `javascript:`, `data:`) to reach rendered output under
  the default policy.
- **Resolver trust boundary violations**: cases where the library itself
  (not a host's `Resolver` implementation) fails to honor its documented
  contracts.
- Denial-of-service via pathological input (parser/renderer hangs or
  unbounded memory growth) is also in scope, though treated with lower
  severity than injection issues.

Issues that only reproduce when `AllowRawHTML()` / `-unsafe` is explicitly
enabled, or that stem entirely from a host-supplied `Resolver` echoing
untrusted input back unexamined (see the `Resolver` trust contract in
`mdviewer.go`), are generally **out of scope** — that mode is documented as
trusting the caller.

## Resource exhaustion

Parsing is not bounded by a wall-clock or work budget, and at least one
input shape is known to be pathological: **deeply nested lists**. goldmark's
list-nesting logic is super-quadratic in nesting depth. Empirically, on
typical hardware, a list nested ~2000 levels deep (about 16KB of input)
takes roughly 6 seconds to parse; ~4000 levels takes more than 30 seconds.
The cost is in goldmark's parser, not in this library's own code, and it
scales with nesting *depth*, not input *size* — so a size cap on the input
does not bound it; a small, deeply-indented file is enough.

**If you render untrusted Markdown, wrap `Parse` / `Render` (or
`parser.Parse` / `parser.ParseWith`) in a wall-clock timeout** — for example
by running the call in a goroutine and racing it against a timer — and treat
a timeout as a rejected input. Context-aware `Parse`/`Render` variants that
support cancellation natively are on the roadmap (see `docs/Design.md`); no
such variant exists yet.

This is treated as a known, documented limitation for v0.1 rather than a
vulnerability to patch: we do not intend to patch goldmark internals or add
heuristics (e.g. guessing a "safe" nesting depth) that would be fragile and
could reject legitimate input. Reports of *other* pathological input shapes
(unbounded memory growth, other superlinear parse/render costs) are still
welcome under the Scope section above.

## Disclosure credit

With your permission, we're happy to credit reporters in release notes.
