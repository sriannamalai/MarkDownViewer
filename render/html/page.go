package htmlrender

import (
	"fmt"
	"io"

	"github.com/sriannamalai/markdownviewer/document"
	"github.com/sriannamalai/markdownviewer/theme"
)

func renderPage(w io.Writer, doc *document.Document, opts Options) error {
	th, err := theme.Get(opts.ThemeName)
	if err != nil {
		return err
	}
	lightChroma, err := chromaCSS(theme.Light().ChromaStyle)
	if err != nil {
		return err
	}
	fmt.Fprint(w, "<!doctype html>\n<html>\n<head>\n<meta charset=\"utf-8\">\n")
	fmt.Fprint(w, "<meta name=\"viewport\" content=\"width=device-width, initial-scale=1\">\n<style>\n")
	if opts.ThemeName == "dark" {
		darkChroma, err := chromaCSS(theme.Dark().ChromaStyle)
		if err != nil {
			return err
		}
		fmt.Fprint(w, theme.Dark().CSS(":root")+"\n"+darkChroma)
	} else {
		fmt.Fprint(w, th.CSS(":root")+"\n"+lightChroma)
	}
	if opts.ThemeName == "auto" || opts.ThemeName == "" {
		darkChroma, err := chromaCSS(theme.Dark().ChromaStyle)
		if err != nil {
			return err
		}
		fmt.Fprint(w, "\n@media (prefers-color-scheme: dark){\n"+theme.Dark().CSS(":root")+"\n"+darkChroma+"}\n")
	}
	fmt.Fprint(w, theme.BaseCSS())
	fmt.Fprint(w, "</style>\n</head>\n<body class=\"markdown-body\">\n")
	if err := renderFragment(w, doc, opts); err != nil {
		return err
	}
	// JS assets (mermaid/KaTeX) are appended here by Task 13.
	fmt.Fprint(w, "</body>\n</html>\n")
	return nil
}
