package htmlrender

import (
	"fmt"
	"io"
	"sort"
	"strings"

	"github.com/sriannamalai/markdownviewer/document"
	"github.com/sriannamalai/markdownviewer/internal/assets"
	"github.com/sriannamalai/markdownviewer/theme"
)

// usesFeatures reports whether doc contains any diagram or math nodes, so
// the page only pays for mermaid/KaTeX assets when it actually needs them.
func usesFeatures(doc *document.Document) (mermaid, math bool) {
	document.Walk(doc, func(n document.Node, entering bool) document.WalkStatus {
		if !entering {
			return document.Continue
		}
		switch n.(type) {
		case *document.Diagram:
			mermaid = true
		case *document.MathBlock, *document.MathInline:
			math = true
		}
		return document.Continue
	})
	return mermaid, math
}

// emitThemeOverrides emits CSS custom-property overrides in sorted key order.
func emitThemeOverrides(overrides map[string]string) string {
	if len(overrides) == 0 {
		return ""
	}
	keys := make([]string, 0, len(overrides))
	for k := range overrides {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	var b strings.Builder
	b.WriteString(":root{")
	for _, k := range keys {
		b.WriteString(k + ":" + overrides[k] + ";")
	}
	b.WriteString("}")
	return b.String()
}

func renderPage(w io.Writer, doc *document.Document, opts Options) error {
	th, err := theme.Get(opts.ThemeName)
	if err != nil {
		return err
	}
	lightChroma, err := chromaCSS(theme.Light().ChromaStyle)
	if err != nil {
		return err
	}
	mermaidUsed, mathUsed := usesFeatures(doc)
	fmt.Fprint(w, "<!doctype html>\n<html>\n<head>\n<meta charset=\"utf-8\">\n")
	fmt.Fprint(w, "<meta name=\"viewport\" content=\"width=device-width, initial-scale=1\">\n<style>\n")
	if opts.ThemeName == "dark" {
		darkChroma, err := chromaCSS(theme.Dark().ChromaStyle)
		if err != nil {
			return err
		}
		fmt.Fprint(w, theme.Dark().CSS(":root")+"\n"+darkChroma)
		if overrides := emitThemeOverrides(opts.ThemeOverrides); overrides != "" {
			fmt.Fprint(w, "\n"+overrides)
		}
	} else {
		fmt.Fprint(w, th.CSS(":root")+"\n"+lightChroma)
		if overrides := emitThemeOverrides(opts.ThemeOverrides); overrides != "" {
			fmt.Fprint(w, "\n"+overrides)
		}
	}
	if opts.ThemeName == "auto" || opts.ThemeName == "" {
		darkChroma, err := chromaCSS(theme.Dark().ChromaStyle)
		if err != nil {
			return err
		}
		fmt.Fprint(w, "\n@media (prefers-color-scheme: dark){\n"+theme.Dark().CSS(":root")+"\n"+darkChroma)
		if overrides := emitThemeOverrides(opts.ThemeOverrides); overrides != "" {
			fmt.Fprint(w, "\n"+overrides)
		}
		fmt.Fprint(w, "}\n")
	}
	if opts.Stylesheet != "" {
		fmt.Fprint(w, opts.Stylesheet)
	} else {
		fmt.Fprint(w, theme.BaseCSS())
	}
	if mathUsed && opts.Math {
		fmt.Fprint(w, assets.KatexCSS())
	}
	fmt.Fprint(w, "</style>\n</head>\n<body class=\"markdown-body\">\n")
	if err := renderFragment(w, doc, opts); err != nil {
		return err
	}
	if mermaidUsed && opts.Mermaid {
		mtheme := "default"
		if opts.ThemeName == "dark" {
			mtheme = "dark"
		}
		fmt.Fprint(w, "<script>"+assets.MermaidJS()+"</script>\n")
		fmt.Fprintf(w, "<script>mermaid.initialize({startOnLoad:true,theme:%q});</script>\n", mtheme)
	}
	if mathUsed && opts.Math {
		fmt.Fprint(w, "<script>"+assets.KatexJS()+"</script>\n")
		fmt.Fprint(w, "<script>document.querySelectorAll('.math').forEach(function(el){"+
			"katex.render(el.textContent,el,{displayMode:el.classList.contains('math-display'),throwOnError:false});});</script>\n")
	}
	fmt.Fprint(w, "</body>\n</html>\n")
	return nil
}
