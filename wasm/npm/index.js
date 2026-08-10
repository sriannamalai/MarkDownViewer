// ESM wrapper for the libmdviewer wasm build. Works in browsers and
// Node >= 20 without a bundler. See README.md for usage.
import './wasm_exec.js'; // side effect: defines globalThis.Go

let loadPromise = null;

async function instantiate(wasmSource, importObject) {
  if (wasmSource instanceof WebAssembly.Module) {
    return WebAssembly.instantiate(wasmSource, importObject);
  }
  if (wasmSource instanceof ArrayBuffer || ArrayBuffer.isView(wasmSource)) {
    const { instance } = await WebAssembly.instantiate(wasmSource, importObject);
    return instance;
  }
  const url = wasmSource instanceof URL ? wasmSource : new URL(wasmSource, import.meta.url);
  if (url.protocol === 'file:') {
    // Node: fetch() does not support file URLs.
    const { readFile } = await import('node:fs/promises');
    const buf = await readFile(url);
    const { instance } = await WebAssembly.instantiate(buf, importObject);
    return instance;
  }
  const resp = await fetch(url);
  if (WebAssembly.instantiateStreaming) {
    const { instance } = await WebAssembly.instantiateStreaming(resp, importObject);
    return instance;
  }
  const { instance } = await WebAssembly.instantiate(await resp.arrayBuffer(), importObject);
  return instance;
}

function unwrap(res) {
  if (res.error !== null && res.error !== undefined) throw new Error(res.error);
  return res.value;
}

// Splits {resolver, ...rest} and serializes rest to the strict version-1
// options JSON (undefined options -> null, i.e. library defaults).
function splitOptions(options) {
  if (options === undefined || options === null) return { json: null, resolver: null };
  if (typeof options !== 'object') throw new TypeError('options must be an object');
  const { resolver = null, ...rest } = options;
  if (resolver !== null && typeof resolver !== 'function') {
    throw new TypeError('options.resolver must be a function');
  }
  const json = Object.keys(rest).length === 0 ? null : JSON.stringify(rest);
  return { json, resolver };
}

/**
 * Load the wasm module (singleton). wasmSource: URL string, URL,
 * ArrayBuffer, Uint8Array, WebAssembly.Module, or omitted for
 * ./mdviewer.wasm next to this file.
 */
export function loadMdviewer(wasmSource) {
  if (loadPromise) return loadPromise;
  loadPromise = (async () => {
    const go = new globalThis.Go();
    const ready = new Promise((resolve) => {
      globalThis.__libmdviewer_onready = resolve;
    });
    const instance = await instantiate(
      wasmSource ?? new URL('./mdviewer.wasm', import.meta.url),
      go.importObject,
    );
    go.run(instance); // resolves only on Go exit; do not await
    await ready;
    delete globalThis.__libmdviewer_onready;
    const raw = globalThis.__libmdviewer;
    return {
      version: () => raw.version(),
      render(md, options) {
        const { json, resolver } = splitOptions(options);
        return unwrap(raw.render(String(md), json, resolver));
      },
      parse(md, options) {
        const { json, resolver } = splitOptions(options);
        if (resolver) throw new TypeError('parse does not take a resolver (resolution is a render-time concern)');
        return JSON.parse(unwrap(raw.parse(String(md), json)));
      },
      renderDoc(doc, options) {
        const { json, resolver } = splitOptions(options);
        const docJSON = typeof doc === 'string' ? doc : JSON.stringify(doc);
        return unwrap(raw.renderDoc(docJSON, json, resolver));
      },
      asset(name) {
        return unwrap(raw.asset(String(name)));
      },
    };
  })();
  return loadPromise;
}
