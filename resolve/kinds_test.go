package resolve

import "testing"

// The FFI and WASM boundaries expose ResolveKind as bare ints. These
// values are ABI-frozen: 0=link, 1=image, 2=wiki-link, append-only
// forever (like document Kind values). If this test fails, the ABI has
// been broken — fix the constants, never this test.
// internal/boundary/kinds_test.go pins the same values as seen through
// the render/html re-export chain.
func TestResolveKindValuesAreABIFrozen(t *testing.T) {
	if ResolveLink != 0 || ResolveImage != 1 || ResolveWikiLink != 2 {
		t.Fatalf("ResolveKind values moved: link=%d image=%d wiki=%d (must be 0/1/2)",
			ResolveLink, ResolveImage, ResolveWikiLink)
	}
}
