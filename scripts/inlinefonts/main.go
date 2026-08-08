// Command inlinefonts rewrites KaTeX's CSS to carry its woff2 fonts as
// data: URIs so rendered pages are fully self-contained.
package main

import (
	"encoding/base64"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"regexp"
)

func main() {
	if len(os.Args) != 4 {
		log.Fatal("usage: inlinefonts <katex.min.css> <fontsdir> <out.css>")
	}
	css, err := os.ReadFile(os.Args[1])
	if err != nil {
		log.Fatal(err)
	}
	// Drop non-woff2 fallback sources.
	drop := regexp.MustCompile(`,\s*url\(fonts/[^)]+\.(woff|ttf)\)\s*format\((?:'|")?(?:woff|truetype)(?:'|")?\)`)
	out := drop.ReplaceAll(css, nil)
	// Inline woff2 references.
	woff2 := regexp.MustCompile(`url\(fonts/([^)]+\.woff2)\)`)
	out = woff2.ReplaceAllFunc(out, func(m []byte) []byte {
		name := woff2.FindSubmatch(m)[1]
		data, err := os.ReadFile(filepath.Join(os.Args[2], string(name)))
		if err != nil {
			log.Fatalf("font %s: %v", name, err)
		}
		return []byte(fmt.Sprintf("url(data:font/woff2;base64,%s)", base64.StdEncoding.EncodeToString(data)))
	})
	if err := os.WriteFile(os.Args[3], out, 0o644); err != nil {
		log.Fatal(err)
	}
}
