package main

/*
#include <stdlib.h>
#include <string.h>
*/
import "C"

import (
	"errors"
	"fmt"
	"math"
	"unsafe"
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
		return renderImpl(src, goOpts(optsJSON))
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
		return parseImpl(src, goOpts(optsJSON))
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
		return renderDocImpl(doc, goOpts(optsJSON))
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
			return assetImpl("")
		}
		return assetImpl(C.GoString(name))
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
