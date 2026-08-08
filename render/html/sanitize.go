package htmlrender

import "github.com/microcosm-cc/bluemonday"

// sanitizePolicy is the default policy for raw HTML embedded in markdown:
// bluemonday's UGC policy (basic formatting, links, images; no scripts,
// event handlers, or iframes).
var sanitizePolicy = bluemonday.UGCPolicy()
