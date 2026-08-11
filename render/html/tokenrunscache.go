package htmlrender

import (
	"container/list"
	"sync"
)

// runsCacheMaxBytes bounds the package-level token-runs cache. The
// store is deliberately SEPARATE from the HTML highlight cache (same
// {lang, sha256(code)} key space, different value type) so eviction
// pressure from one renderer never churns the other's entries. Half the
// HTML cache's 8 MB budget suffices because runs are lighter than the
// formatted HTML of the same fence — the code text and type names
// without any per-token markup around them — so 4 MB covers the same
// corpus of fences the HTML budget was sized for.
const runsCacheMaxBytes = 4 << 20

// runsEntry pairs a cache key with its token runs. The runs slice is
// handed out shared (never copied on get) — see the immutability note
// on TokenRuns.
type runsEntry struct {
	key  highlightKey
	runs []TokenRun
}

// tokenRunsCache is a mutex-protected, size-bounded LRU of token-run
// lists, mirroring highlightCache (highlightcache.go) — single mutex
// over lookup (which reorders the LRU list) and insert; tokenising even
// a small fence costs orders of magnitude more than the critical
// section, so contention is not a concern.
type tokenRunsCache struct {
	mu       sync.Mutex
	maxBytes int
	bytes    int                            // sum of runsCost over entries
	order    *list.List                     // *runsEntry; front = most recently used
	entries  map[highlightKey]*list.Element // into order
}

func newTokenRunsCache(maxBytes int) *tokenRunsCache {
	return &tokenRunsCache{
		maxBytes: maxBytes,
		order:    list.New(),
		entries:  map[highlightKey]*list.Element{},
	}
}

// runsCache is the package-level cache all TokenRuns calls share.
// Concurrent tree builds hit it under its mutex; the -race concurrency
// tests in tokenruns_test.go and render/tree gate this.
var runsCache = newTokenRunsCache(runsCacheMaxBytes)

func (c *tokenRunsCache) get(k highlightKey) ([]TokenRun, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	el, ok := c.entries[k]
	if !ok {
		return nil, false
	}
	c.order.MoveToFront(el)
	return el.Value.(*runsEntry).runs, true
}

func (c *tokenRunsCache) put(k highlightKey, runs []TokenRun) {
	cost := runsCost(k, runs)
	if cost > c.maxBytes {
		// Larger than the entire budget: caching it would evict
		// everything and immediately exceed the bound. Serve it uncached.
		return
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if el, ok := c.entries[k]; ok {
		// Another builder raced us to the insert; the runs are identical
		// by construction (same key, deterministic tokenisation).
		c.order.MoveToFront(el)
		return
	}
	c.entries[k] = c.order.PushFront(&runsEntry{key: k, runs: runs})
	c.bytes += cost
	for c.bytes > c.maxBytes {
		back := c.order.Back()
		e := back.Value.(*runsEntry)
		c.order.Remove(back)
		delete(c.entries, e.key)
		c.bytes -= runsCost(e.key, e.runs)
	}
}

// runStructOverhead approximates the fixed per-run memory beyond the
// string bytes themselves: two string headers (pointer + length each)
// on a 64-bit platform.
const runStructOverhead = 32

// runsCost approximates an entry's memory footprint: the key bytes plus
// every run's text and type-name bytes plus per-run struct overhead.
func runsCost(k highlightKey, runs []TokenRun) int {
	cost := len(k.lang) + len(k.sum)
	for _, r := range runs {
		cost += len(r.Text) + len(r.TokenType) + runStructOverhead
	}
	return cost
}
