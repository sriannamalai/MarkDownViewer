package htmlrender

import (
	"crypto/sha256"
	"fmt"
	"reflect"
	"strings"
	"sync"
	"testing"
)

// withFreshRunsCache swaps the package token-runs cache for a fresh one
// bounded at maxBytes for the duration of the test.
func withFreshRunsCache(t *testing.T, maxBytes int) *tokenRunsCache {
	t.Helper()
	old := runsCache
	fresh := newTokenRunsCache(maxBytes)
	runsCache = fresh
	t.Cleanup(func() { runsCache = old })
	return fresh
}

// TestTokenRunsPinnedGolden pins the exact runs — text slicing AND
// chroma's canonical TokenType.String() names — for one Go fence. If
// chroma renames token types or re-slices, this fails loudly: the names
// are a wire contract (highlight-*.json keys must keep matching them).
func TestTokenRunsPinnedGolden(t *testing.T) {
	withFreshRunsCache(t, runsCacheMaxBytes)
	runs, ok := TokenRuns("x := 42 // n\n", "go")
	if !ok {
		t.Fatal("TokenRuns failed for go")
	}
	want := []TokenRun{
		{Text: "x", TokenType: "NameOther"},
		{Text: " ", TokenType: "TextWhitespace"},
		{Text: ":=", TokenType: "Operator"},
		{Text: " ", TokenType: "TextWhitespace"},
		{Text: "42", TokenType: "LiteralNumberInteger"},
		{Text: " ", TokenType: "TextWhitespace"},
		{Text: "// n", TokenType: "CommentSingle"},
		{Text: "\n", TokenType: "TextWhitespace"},
	}
	if !reflect.DeepEqual(runs, want) {
		t.Fatalf("go runs:\n got %#v\nwant %#v", runs, want)
	}
}

// TestTokenRunsCanonicalNames pins a few canonical chroma type names
// across languages (empirically verified against chroma v2).
func TestTokenRunsCanonicalNames(t *testing.T) {
	withFreshRunsCache(t, runsCacheMaxBytes)
	cases := []struct {
		lang, code, text, tokenType string
	}{
		{"python", "def f():\n    return 'hi'\n", "def", "Keyword"},
		{"python", "def f():\n    return 'hi'\n", "'hi'", "LiteralStringSingle"},
		{"python", "def f():\n    return 'hi'\n", "f", "NameFunction"},
		{"go", "import \"fmt\"\n", "\"fmt\"", "LiteralString"},
		{"json", "{\"a\": true}\n", "true", "KeywordConstant"},
		{"json", "{\"a\": true}\n", "\"a\"", "NameTag"},
	}
	for _, tc := range cases {
		runs, ok := TokenRuns(tc.code, tc.lang)
		if !ok {
			t.Fatalf("TokenRuns(%q, %q) failed", tc.code, tc.lang)
		}
		found := false
		for _, r := range runs {
			if r.Text == tc.text && r.TokenType == tc.tokenType {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("%s: no run {%q %q} in %v", tc.lang, tc.text, tc.tokenType, runs)
		}
	}
}

// TestTokenRunsConcatenationProperty is the parity guarantee native
// renderers rely on: for every (lang, code) that yields runs, the
// concatenation of run texts equals the code byte-for-byte.
func TestTokenRunsConcatenationProperty(t *testing.T) {
	withFreshRunsCache(t, runsCacheMaxBytes)
	langs := []string{"go", "python", "javascript", "json", "c", "rust", "html", "shell", "text", "yaml", "sql"}
	samples := []string{
		"x := 1\n",
		"if a < b && c > d { panic(\"<oops> & such\") }\n",
		"# unicode: héllo wörld — ünïcode 世界 🚀\nprint('done')\n",
		"\tleading tab\n  trailing spaces  \n",
		"no trailing newline",
		"",
		"crlf line one\r\nline two\r\n",
		strings.Repeat("line := \"padding\" // filler\n", 40),
	}
	for _, lang := range langs {
		for si, code := range samples {
			runs, ok := TokenRuns(code, lang)
			if !ok {
				// Fallback-to-plain is always a legal outcome; what is
				// NOT legal is ok with a diverging concatenation.
				continue
			}
			var b strings.Builder
			for _, r := range runs {
				if r.Text == "" {
					t.Errorf("lang %q sample %d: empty run emitted", lang, si)
				}
				b.WriteString(r.Text)
			}
			if b.String() != code {
				t.Errorf("lang %q sample %d: concatenation diverges\n got %q\nwant %q", lang, si, b.String(), code)
			}
		}
	}
}

// TestTokenRunsUnknownLangNotCached: unknown/empty language reports
// !ok, mirrors the HTML path's fallback, and leaves no cache entry.
func TestTokenRunsUnknownLangNotCached(t *testing.T) {
	c := withFreshRunsCache(t, runsCacheMaxBytes)
	if _, ok := TokenRuns("zzz\n", "nosuchlang"); ok {
		t.Fatal("TokenRuns succeeded for unknown language")
	}
	if _, ok := TokenRuns("zzz\n", ""); ok {
		t.Fatal("TokenRuns succeeded for empty language")
	}
	if n := len(c.entries); n != 0 {
		t.Fatalf("failed lookups left %d cache entries", n)
	}
}

// TestTokenRunsEmptyCode: a known language with empty code yields ok
// with zero runs (wire: []), not null — the fence exists, it is just
// empty.
func TestTokenRunsEmptyCode(t *testing.T) {
	withFreshRunsCache(t, runsCacheMaxBytes)
	runs, ok := TokenRuns("", "go")
	if !ok || runs == nil || len(runs) != 0 {
		t.Fatalf("empty go fence: runs=%v ok=%v, want empty non-nil and ok", runs, ok)
	}
}

// TestTokenRunsCacheHitIdentical: a cache hit returns the same runs as
// the cold path, and hit vs miss is observable in entry count.
func TestTokenRunsCacheHitIdentical(t *testing.T) {
	c := withFreshRunsCache(t, runsCacheMaxBytes)
	code, lang := "x := 42 // n\n", "go"
	cold, ok := TokenRuns(code, lang)
	if !ok {
		t.Fatal("cold TokenRuns failed")
	}
	if n := len(c.entries); n != 1 {
		t.Fatalf("after cold call: %d entries, want 1", n)
	}
	warm, ok := TokenRuns(code, lang)
	if !ok {
		t.Fatal("warm TokenRuns failed")
	}
	if !reflect.DeepEqual(cold, warm) {
		t.Fatalf("cache hit diverges: %v vs %v", cold, warm)
	}
	if n := len(c.entries); n != 1 {
		t.Fatalf("after warm call: %d entries, want 1", n)
	}
	// Same code under a different language is a distinct entry (key
	// discrimination — {lang, sha256(code)}).
	if _, ok := TokenRuns(code, "python"); !ok {
		t.Fatal("python TokenRuns failed")
	}
	if n := len(c.entries); n != 2 {
		t.Fatalf("two languages: %d entries, want 2", n)
	}
}

// TestTokenRunsCacheEviction: the cache stays within its byte bound and
// evicts least-recently-used entries first.
func TestTokenRunsCacheEviction(t *testing.T) {
	probe, ok := TokenRuns("var v0 = 0;\n", "javascript")
	if !ok {
		t.Fatal("probe failed")
	}
	bound := 4 * (runsCost(highlightKey{lang: "javascript"}, probe) + 64)
	c := withFreshRunsCache(t, bound)

	code := func(i int) string { return fmt.Sprintf("var v%d = %d;\n", i, i) }
	key := func(i int) highlightKey {
		return highlightKey{lang: "javascript", sum: sha256.Sum256([]byte(code(i)))}
	}
	for i := 0; i < 20; i++ {
		if _, ok := TokenRuns(code(i), "javascript"); !ok {
			t.Fatalf("TokenRuns %d failed", i)
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

	// LRU, not FIFO: touch the oldest surviving entry, insert once more,
	// and the touched entry must survive the eviction.
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
	if _, ok := TokenRuns(code(20), "javascript"); !ok {
		t.Fatal("TokenRuns 20 failed")
	}
	if _, ok := c.entries[key(oldest)]; !ok {
		t.Fatal("recently-touched entry was evicted (FIFO, not LRU)")
	}
	if _, ok := c.entries[key(second)]; ok {
		t.Fatal("least-recently-used entry was not the one evicted")
	}
}

// TestTokenRunsCacheOversizeEntry: an entry bigger than the whole
// budget is served but never stored (and does not wipe the cache).
func TestTokenRunsCacheOversizeEntry(t *testing.T) {
	small, ok := TokenRuns("x := 1\n", "go")
	if !ok {
		t.Fatal("small probe failed")
	}
	c := withFreshRunsCache(t, 2*(runsCost(highlightKey{lang: "go"}, small)+64))
	if _, ok := TokenRuns("x := 1\n", "go"); !ok {
		t.Fatal("TokenRuns failed")
	}
	huge := strings.Repeat("v := \"padding\" // filler\n", 200)
	hugeRuns, ok := TokenRuns(huge, "go")
	if !ok {
		t.Fatal("oversize TokenRuns failed")
	}
	var b strings.Builder
	for _, r := range hugeRuns {
		b.WriteString(r.Text)
	}
	if b.String() != huge {
		t.Fatal("oversize runs diverge from code")
	}
	if n := len(c.entries); n != 1 {
		t.Fatalf("oversize entry disturbed the cache: %d entries", n)
	}
	if c.bytes > c.maxBytes {
		t.Fatalf("cache over budget: %d > %d", c.bytes, c.maxBytes)
	}
}

// TestConcurrentTokenRuns hammers the runs cache from many goroutines
// over a shared set of fences; under -race this gates the cache's
// concurrency safety, and every result must equal the sequential one.
func TestConcurrentTokenRuns(t *testing.T) {
	withFreshRunsCache(t, runsCacheMaxBytes)
	codes := make([]string, 8)
	wants := make([][]TokenRun, len(codes))
	for i := range codes {
		codes[i] = fmt.Sprintf("var v%d = %d; // fence\n", i, i)
	}
	for i, code := range codes {
		w, ok := TokenRuns(code, "javascript")
		if !ok {
			t.Fatalf("sequential TokenRuns %d failed", i)
		}
		wants[i] = w
	}
	withFreshRunsCache(t, runsCacheMaxBytes) // drop the warm entries
	const goroutines = 16
	errs := make([]error, goroutines)
	var wg sync.WaitGroup
	for g := 0; g < goroutines; g++ {
		wg.Add(1)
		go func(g int) {
			defer wg.Done()
			for i := 0; i < 5; i++ {
				for ci, code := range codes {
					runs, ok := TokenRuns(code, "javascript")
					if !ok {
						errs[g] = fmt.Errorf("TokenRuns %d failed", ci)
						return
					}
					if !reflect.DeepEqual(runs, wants[ci]) {
						errs[g] = fmt.Errorf("fence %d diverges from sequential runs", ci)
						return
					}
				}
			}
		}(g)
	}
	wg.Wait()
	for g, err := range errs {
		if err != nil {
			t.Fatalf("goroutine %d: %v", g, err)
		}
	}
}

// TestNormalizeRunsTrailingNewline pins the EnsureNL trim rule directly:
// an appended trailing newline is trimmed off (dropping a run that was
// only the newline), anything else fails closed.
func TestNormalizeRunsTrailingNewline(t *testing.T) {
	// Appended '\n' inside the final run's text.
	runs, ok := normalizeRuns([]TokenRun{{Text: "x", TokenType: "Text"}, {Text: "y\n", TokenType: "Text"}}, "xy\n", "xy")
	if !ok || len(runs) != 2 || runs[1].Text != "y" {
		t.Fatalf("trim within final run: %v ok=%v", runs, ok)
	}
	// Appended '\n' as its own run: the run is dropped.
	runs, ok = normalizeRuns([]TokenRun{{Text: "xy", TokenType: "Text"}, {Text: "\n", TokenType: "Text"}}, "xy\n", "xy")
	if !ok || len(runs) != 1 || runs[0].Text != "xy" {
		t.Fatalf("drop newline-only run: %v ok=%v", runs, ok)
	}
	// Any other divergence fails closed.
	if _, ok = normalizeRuns([]TokenRun{{Text: "a\nb", TokenType: "Text"}}, "a\nb", "a\r\nb"); ok {
		t.Fatal("CRLF rewrite must fail closed")
	}
}
