#!/usr/bin/env bash
# Builds the js/wasm binary and assembles the npm-ready layout into
# dist/wasm/npm/. VERSION env overrides the embedded version (default:
# git describe); a leading "v" is stripped.
set -euo pipefail
cd "$(dirname "$0")/.."

version="${VERSION:-$(git describe --tags --always)}"
version="${version#v}"

out="dist/wasm/npm"
mkdir -p "$out"

GOOS=js GOARCH=wasm go build -trimpath \
  -ldflags "-s -w -X main.version=${version}" \
  -o "$out/mdviewer.wasm" ./wasm

cp "$(go env GOROOT)/lib/wasm/wasm_exec.js" "$out/"
cp wasm/npm/index.js wasm/npm/index.d.ts wasm/npm/README.md "$out/"
cp LICENSE "$out/"
node -e "
  const fs = require('fs');
  const p = JSON.parse(fs.readFileSync('wasm/npm/package.json', 'utf8'));
  p.version = '${version}';
  fs.writeFileSync('${out}/package.json', JSON.stringify(p, null, 2) + '\n');
"
echo "built ${out} (version ${version})"
