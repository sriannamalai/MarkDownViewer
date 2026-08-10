//go:build js && wasm

// Command wasm builds libmdviewer for GOOS=js GOARCH=wasm. It registers
// its API on globalThis.__libmdviewer for the ESM wrapper (wasm/npm/
// index.js) — hosts use the wrapper, not this object, so its shape may
// change with the wrapper in lockstep. Options/error semantics are the
// same strict version-1 JSON boundary the C ABI uses
// (internal/boundary).
package main

import (
	"fmt"
	"syscall/js"

	"github.com/sriannamalai/markdownviewer/internal/boundary"
	htmlrender "github.com/sriannamalai/markdownviewer/render/html"
)

// version is injected at build time via -ldflags "-X main.version=…".
var version = "dev"

// result marshals a boundary call into {value, error} for the wrapper.
// Panics (including js.Error from a throwing resolver) become errors —
// same containment promise as the C ABI's call envelope.
func result(f func() (any, error)) (out map[string]any) {
	out = map[string]any{"value": nil, "error": nil}
	defer func() {
		if r := recover(); r != nil {
			out["value"] = nil
			if err, ok := r.(error); ok {
				out["error"] = fmt.Errorf("panic: %w", err).Error()
			} else {
				out["error"] = fmt.Sprintf("panic: %v", r)
			}
		}
	}()
	v, err := f()
	if err != nil {
		out["error"] = err.Error()
		return out
	}
	out["value"] = v
	return out
}

// jsResolver wraps a JS resolver function. null/undefined -> no resolver.
// Contract: return a string (resolved, trusted verbatim) or null/
// undefined (declined). Throwing, or returning any other type, fails the
// render.
func jsResolver(fn js.Value) htmlrender.Resolver {
	if fn.IsUndefined() || fn.IsNull() {
		return nil
	}
	return func(kind htmlrender.ResolveKind, target string) (string, bool) {
		res := fn.Invoke(int(kind), target) // JS throw -> panic(js.Error) -> result() recovers
		if res.IsNull() || res.IsUndefined() {
			return "", false
		}
		if res.Type() != js.TypeString {
			panic(fmt.Errorf("resolver returned %s; want string or null", res.Type()))
		}
		return res.String(), true
	}
}

// optsArg returns the options JSON bytes from an optional string arg.
func optsArg(v js.Value) []byte {
	if v.IsUndefined() || v.IsNull() {
		return nil
	}
	return []byte(v.String())
}

func main() {
	api := map[string]any{
		"version": js.FuncOf(func(_ js.Value, _ []js.Value) any {
			return version
		}),
		// render(mdString, optsJSONString|null, resolverFn|null)
		"render": js.FuncOf(func(_ js.Value, args []js.Value) any {
			return result(func() (any, error) {
				b, err := boundary.Render([]byte(args[0].String()), optsArg(args[1]), jsResolver(args[2]))
				return string(b), err
			})
		}),
		// parse(mdString, optsJSONString|null) -> document JSON string
		"parse": js.FuncOf(func(_ js.Value, args []js.Value) any {
			return result(func() (any, error) {
				b, err := boundary.Parse([]byte(args[0].String()), optsArg(args[1]))
				return string(b), err
			})
		}),
		// renderDoc(docJSONString, optsJSONString|null, resolverFn|null)
		"renderDoc": js.FuncOf(func(_ js.Value, args []js.Value) any {
			return result(func() (any, error) {
				b, err := boundary.RenderDoc([]byte(args[0].String()), optsArg(args[1]), jsResolver(args[2]))
				return string(b), err
			})
		}),
		// asset(name) -> Uint8Array
		"asset": js.FuncOf(func(_ js.Value, args []js.Value) any {
			return result(func() (any, error) {
				b, err := boundary.Asset(args[0].String())
				if err != nil {
					return nil, err
				}
				u8 := js.Global().Get("Uint8Array").New(len(b))
				js.CopyBytesToJS(u8, b)
				return u8, nil
			})
		}),
	}
	js.Global().Set("__libmdviewer", js.ValueOf(api))
	if onready := js.Global().Get("__libmdviewer_onready"); onready.Type() == js.TypeFunction {
		onready.Invoke()
	}
	select {} // keep the Go runtime (and the js.FuncOf callbacks) alive
}
