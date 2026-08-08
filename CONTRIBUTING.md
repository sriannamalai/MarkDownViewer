# Contributing to MarkDownViewer

Thanks for your interest in contributing. This document covers how to build,
test, and submit changes.

## Development setup

Requires Go 1.26+.

```bash
git clone https://github.com/sriannamalai/markdownviewer.git
cd markdownviewer
go build ./...
go test ./...
```

## Running tests

```bash
go test ./...              # full suite
go test -race ./...        # with the race detector (required before PR)
go vet ./...
gofmt -l .                 # must print nothing
```

Coverage is gated in CI at 75% total. Check locally with:

```bash
go test -coverprofile=/tmp/cover.out ./...
./scripts/check-coverage.sh /tmp/cover.out 75
```

## Golden-file tests

`render/html` and `parser` compare output against checked-in golden files
under `testdata/`. When you intentionally change rendering or parsing
behavior, regenerate them and eyeball the diff before committing:

```bash
go test ./... -update
git diff --stat  # review every changed golden file by hand
```

Never regenerate golden files to paper over an unintended behavior change —
if a diff surprises you, treat it as a bug first.

## Upgrading vendored assets

Mermaid and KaTeX are vendored (embedded via `go:embed` in
`internal/assets/`) for fully offline, self-contained output. To bump a
pinned version, edit `MERMAID_VERSION` / `KATEX_VERSION` in
`scripts/fetch-assets.sh`, then run:

```bash
./scripts/fetch-assets.sh
```

This re-fetches the pinned files, re-inlines KaTeX fonts as `data:` URIs,
and refreshes `third_party/*/LICENSE`. Commit the resulting
`internal/assets/*` and `third_party/*/LICENSE` changes together, and update
`third_party/README.md`'s version table.

## Developer Certificate of Origin (DCO)

All contributions must be signed off, certifying you wrote the change or
otherwise have the right to submit it under the project license (see
[developercertificate.org](https://developercertificate.org/)):

```bash
git commit -s -m "your commit message"
```

This adds a `Signed-off-by: Your Name <you@example.com>` trailer. PRs
without sign-off will not be merged.

## Commit style

Please use [Conventional Commits](https://www.conventionalcommits.org/)
style commit messages (`fix:`, `feat:`, `chore:`, `docs:`, etc.) — it keeps
history scannable and makes changelog generation possible later.

## Pull requests

- Keep PRs focused; unrelated cleanups belong in a separate PR.
- All CI checks (test matrix, `go vet`, `staticcheck`, coverage gate) must
  pass before merge.
- Add or update tests for behavior changes. New Markdown features should
  include CommonMark/GFM-style examples and, where relevant, a fuzz seed.
- If you touch the sanitizer or URL-scheme allowlist in `render/html`,
  call that out explicitly in the PR description — those changes get
  extra scrutiny (see `SECURITY.md`).

## License

By contributing, you agree your contributions are licensed under the
project's [Apache-2.0 license](LICENSE).
