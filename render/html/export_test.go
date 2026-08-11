package htmlrender

// resetHighlightCacheForTest empties the package highlight cache so a test
// or benchmark can measure/exercise the cold (uncached) path.
func resetHighlightCacheForTest() {
	hlCache = newHighlightCache(highlightCacheMaxBytes)
}
