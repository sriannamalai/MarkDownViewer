#!/usr/bin/env bash
# Builds mobile binaries from this repository and installs them into the
# plugin's platform dirs. Dev-loop counterpart of fetch_binaries.sh.
set -euo pipefail
cd "$(dirname "$0")/../../.."   # repo root
./scripts/build-mobile.sh all
rm -rf flutter/mdviewer/ios/Frameworks/libmdviewer.xcframework \
       flutter/mdviewer/android/src/main/jniLibs
mkdir -p flutter/mdviewer/ios/Frameworks flutter/mdviewer/android/src/main
cp -R dist/mobile/ios/libmdviewer.xcframework flutter/mdviewer/ios/Frameworks/
cp -R dist/mobile/android/jniLibs flutter/mdviewer/android/src/main/jniLibs
echo "installed plugin binaries from local build"
