#!/usr/bin/env bash
# Fetches release-pinned mobile binaries (checksummed via tool/checksums.txt)
# from a markdownviewer GitHub release and installs them into the plugin's
# platform dirs. Consumer counterpart of build_binaries.sh, which builds
# them locally from source instead of downloading a release artifact.
#
# Zip naming matches .github/workflows/release-ffi.yml exactly:
#   libmdviewer-<ver>-ios.xcframework.zip   (ver has no leading "v")
#   libmdviewer-<ver>-android.zip
# while the release tag — and therefore the download URL — keeps the "v".
#
# Version selection: MDVIEWER_VERSION (with or without a leading "v")
# overrides; otherwise the newest version pinned in tool/checksums.txt is
# fetched. Not every plugin release ships new binaries, so the pubspec
# version may have no artifacts of its own — the checksums file is the
# source of truth for what is fetchable. A version absent from the file
# (via override) is still refused as unpinned.
set -euo pipefail
cd "$(dirname "$0")/.."   # plugin root (flutter/mdviewer)

CHECKSUMS="tool/checksums.txt"

# Newest pinned version: checksums.txt entries are appended per release
# (see the file's header), so the last data line is always the newest.
# Its zip name is "libmdviewer-<ver>-ios.xcframework.zip" or
# "libmdviewer-<ver>-android.zip"; stripping that prefix and suffix
# leaves the version (works even if <ver> ever contains dashes).
newest_pinned_version() {
  awk '$0 !~ /^#/ && NF { last=$2 } END { if (last == "") exit 1; print last }' "$CHECKSUMS" \
    | sed -E 's/^libmdviewer-//; s/-(ios\.xcframework|android)\.zip$//'
}

VERSION="${MDVIEWER_VERSION:-$(newest_pinned_version)}"
VERSION="${VERSION#v}"
BASE_URL="${MDVIEWER_RELEASE_URL:-https://github.com/sriannamalai/markdownviewer/releases/download/v${VERSION}}"
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

# Downloads $1, verifies its checksum, unzips it into a scratch dir, and
# prints that scratch dir's path so the caller can pick out just the
# binary it needs (release zips also bundle LICENSE/README.md alongside it).
fetch_and_verify() {
  local zip="$1" sha extract
  sha="$(checksum_for "$zip")" || {
    echo "error: no checksum entry for '$zip' in $CHECKSUMS — refusing to download an unpinned artifact" >&2
    exit 1
  }
  echo "fetching $zip ..." >&2
  curl -fsSL "$BASE_URL/$zip" -o "$tmp/$zip"
  echo "$sha  $tmp/$zip" | shasum -a 256 -c - >&2
  extract="$tmp/${zip%.zip}"
  mkdir -p "$extract"
  unzip -q -o "$tmp/$zip" -d "$extract"
  echo "$extract"
}

rm -rf ios/Frameworks/libmdviewer.xcframework android/src/main/jniLibs
mkdir -p ios/Frameworks android/src/main

ios_extract="$(fetch_and_verify "libmdviewer-${VERSION}-ios.xcframework.zip")"
cp -R "$ios_extract/libmdviewer.xcframework" ios/Frameworks/

android_extract="$(fetch_and_verify "libmdviewer-${VERSION}-android.zip")"
cp -R "$android_extract/jniLibs" android/src/main/jniLibs

touch "$MARKER"
echo "installed plugin binaries from release v${VERSION}"
