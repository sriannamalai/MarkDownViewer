package htmlrender

import (
	"container/list"
	"sync"
)

// highlightCacheMaxBytes bounds the package-level highlight cache. 8 MB of
// formatted HTML covers hundreds of typical fences — enough for re-renders
// of large code-heavy documents (theme flips, font-size steps, live
// preview) without growing unboundedly across documents.
const highlightCacheMaxBytes = 8 << 20

// highlightKey identifies a highlighted fence: the fence's language tag plus
// a digest of its code. The chroma style is deliberately absent — the HTML
// formatter always formats against the single package-fixed style in classes
// mode (see highlightStyleName in highlight.go), so style identity is a
// build-time constant. If the style ever becomes configurable per render,
// the key must grow a style component.
type highlightKey struct {
	lang string
	sum  [32]byte // sha256 of the fence's code
}

type highlightEntry struct {
	key  highlightKey
	html string
}

// highlightCache is a mutex-protected, size-bounded LRU of formatted
// highlight HTML. Only the formatted HTML string is cached — token streams
// are far larger than their rendered HTML and re-renders only need the HTML.
// The single mutex covers both lookup (which reorders the LRU list) and
// insert; highlighting even a small fence costs orders of magnitude more
// than the critical section, so contention is not a concern.
type highlightCache struct {
	mu       sync.Mutex
	maxBytes int
	bytes    int                            // sum of entryCost over entries
	order    *list.List                     // *highlightEntry; front = most recently used
	entries  map[highlightKey]*list.Element // into order
}

func newHighlightCache(maxBytes int) *highlightCache {
	return &highlightCache{
		maxBytes: maxBytes,
		order:    list.New(),
		entries:  map[highlightKey]*list.Element{},
	}
}

// hlCache is the package-level cache all renders share. Concurrent Render
// calls (the facade documents concurrent use) hit it under its mutex; the
// -race concurrency test in highlightcache_test.go gates this.
var hlCache = newHighlightCache(highlightCacheMaxBytes)

func (c *highlightCache) get(k highlightKey) (string, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	el, ok := c.entries[k]
	if !ok {
		return "", false
	}
	c.order.MoveToFront(el)
	return el.Value.(*highlightEntry).html, true
}

func (c *highlightCache) put(k highlightKey, html string) {
	cost := entryCost(k, html)
	if cost > c.maxBytes {
		// Larger than the entire budget: caching it would evict everything
		// and immediately exceed the bound. Serve it uncached.
		return
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if el, ok := c.entries[k]; ok {
		// Another renderer raced us to the insert; the HTML is identical
		// by construction (same key, deterministic formatting).
		c.order.MoveToFront(el)
		return
	}
	c.entries[k] = c.order.PushFront(&highlightEntry{key: k, html: html})
	c.bytes += cost
	for c.bytes > c.maxBytes {
		back := c.order.Back()
		e := back.Value.(*highlightEntry)
		c.order.Remove(back)
		delete(c.entries, e.key)
		c.bytes -= entryCost(e.key, e.html)
	}
}

// entryCost approximates an entry's memory footprint: key bytes plus the
// cached HTML.
func entryCost(k highlightKey, html string) int {
	return len(k.lang) + len(k.sum) + len(html)
}
