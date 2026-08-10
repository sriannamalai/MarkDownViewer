package main

/*
#include <stdlib.h>
#include <string.h>

// mdv_resolver_fn is the host-supplied link/image/wiki-link resolution
// callback. kind: 0=link, 1=image, 2=wiki-link (ABI-frozen). target is
// NOT NUL-terminated and is only valid during the call. Return 1 =
// resolved (*out_url allocated with mdv_alloc, *out_url_len set; the
// library copies and frees it), 0 = declined (default resolution
// applies). Any other return value fails the render.
typedef int (*mdv_resolver_fn)(int kind, const char* target,
                               size_t target_len, void* userdata,
                               char** out_url, size_t* out_url_len);

// cgo cannot call C function pointers directly; this bridge does.
static int mdv_call_resolver(mdv_resolver_fn f, int kind,
                             const char* target, size_t target_len,
                             void* userdata,
                             char** out_url, size_t* out_url_len) {
	return f(kind, target, target_len, userdata, out_url, out_url_len);
}

// Plain malloc, NOT C.malloc from Go: cgo's C.malloc aborts the process
// on failure instead of returning NULL, and mdv_alloc must be able to
// report failure to the host honestly.
static void* mdv_malloc_raw(size_t n) { return malloc(n); }
*/
import "C"

import (
	"errors"
	"fmt"
	"math"
	"unsafe"

	"github.com/sriannamalai/markdownviewer/internal/boundary"
	htmlrender "github.com/sriannamalai/markdownviewer/render/html"
)

// version is injected at build time via -ldflags "-X main.version=…".
var version = "dev"

var cVersion = C.CString(version)

// goInput copies an FFI input buffer into Go memory. A nil pointer with a
// zero length is valid empty input; nil with a non-zero length is an error.
func goInput(p *C.char, n C.size_t) ([]byte, error) {
	if p == nil {
		if n != 0 {
			return nil, errors.New("nil input pointer with non-zero length")
		}
		return nil, nil
	}
	if n > C.size_t(math.MaxInt) {
		return nil, errors.New("input length exceeds maximum")
	}
	src := unsafe.Slice((*byte)(unsafe.Pointer(p)), int(n))
	out := make([]byte, len(src))
	copy(out, src)
	return out, nil
}

// panicError renders a recovered panic value as an FFI error.
func panicError(r any) error {
	if err, ok := r.(error); ok {
		return fmt.Errorf("panic: %w", err)
	}
	return fmt.Errorf("panic: %v", r)
}

// goOpts copies the NUL-terminated options string (may be NULL).
func goOpts(p *C.char) []byte {
	if p == nil {
		return nil
	}
	return []byte(C.GoString(p))
}

// cBuf copies b into a C-malloc'd buffer with a trailing NUL (not counted
// in the returned length). The caller owns the buffer (mdv_free). The nil
// check is defense-in-depth only: cgo's malloc wrapper never returns NULL —
// on true allocation failure it aborts the process with an unrecoverable
// runtime throw (not a panic), so no error path can observe it.
func cBuf(b []byte) (*C.char, C.size_t, error) {
	n := len(b)
	p := C.malloc(C.size_t(n + 1))
	if p == nil {
		return nil, 0, errors.New("out of memory")
	}
	if n > 0 {
		C.memcpy(p, unsafe.Pointer(&b[0]), C.size_t(n))
	}
	*(*byte)(unsafe.Add(p, n)) = 0
	return (*C.char)(p), C.size_t(n), nil
}

// call runs f and marshals its result or error into the out-parameters.
// Returns 0 on success, 1 on error. A Go panic anywhere below this point
// is converted into an error result so it cannot unwind through the
// exported entry point and kill the host. This does not cover C
// allocation failure: cgo's malloc wrapper aborts the process via an
// unrecoverable runtime throw, not a panic.
func call(out **C.char, outLen *C.size_t, outErr **C.char, f func() ([]byte, error)) (code C.int) {
	if outErr != nil {
		*outErr = nil
	}
	if out == nil || outLen == nil {
		if outErr != nil {
			*outErr = C.CString("nil output pointer")
		}
		return 1
	}
	*out, *outLen = nil, 0
	defer func() {
		if r := recover(); r != nil {
			*out, *outLen = nil, 0
			if outErr != nil {
				*outErr = C.CString(panicError(r).Error())
			}
			code = 1
		}
	}()
	res, err := f()
	if err != nil {
		if outErr != nil {
			*outErr = C.CString(err.Error())
		}
		return 1
	}
	buf, bufLen, err := cBuf(res)
	if err != nil {
		if outErr != nil {
			*outErr = C.CString(err.Error())
		}
		return 1
	}
	*out, *outLen = buf, bufLen
	return 0
}

// mdv_render renders markdown (md, md_len) to HTML per the version-1
// options JSON (opts_json, NUL-terminated, may be NULL for defaults).
// On success returns 0 and sets *out_html/*out_len (caller frees *out_html
// with mdv_free; the buffer has an uncounted trailing NUL). On error
// returns nonzero and sets *out_err (caller frees with mdv_free).
//
//export mdv_render
func mdv_render(md *C.char, mdLen C.size_t, optsJSON *C.char, outHTML **C.char, outLen *C.size_t, outErr **C.char) C.int {
	return call(outHTML, outLen, outErr, func() ([]byte, error) {
		src, err := goInput(md, mdLen)
		if err != nil {
			return nil, err
		}
		return boundary.Render(src, goOpts(optsJSON), nil)
	})
}

// mdv_parse parses markdown to version-1 document JSON. Same conventions
// as mdv_render.
//
//export mdv_parse
func mdv_parse(md *C.char, mdLen C.size_t, optsJSON *C.char, outJSON **C.char, outLen *C.size_t, outErr **C.char) C.int {
	return call(outJSON, outLen, outErr, func() ([]byte, error) {
		src, err := goInput(md, mdLen)
		if err != nil {
			return nil, err
		}
		return boundary.Parse(src, goOpts(optsJSON))
	})
}

// mdv_render_doc renders version-1 document JSON (as produced by
// mdv_parse) to HTML. Same conventions as mdv_render.
//
//export mdv_render_doc
func mdv_render_doc(docJSON *C.char, jsonLen C.size_t, optsJSON *C.char, outHTML **C.char, outLen *C.size_t, outErr **C.char) C.int {
	return call(outHTML, outLen, outErr, func() ([]byte, error) {
		doc, err := goInput(docJSON, jsonLen)
		if err != nil {
			return nil, err
		}
		return boundary.RenderDoc(doc, goOpts(optsJSON), nil)
	})
}

// mdv_alloc allocates n bytes on the library's heap. Use it for every
// buffer the host hands to the library (resolver out_url); the library
// frees such buffers itself. mdv_alloc(0) returns a valid non-NULL
// pointer. Returns NULL on allocation failure. Buffers the library
// hands to the host are still freed with mdv_free; mdv_alloc/mdv_free
// are the same allocator, which is the point: on Windows the host CRT's
// heap and the library's heap may differ, so cross-boundary ownership
// transfer must go through this pair.
//
//export mdv_alloc
func mdv_alloc(n C.size_t) unsafe.Pointer {
	if n == 0 {
		n = 1
	}
	return C.mdv_malloc_raw(n)
}

// mdv_render_r is mdv_render plus a host resolver callback. resolver
// may be NULL (identical to mdv_render). The callback runs
// synchronously on the calling thread during render; it must not
// unwind (longjmp, C++ exceptions) across the boundary. userdata is
// passed through untouched. See mdv_resolver_fn for the contract; a
// contract violation (return code other than 0/1, or 1 with NULL
// *out_url) fails the render with a descriptive error.
//
//export mdv_render_r
func mdv_render_r(md *C.char, mdLen C.size_t, optsJSON *C.char, resolver C.mdv_resolver_fn, userdata unsafe.Pointer, outHTML **C.char, outLen *C.size_t, outErr **C.char) C.int {
	return call(outHTML, outLen, outErr, func() ([]byte, error) {
		src, err := goInput(md, mdLen)
		if err != nil {
			return nil, err
		}
		return boundary.Render(src, goOpts(optsJSON), cResolver(resolver, userdata))
	})
}

// mdv_render_doc_r is mdv_render_doc plus a host resolver callback.
// Same conventions as mdv_render_r.
//
//export mdv_render_doc_r
func mdv_render_doc_r(docJSON *C.char, jsonLen C.size_t, optsJSON *C.char, resolver C.mdv_resolver_fn, userdata unsafe.Pointer, outHTML **C.char, outLen *C.size_t, outErr **C.char) C.int {
	return call(outHTML, outLen, outErr, func() ([]byte, error) {
		doc, err := goInput(docJSON, jsonLen)
		if err != nil {
			return nil, err
		}
		return boundary.RenderDoc(doc, goOpts(optsJSON), cResolver(resolver, userdata))
	})
}

// mdv_asset writes a copy of the embedded static asset registered under
// name (NUL-terminated; e.g. "mermaid.js", "katex.css",
// "theme-dark.css" — see the packaged README for the full registry)
// into the out-parameters. Same conventions as mdv_render: 0 on
// success, caller frees *out with mdv_free; nonzero + *out_err on
// error (unknown names list the valid ones).
//
//export mdv_asset
func mdv_asset(name *C.char, out **C.char, outLen *C.size_t, outErr **C.char) C.int {
	return call(out, outLen, outErr, func() ([]byte, error) {
		if name == nil {
			return boundary.Asset("")
		}
		return boundary.Asset(C.GoString(name))
	})
}

// mdv_free frees any buffer returned by this library. mdv_free(NULL) is a
// no-op. Do not free the mdv_version string.
//
//export mdv_free
func mdv_free(p *C.char) {
	C.free(unsafe.Pointer(p))
}

// mdv_version returns the library version as a static NUL-terminated
// string. The caller must NOT free it.
//
//export mdv_version
func mdv_version() *C.char {
	return cVersion
}

// cResolver wraps a host C callback as an htmlrender.Resolver. A nil fn
// yields a nil Resolver (default resolution). Contract violations panic;
// the panic is converted to an FFI error by the call envelope in mdv.go,
// so the host sees a failed render, never a crash.
func cResolver(fn C.mdv_resolver_fn, userdata unsafe.Pointer) htmlrender.Resolver {
	if fn == nil {
		return nil
	}
	return func(kind htmlrender.ResolveKind, target string) (string, bool) {
		ctarget := C.CString(target)
		defer C.free(unsafe.Pointer(ctarget))
		var outURL *C.char
		var outLen C.size_t
		rc := C.mdv_call_resolver(fn, C.int(kind), ctarget,
			C.size_t(len(target)), userdata, &outURL, &outLen)
		switch rc {
		case 0:
			return "", false
		case 1:
			if outURL == nil {
				panic(fmt.Errorf("resolver contract violation: returned 1 with NULL out_url"))
			}
			if outLen > C.size_t(math.MaxInt) {
				C.free(unsafe.Pointer(outURL))
				panic(fmt.Errorf("resolver contract violation: out_url_len exceeds maximum"))
			}
			b := unsafe.Slice((*byte)(unsafe.Pointer(outURL)), int(outLen))
			u := string(b) // copy before freeing
			C.free(unsafe.Pointer(outURL))
			return u, true
		default:
			// No ownership transfer happened: out_url is only the
			// library's to free when the callback returns 1. Freeing an
			// arbitrary pointer a misbehaving host left here could be an
			// invalid free (static/stack/foreign memory) — worse than the
			// leak, which belongs to the host that violated the contract.
			panic(fmt.Errorf("resolver contract violation: invalid return code %d (want 0 or 1)", int(rc)))
		}
	}
}
