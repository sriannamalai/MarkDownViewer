package boundary

import (
	"testing"

	htmlrender "github.com/sriannamalai/markdownviewer/render/html"
)

// The FFI and WASM boundaries expose ResolveKind as bare ints. These
// values are ABI-frozen: 0=link, 1=image, 2=wiki-link, append-only
// forever (like document Kind values). If this test fails, the ABI has
// been broken — fix the constants, never this test.
func TestResolveKindValuesAreABIFrozen(t *testing.T) {
	if htmlrender.ResolveLink != 0 || htmlrender.ResolveImage != 1 || htmlrender.ResolveWikiLink != 2 {
		t.Fatalf("ResolveKind values moved: link=%d image=%d wiki=%d (must be 0/1/2)",
			htmlrender.ResolveLink, htmlrender.ResolveImage, htmlrender.ResolveWikiLink)
	}
}
