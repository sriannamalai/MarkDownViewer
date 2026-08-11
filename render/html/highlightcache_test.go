package htmlrender

import (
	"bytes"
	"crypto/sha256"
	"fmt"
	"strings"
	"sync"
	"testing"

	"github.com/sriannamalai/markdownviewer/parser"
)

// withFreshHighlightCache swaps the package highlight cache for a fresh one
// bounded at maxBytes for the duration of the test.
func withFreshHighlightCache(t *testing.T, maxBytes int) *highlightCache {
	t.Helper()
	old := hlCache
	fresh := newHighlightCache(maxBytes)
	hlCache = fresh
	t.Cleanup(func() { hlCache = old })
	return fresh
}

// uncachedHighlightHTML computes the highlighted HTML directly via the
// tokenise/format split, bypassing the cache entirely.
func uncachedHighlightHTML(t *testing.T, code, lang string) string {
	t.Helper()
	it := tokenise(code, lang)
	if it == nil {
		t.Fatalf("tokenise failed for lang %q", lang)
	}
	html, ok := formatTokens(it)
	if !ok {
		t.Fatalf("formatTokens failed for lang %q", lang)
	}
	return html
}

// TestHighlightCachedByteIdentical is the core correctness property: for a
// spread of fences, the cached path (miss then hit) must be byte-identical
// to a cache-bypassed tokenise+format.
func TestHighlightCachedByteIdentical(t *testing.T) {
	withFreshHighlightCache(t, highlightCacheMaxBytes)
	langs := []string{"go", "python", "javascript", "json", "c", "rust", "html"}
	samples := []string{
		"x := 1\n",
		"if a < b && c > d { panic(\"<oops> & such\") }\n",
		"# unicode: héllo wörld — ünïcode 世界 🚀\nprint('done')\n",
		"\tleading tab\n  trailing spaces  \n",
		"no trailing newline",
		"",
		strings.Repeat("line := \"padding\" // filler\n", 40),
	}
	for _, lang := range langs {
		for si, code := range samples {
			want := uncachedHighlightHTML(t, code, lang)
			var first, second strings.Builder
			if !highlight(&first, code, lang) {
				t.Fatalf("lang %q sample %d: highlight (cold) failed", lang, si)
			}
			if !highlight(&second, code, lang) {
				t.Fatalf("lang %q sample %d: highlight (cached) failed", lang, si)
			}
			if first.String() != want {
				t.Errorf("lang %q sample %d: cold path diverges from uncached\n got %q\nwant %q", lang, si, first.String(), want)
			}
			if second.String() != want {
				t.Errorf("lang %q sample %d: cached path diverges from uncached\n got %q\nwant %q", lang, si, second.String(), want)
			}
		}
	}
}

// TestHighlightUnknownLangNotCached: fallback semantics are unchanged and
// failed lookups leave no cache entry behind.
func TestHighlightUnknownLangNotCached(t *testing.T) {
	c := withFreshHighlightCache(t, highlightCacheMaxBytes)
	var b strings.Builder
	if highlight(&b, "zzz\n", "nosuchlang") {
		t.Fatal("highlight succeeded for unknown language")
	}
	if highlight(&b, "zzz\n", "") {
		t.Fatal("highlight succeeded for empty language")
	}
	if n := len(c.entries); n != 0 {
		t.Fatalf("failed highlights left %d cache entries", n)
	}
}

// TestHighlightCacheKeyDiscrimination: same code under different languages
// and different code under the same language are distinct entries.
func TestHighlightCacheKeyDiscrimination(t *testing.T) {
	c := withFreshHighlightCache(t, highlightCacheMaxBytes)
	code := "x = 1\n"
	var b strings.Builder
	if !highlight(&b, code, "go") || !highlight(&b, code, "python") {
		t.Fatal("highlight failed")
	}
	if n := len(c.entries); n != 2 {
		t.Fatalf("same code, two languages: want 2 entries, got %d", n)
	}
	if !highlight(&b, "y = 2\n", "go") {
		t.Fatal("highlight failed")
	}
	if n := len(c.entries); n != 3 {
		t.Fatalf("two codes under go + one under python: want 3 entries, got %d", n)
	}
	// The two languages must not share output: go and python tokenise
	// "x = 1" differently.
	goHTML, _ := c.get(highlightKey{lang: "go", sum: sha256.Sum256([]byte(code))})
	pyHTML, _ := c.get(highlightKey{lang: "python", sum: sha256.Sum256([]byte(code))})
	if goHTML == "" || pyHTML == "" {
		t.Fatal("expected both entries retrievable")
	}
	if goHTML == pyHTML {
		t.Fatalf("go and python produced identical HTML for %q — key discrimination untestable", code)
	}
}

// TestHighlightCacheEviction: the cache stays within its byte bound and
// evicts least-recently-used entries first.
func TestHighlightCacheEviction(t *testing.T) {
	// Size the bound to hold only a few entries.
	probe := uncachedHighlightHTML(t, "var v0 = 0;\n", "javascript")
	bound := 4 * (len(probe) + 64)
	c := withFreshHighlightCache(t, bound)

	code := func(i int) string { return fmt.Sprintf("var v%d = %d;\n", i, i) }
	key := func(i int) highlightKey {
		return highlightKey{lang: "javascript", sum: sha256.Sum256([]byte(code(i)))}
	}
	var b strings.Builder
	for i := 0; i < 20; i++ {
		if !highlight(&b, code(i), "javascript") {
			t.Fatalf("highlight %d failed", i)
		}
		if c.bytes > bound {
			t.Fatalf("after insert %d: cache holds %d bytes, bound %d", i, c.bytes, bound)
		}
	}
	if _, ok := c.get(key(0)); ok {
		t.Fatal("oldest entry survived 20 inserts into a ~4-entry cache")
	}
	if _, ok := c.get(key(19)); !ok {
		t.Fatal("newest entry missing")
	}

	// LRU, not FIFO: touch the oldest surviving entry, insert until one
	// eviction happens, and the touched entry must survive it.
	var oldest, second = -1, -1
	for i := 0; i < 20; i++ {
		if _, ok := c.entries[key(i)]; ok {
			if oldest < 0 {
				oldest = i
			} else if second < 0 {
				second = i
			}
		}
	}
	if oldest < 0 || second < 0 {
		t.Fatal("expected at least two surviving entries")
	}
	if _, ok := c.get(key(oldest)); !ok { // touch → most recently used
		t.Fatal("touch failed")
	}
	// One more insert must push past the bound and evict at least one
	// entry — the LRU one, which after the touch is `second`, not the
	// touched `oldest`.
	if !highlight(&b, code(20), "javascript") {
		t.Fatal("highlight 20 failed")
	}
	if _, ok := c.entries[key(oldest)]; !ok {
		t.Fatal("recently-touched entry was evicted (FIFO, not LRU)")
	}
	if _, ok := c.entries[key(second)]; ok {
		t.Fatal("least-recently-used entry was not the one evicted")
	}
}

// TestHighlightCacheOversizeEntry: an entry bigger than the whole budget is
// served but never stored (and does not wipe the cache).
func TestHighlightCacheOversizeEntry(t *testing.T) {
	small := uncachedHighlightHTML(t, "x := 1\n", "go")
	c := withFreshHighlightCache(t, 2*(len(small)+64))
	var b strings.Builder
	if !highlight(&b, "x := 1\n", "go") {
		t.Fatal("highlight failed")
	}
	huge := strings.Repeat("v := \"padding\" // filler\n", 200)
	var h strings.Builder
	if !highlight(&h, huge, "go") {
		t.Fatal("oversize highlight failed")
	}
	if h.String() != uncachedHighlightHTML(t, huge, "go") {
		t.Fatal("oversize output diverges from uncached")
	}
	if n := len(c.entries); n != 1 {
		t.Fatalf("oversize entry disturbed the cache: %d entries", n)
	}
	if c.bytes > c.maxBytes {
		t.Fatalf("cache over budget: %d > %d", c.bytes, c.maxBytes)
	}
}

// TestConcurrentHighlightRenders renders the same code-heavy document from
// many goroutines; under -race this gates the cache's concurrency safety,
// and every output must be byte-identical to a sequential render.
func TestConcurrentHighlightRenders(t *testing.T) {
	withFreshHighlightCache(t, highlightCacheMaxBytes)
	src := string(codeHeavySource(20))
	want := render(t, src, func(o *Options) { o.Highlight = true })
	doc, err := parser.Parse([]byte(src))
	if err != nil {
		t.Fatal(err)
	}
	opts := DefaultOptions()
	opts.Fragment = true
	opts.Highlight = true
	const goroutines = 16
	outs := make([]string, goroutines)
	errs := make([]error, goroutines)
	var wg sync.WaitGroup
	for g := 0; g < goroutines; g++ {
		wg.Add(1)
		go func(g int) {
			defer wg.Done()
			for i := 0; i < 5; i++ {
				var buf bytes.Buffer
				if err := Render(&buf, doc, opts); err != nil {
					errs[g] = err
					return
				}
				outs[g] = buf.String()
			}
		}(g)
	}
	wg.Wait()
	for g := range outs {
		if errs[g] != nil {
			t.Fatalf("goroutine %d: %v", g, errs[g])
		}
		if outs[g] != want {
			t.Fatalf("goroutine %d output diverges from sequential render", g)
		}
	}
}
