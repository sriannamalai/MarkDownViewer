#!/usr/bin/env bash
# Fetches release-pinned mobile binaries (checksummed via tool/checksums.txt)
# from a markdownviewer GitHub release and installs them into the plugin's
# platform dirs. Consumer counterpart of build_binaries.sh, which builds
# them locally from source instead of downloading a release artifact.
set -euo pipefail
cd "$(dirname "$0")/.."   # plugin root (flutter/mdviewer)

VERSION="${MDVIEWER_VERSION:-$(sed -n 's/^version: *//p' pubspec.yaml)}"
BASE_URL="${MDVIEWER_RELEASE_URL:-https://github.com/sriannamalai/markdownviewer/releases/download/v${VERSION}}"
CHECKSUMS="tool/checksums.txt"
MARKER=".fetched-binaries-v${VERSION}"

if [[ -f "$MARKER" ]]; then
  echo "binaries for v${VERSION} already fetched (found $MARKER) — remove it to re-fetch"
  exit 0
fi

checksum_for() {
  awk -v z="$1" '$0 !~ /^#/ && $2 == z { print $1; found=1 } END { exit !found }' "$CHECKSUMS"
}

tmp="$(mktemp -d)"
trap 'rm -rf "$tmp"' EXIT

fetch_and_verify() {
  local zip="$1" dest_dir="$2" sha
  sha="$(checksum_for "$zip")" || {
    echo "error: no checksum entry for '$zip' in $CHECKSUMS — refusing to download an unpinned artifact" >&2
    exit 1
  }
  echo "fetching $zip ..."
  curl -fsSL "$BASE_URL/$zip" -o "$tmp/$zip"
  echo "$sha  $tmp/$zip" | shasum -a 256 -c -
  mkdir -p "$dest_dir"
  unzip -q -o "$tmp/$zip" -d "$dest_dir"
}

rm -rf ios/Frameworks/libmdviewer.xcframework android/src/main/jniLibs
mkdir -p ios/Frameworks android/src/main

fetch_and_verify "libmdviewer-ios-v${VERSION}.zip" ios/Frameworks
fetch_and_verify "libmdviewer-android-v${VERSION}.zip" android/src/main/jniLibs

touch "$MARKER"
echo "installed plugin binaries from release v${VERSION}"
