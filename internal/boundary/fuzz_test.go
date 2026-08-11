package boundary

import (
	"bytes"
	"testing"
)

// FuzzRenderDocJSON feeds arbitrary bytes as document JSON into RenderDoc
// in fragment mode. Two properties must hold no matter the input:
//
//  1. no panic escapes the boundary (RenderDoc returns an error instead), and
//  2. the output never contains "<script" in any case — fragment mode emits
//     no library-authored scripts (those exist only in full-page chrome,
//     render/html/page.go), text/attribute content is HTML-escaped, and raw
//     HTML nodes pass through bluemonday's UGC policy which strips script
//     tags. Any occurrence therefore means hostile input bytes reached the
//     output unescaped.
func FuzzRenderDocJSON(f *testing.F) {
	// Seed: a valid Parse() output for a document exercising several kinds.
	valid, err := Parse([]byte("# T\n\n> [!note]\n> hi\n\n- [x] task\n\n```go\ncode\n```\n"), nil)
	if err != nil {
		f.Fatal(err)
	}
	f.Add(valid)
	// Seed: the XSS reproducer (attribute-breakout admonition variant).
	f.Add([]byte(hostileAdmonitionDocJSON))
	// Seeds: truncated and otherwise malformed variants.
	f.Add([]byte(hostileAdmonitionDocJSON)[:len(hostileAdmonitionDocJSON)/2])
	f.Add([]byte(`{`))
	f.Add([]byte(``))
	f.Add([]byte(`{"version":1,"kind":"document"}`))
	f.Add([]byte(`{"version":1,"kind":"document","children":[{"kind":"admonition","variant":""}]}`))
	f.Add([]byte(`{"version":1,"kind":"document","children":[{"kind":"nosuchkind"}]}`))
	f.Add([]byte(`{"version":1,"kind":"document","children":[{"kind":"html","html":"<script>alert(1)</script>"}]}`))
	f.Add([]byte(`{"version":2,"kind":"document"}`))

	opts := []byte(`{"fragment": true}`)
	f.Fuzz(func(t *testing.T, docJSON []byte) {
		out, err := RenderDoc(docJSON, opts, nil)
		if err != nil {
			return // rejecting hostile input with an error is fine
		}
		if bytes.Contains(bytes.ToLower(out), []byte("<script")) {
			t.Fatalf("fragment render of arbitrary doc JSON emitted <script\ninput: %q\noutput: %s", docJSON, out)
		}
	})
}
