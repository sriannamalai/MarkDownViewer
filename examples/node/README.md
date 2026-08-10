# Node harness for libmdviewer wasm

Smoke-tests the ESM wrapper (`dist/wasm/npm/index.js`) against a locally
built wasm binary — the Node twin of `examples/c/harness.c`.

## Build and run

From the repository root:

    ./scripts/build-wasm.sh
    node examples/node/harness.mjs

Requires Node >= 20. CI runs this in the `wasm` job on every push/PR,
alongside the C harness.
