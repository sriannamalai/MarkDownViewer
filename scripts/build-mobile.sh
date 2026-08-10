#!/usr/bin/env bash
# Builds libmdviewer for mobile targets into dist/mobile/.
#   build-mobile.sh ios      -> dist/mobile/ios/libmdviewer.xcframework
#   build-mobile.sh android  -> dist/mobile/android/jniLibs/<abi>/libmdviewer.so
#   build-mobile.sh all      -> both
# VERSION env overrides the embedded version (default: git describe);
# a leading "v" is stripped.
set -euo pipefail
cd "$(dirname "$0")/.."

version="${VERSION:-$(git describe --tags --always)}"
version="${version#v}"
ldflags="-s -w -X main.version=${version}"

build_ios() {
  command -v xcrun >/dev/null || { echo "error: xcrun not found (Xcode required for the ios target)" >&2; exit 1; }
  local out=dist/mobile/ios work=dist/mobile/ios/.slices
  rm -rf "$out"
  mkdir -p "$work/device" "$work/sim"

  local dev_sdk sim_sdk clang
  dev_sdk="$(xcrun --sdk iphoneos --show-sdk-path)"
  sim_sdk="$(xcrun --sdk iphonesimulator --show-sdk-path)"
  clang="$(xcrun --sdk iphoneos -f clang)"

  CGO_ENABLED=1 GOOS=ios GOARCH=arm64 \
    CC="$clang -target arm64-apple-ios13.0 -isysroot $dev_sdk" \
    go build -trimpath -buildmode=c-archive -ldflags "$ldflags" \
    -o "$work/device/libmdviewer.a" ./ffi
  CGO_ENABLED=1 GOOS=ios GOARCH=arm64 \
    CC="$clang -target arm64-apple-ios13.0-simulator -isysroot $sim_sdk" \
    go build -trimpath -buildmode=c-archive -ldflags "$ldflags" \
    -o "$work/sim/libmdviewer.a" ./ffi

  # c-archive writes libmdviewer.h next to the .a. -headers must point at
  # a dir containing ONLY the header (xcodebuild copies its entire
  # contents into the slice's Headers/), so move each header out into its
  # own include/ dir before invoking -create-xcframework.
  test -f "$work/device/libmdviewer.h" || { echo "error: c-archive did not emit $work/device/libmdviewer.h" >&2; exit 1; }
  test -f "$work/sim/libmdviewer.h" || { echo "error: c-archive did not emit $work/sim/libmdviewer.h" >&2; exit 1; }
  mkdir -p "$work/device/include" "$work/sim/include"
  mv "$work/device/libmdviewer.h" "$work/device/include/libmdviewer.h"
  mv "$work/sim/libmdviewer.h" "$work/sim/include/libmdviewer.h"

  xcodebuild -create-xcframework \
    -library "$work/device/libmdviewer.a" -headers "$work/device/include" \
    -library "$work/sim/libmdviewer.a" -headers "$work/sim/include" \
    -output "$out/libmdviewer.xcframework"
  rm -rf "$work"
  echo "built $out/libmdviewer.xcframework (version ${version})"
}

build_android() {
  local ndk="${ANDROID_NDK_HOME:-${ANDROID_NDK_LATEST_HOME:-}}"
  if [[ -z "$ndk" ]]; then
    for base in "$HOME/Library/Android/sdk/ndk" "${ANDROID_HOME:-$HOME/Library/Android/sdk}/ndk"; do
      [[ -d "$base" ]] && ndk="$(command ls -d "$base"/* 2>/dev/null | sort -V | tail -1)" && break
    done
  fi
  [[ -n "$ndk" && -d "$ndk" ]] || { echo "error: Android NDK not found (set ANDROID_NDK_HOME)" >&2; exit 1; }
  local host
  case "$(uname -s)" in
    Darwin) host=darwin-x86_64 ;;   # NDK ships x86_64 host tools; they run under Rosetta on arm64 Macs
    *)      host=linux-x86_64 ;;
  esac
  local tc="$ndk/toolchains/llvm/prebuilt/$host/bin"
  [[ -d "$tc" ]] || { echo "error: NDK toolchain dir not found: $tc" >&2; exit 1; }

  local out=dist/mobile/android
  rm -rf "$out"
  mkdir -p "$out/jniLibs/arm64-v8a" "$out/jniLibs/x86_64"

  CGO_ENABLED=1 GOOS=android GOARCH=arm64 CC="$tc/aarch64-linux-android24-clang" \
    go build -trimpath -buildmode=c-shared -ldflags "$ldflags" \
    -o "$out/jniLibs/arm64-v8a/libmdviewer.so" ./ffi
  CGO_ENABLED=1 GOOS=android GOARCH=amd64 CC="$tc/x86_64-linux-android24-clang" \
    go build -trimpath -buildmode=c-shared -ldflags "$ldflags" \
    -o "$out/jniLibs/x86_64/libmdviewer.so" ./ffi
  # c-shared writes the header next to the .so; keep one copy at the zip
  # root for reference (content identical per ABI) and keep jniLibs
  # binary-only. Fail loudly if the header didn't materialize.
  test -f "$out/jniLibs/arm64-v8a/libmdviewer.h"
  cp "$out/jniLibs/arm64-v8a/libmdviewer.h" "$out/libmdviewer.h"
  rm -f "$out"/jniLibs/*/libmdviewer.h
  test -f "$out/libmdviewer.h"
  echo "built $out/jniLibs (version ${version})"
}

case "${1:-all}" in
  ios) build_ios ;;
  android) build_android ;;
  all) build_ios; build_android ;;
  *) echo "usage: $0 [ios|android|all]" >&2; exit 2 ;;
esac
