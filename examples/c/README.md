# C harness for libmdviewer

Smoke-tests every exported `mdv_*` symbol against a locally built library.
CI runs this on Linux, macOS, and Windows.

## Build and run

From the repository root:

    ./scripts/build-ffi.sh
    DIR="dist/ffi/$(go env GOOS)-$(go env GOARCH)"
    gcc examples/c/harness.c -I"$DIR" -L"$DIR" -lmdviewer -o harness

`gcc` resolves on all three platforms (on macOS it is the clang shim; on Windows use MinGW gcc from Git Bash) — any C99 compiler works.

Then run it with the library directory on the loader path:

    # Linux
    LD_LIBRARY_PATH="$DIR" ./harness
    # macOS
    DYLD_LIBRARY_PATH="$DIR" ./harness
    # Windows (git bash)
    PATH="$DIR:$PATH" ./harness.exe
