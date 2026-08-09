#!/usr/bin/env bash
# Builds libmdviewer (c-shared) for the current platform into
# dist/ffi/<goos>-<goarch>/. VERSION env overrides the embedded version
# (default: git describe); a leading "v" is stripped.
set -euo pipefail
cd "$(dirname "$0")/.."

goos="$(go env GOOS)"
goarch="$(go env GOARCH)"
version="${VERSION:-$(git describe --tags --always)}"
version="${version#v}"

case "$goos" in
  darwin)  ext=dylib ;;
  windows) ext=dll ;;
  *)       ext=so ;;
esac

out="dist/ffi/${goos}-${goarch}"
mkdir -p "$out"
CGO_ENABLED=1 go build -trimpath -buildmode=c-shared \
  -ldflags "-s -w -X main.version=${version}" \
  -o "${out}/libmdviewer.${ext}" ./ffi
echo "built ${out}/libmdviewer.${ext} (version ${version})"
