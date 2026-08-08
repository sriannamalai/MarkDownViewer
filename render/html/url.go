package htmlrender

import "strings"

var badSchemes = []string{"javascript:", "vbscript:", "data:", "file:"}

// safeURL reports whether a URL is allowed under the default policy.
// Relative URLs and http(s)/mailto pass; script-ish schemes do not.
func safeURL(u string) bool {
	s := strings.ToLower(strings.TrimSpace(u))
	for _, p := range badSchemes {
		if strings.HasPrefix(s, p) {
			return false
		}
	}
	return true
}
