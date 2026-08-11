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

Mermaid and KaTeX are vendored (embedded via `go:embed` in the public
`assets` package) for fully offline, self-contained output. To bump a
pinned version, edit `MERMAID_VERSION` / `KATEX_VERSION` in
`scripts/fetch-assets.sh`, then run:

```bash
./scripts/fetch-assets.sh
```

This re-fetches the pinned files, re-inlines KaTeX fonts as `data:` URIs,
and refreshes `third_party/*/LICENSE`. Commit the resulting
`assets/*` and `third_party/*/LICENSE` changes together, and update
`third_party/README.md`'s version table.

`frontMatterFenceTerminated` in `parser/parser.go` mirrors the
closing-fence grammar of `go.abhg.dev/goldmark/frontmatter` (currently
pinned at v0.3.0) — it has to independently detect whether a front-matter
block was actually closed, because upstream's own parse state doesn't
expose that distinction. Any version bump of that dependency requires
re-verifying the mirror against upstream's `parse.go` and running the
canary test in `parser/frontmatter_test.go`
(`TestUpstreamFrontmatterStillSwallowsUnterminated`), which fails loudly
if upstream's swallow-on-unterminated behavior ever changes.

## FFI (libmdviewer)

`./scripts/build-ffi.sh` builds the C-shared library for your platform
into `dist/ffi/<os>-<arch>/` (needs a C compiler; `CGO_ENABLED=1` is set
by the script). The C harness is the ABI's test suite:

```bash
DIR="dist/ffi/$(go env GOOS)-$(go env GOARCH)"
gcc examples/c/harness.c -I"$DIR" -L"$DIR" -lmdviewer -o harness
DYLD_LIBRARY_PATH="$DIR" ./harness   # LD_LIBRARY_PATH on Linux
# on Windows (Git Bash): PATH="$DIR:$PATH" ./harness.exe
```

CI builds the library and runs the harness on all three OSes. If you
change any `mdv_*` signature or the options JSON, update `ffi/README.md`
(packaged into release artifacts), `examples/c/harness.c`, and
`examples/dart/` together — they are the boundary's consumers.

## Releasing

Releases are cut by the maintainer (tag/publish steps are gated on
explicit approval). The flow, in order:

1. Cut the `CHANGELOG.md` entry for the version and land it on `main`.
2. Push the signed `v<ver>` tag and publish the GitHub release —
   `.github/workflows/release-ffi.yml` builds and attaches the release
   artifacts (desktop C-shared zips, WASM npm package, iOS xcframework
   and Android zips).
3. Run the post-release E2E checks against the shipped artifacts.
4. Append the release's mobile-zip SHA-256 checksums to
   `flutter/mdviewer/tool/checksums.txt` and bump the plugin's
   `flutter/mdviewer/pubspec.yaml` version, in one commit (until then,
   `tool/fetch_binaries.sh` refuses to download the new artifacts).
5. Tag that checksums+pubspec commit `flutter-v<ver>` and push it:

   ```bash
   git tag flutter-v<ver> && git push origin flutter-v<ver>
   ```

   This is a standing step: submodule/git consumers of the Flutter
   plugin pin `flutter-v<ver>` — the first commit able to fetch that
   release's verified binaries — never a raw SHA.

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
